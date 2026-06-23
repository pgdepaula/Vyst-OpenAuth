// Package service contains application services that orchestrate domain logic.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"github.com/pgdepaula/vyst-openauth/internal/domain/session"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

// ErrInvalidCredentials is returned when login credentials are incorrect..
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrUserNotActive is returned when a user tries to login but is not active.
var ErrUserNotActive = errors.New("user is not active")

// TokenPair contains the access token and optional refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// AuthService handles authentication operations (login, token generation).
type AuthService struct {
	userRepo        user.Repository
	policyRepo      policy.Repository
	companyUserRepo company.CompanyUserRepository // Injected to fetch roles
	sessionRepo     session.Repository
	hasher          ports.PasswordHasher
	token           ports.TokenService
	notifier        ports.NotificationService
	logger          ports.Logger
}

// NewAuthService creates a new authentication service.
func NewAuthService(
	userRepo user.Repository,
	policyRepo policy.Repository,
	companyUserRepo company.CompanyUserRepository,
	sessionRepo session.Repository,
	hasher ports.PasswordHasher,
	token ports.TokenService,
	notifier ports.NotificationService,
	logger ports.Logger,
) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		policyRepo:      policyRepo,
		companyUserRepo: companyUserRepo,
		sessionRepo:     sessionRepo,
		hasher:          hasher,
		token:           token,
		notifier:        notifier,
		logger:          logger,
	}
}

// Login authenticates a user and returns a token pair.
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	result, err := s.LoginWithUser(ctx, email, password)
	if err != nil {
		return nil, err
	}
	return result.Token, nil
}

// LoginResult contains the token pair and user info for 2FA flow.
type LoginResult struct {
	Token *TokenPair
	User  *user.User
}

// LoginWithUser authenticates a user and returns both token and user info.
func (s *AuthService) LoginWithUser(ctx context.Context, email, password string) (*LoginResult, error) {
	// Use request-scoped logger if available
	logger := s.logger.WithContext(ctx)
	logger.Info("Attempting login", "email", email)

	// 1. Get user by email
	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			logger.Warn("Login failed: user not found", "email", email)
			return nil, ErrInvalidCredentials
		}
		logger.Error("Login failed: database error", "error", err)
		return nil, err
	}
	// Also check if user is nil (some repos might return nil, nil for not found)
	if u == nil {
		logger.Warn("Login failed: user not found (nil)", "email", email)
		return nil, ErrInvalidCredentials
	}

	// 2. Verify password
	if !s.hasher.Verify(password, u.PasswordHash) {
		logger.Warn("Login failed: invalid password", "user_id", u.ID)
		return nil, ErrInvalidCredentials
	}

	// 3. Check status
	if u.Status != user.StatusActive {
		return nil, ErrUserNotActive
	}

	// 4. Generate token
	roles, err := s.policyRepo.GetRolesForUser(ctx, u.ID)
	if err != nil {
		roles = []string{}
	}

	var companyRole string
	if u.ActiveCompanyID != "" {
		cr, err := s.companyUserRepo.GetUserRole(ctx, u.ActiveCompanyID, u.ID)
		if err == nil {
			companyRole = cr.String()
		} else {
			// If error fetching role, maybe user was removed?
			// We can either fail or just not set the role (which means no company access)
			s.logger.Warn("Failed to fetch company role for active company", "user_id", u.ID, "company_id", u.ActiveCompanyID, "error", err)
		}
	}

	accessToken, err := s.token.GenerateToken(u.ID, u.TenantID, roles, u.ActiveCompanyID, companyRole, u.IdentityType)
	if err != nil {
		return nil, err
	}

	// 5. Create Session with request metadata from context
	refreshToken := s.token.GenerateRefreshToken()

	// Extract request metadata from context (set by RequestMetadata middleware)
	userAgent := getStringFromContext(ctx, "user_agent", "unknown")
	ipAddress := getStringFromContext(ctx, "ip_address", "unknown")

	sess := session.NewSession(
		s.token.GenerateID(),
		u.ID,
		refreshToken,
		userAgent,
		ipAddress,
		time.Now().Add(7*24*time.Hour), // 7 days expiration
	)

	if err := s.sessionRepo.Create(ctx, sess); err != nil {
		return nil, err
	}

	// Update user login stats
	u.LastLoginAt = time.Now()
	u.LoginAttempts = 0
	_ = s.userRepo.Update(ctx, u) // Ignore error for stats update

	logger.Info("Login successful", "user_id", u.ID, "tenant_id", u.TenantID)

	return &LoginResult{
		Token: &TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    86400, // 24 hours
		},
		User: u,
	}, nil
}

// RefreshToken refreshes the access token using a valid refresh token.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	logger := s.logger.WithContext(ctx)
	logger.Debug("Attempting token refresh")

	sess, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	if !sess.IsValid() {
		return nil, errors.New("session expired or revoked")
	}

	u, err := s.userRepo.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}

	roles, err := s.policyRepo.GetRolesForUser(ctx, u.ID)
	if err != nil {
		roles = []string{}
	}

	var companyRole string
	if u.ActiveCompanyID != "" {
		cr, err := s.companyUserRepo.GetUserRole(ctx, u.ActiveCompanyID, u.ID)
		if err == nil {
			companyRole = cr.String()
		}
	}

	accessToken, err := s.token.GenerateToken(u.ID, u.TenantID, roles, u.ActiveCompanyID, companyRole, u.IdentityType)
	if err != nil {
		return nil, err
	}

	// Optionally rotate refresh token here
	// newRefreshToken := s.token.GenerateRefreshToken()
	// sess.RefreshToken = newRefreshToken
	// s.sessionRepo.Update(ctx, sess)

	logger.Info("Token refreshed successfully", "user_id", u.ID)

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: sess.RefreshToken,
		ExpiresIn:    86400,
	}, nil
}

// Logout revokes the session associated with the refresh token.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	logger := s.logger.WithContext(ctx)
	sess, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		logger.Warn("Logout failed: invalid refresh token")
		return nil // Already logged out or invalid
	}

	logger.Info("User logging out", "user_id", sess.UserID)

	sess.Revoked = true
	return s.sessionRepo.Update(ctx, sess)
}

// ValidateToken validates a token and returns the claims.
func (s *AuthService) ValidateToken(tokenString string) (*ports.Claims, error) {
	return s.token.ValidateToken(tokenString)
}

// IntrospectToken validates a JWT and returns its claims.
// Returns an error if the token is invalid, expired, or cannot be parsed.
// The caller is responsible for translating errors into the appropriate HTTP response
// (typically {"active": false} rather than a 4xx status).
func (s *AuthService) IntrospectToken(ctx context.Context, tokenString string) (*ports.Claims, error) {
	claims, err := s.token.ValidateToken(tokenString)
	if err != nil {
		s.logger.WithContext(ctx).Debug("Token introspection: invalid token", "error", err)
		return nil, err
	}
	return claims, nil
}

// GetUser retrieves a user by ID (for /me endpoint).
func (s *AuthService) GetUser(ctx context.Context, userID string) (*user.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// VerifyEmail verifies a user's email address using the token.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	u, err := s.userRepo.GetByVerificationToken(ctx, token)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("invalid token")
	}

	// Check expiration
	if time.Now().After(u.VerificationTokenExpiresAt) {
		return errors.New("token expired")
	}

	// Update user status
	u.Status = user.StatusActive
	u.VerificationToken = ""
	u.VerificationTokenExpiresAt = time.Time{}

	return s.userRepo.Update(ctx, u)
}

// GenerateTokenForUser generates a token for an already-authenticated user (e.g., after 2FA or passkey).
func (s *AuthService) GenerateTokenForUser(ctx context.Context, userID string) (*TokenPair, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("user not found")
	}

	roles, err := s.policyRepo.GetRolesForUser(ctx, u.ID)
	if err != nil {
		roles = []string{}
	}

	var companyRole string
	if u.ActiveCompanyID != "" {
		cr, err := s.companyUserRepo.GetUserRole(ctx, u.ActiveCompanyID, u.ID)
		if err == nil {
			companyRole = cr.String()
		}
	}

	accessToken, err := s.token.GenerateToken(u.ID, u.TenantID, roles, u.ActiveCompanyID, companyRole, u.IdentityType)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken: accessToken,
		ExpiresIn:   86400, // 24 hours
	}, nil
}

// getStringFromContext extracts a string from context with a default fallback.
// This is used to get request metadata set by middleware.
func getStringFromContext(ctx context.Context, key string, fallback string) string {
	if val, ok := ctx.Value(key).(string); ok && val != "" {
		return val
	}
	return fallback
}
