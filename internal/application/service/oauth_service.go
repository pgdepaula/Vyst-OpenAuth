package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

// OAuthProvider represents a supported OAuth provider (e.g., Google, GitHub).
type OAuthProvider string

const (
	ProviderGoogle OAuthProvider = "google"
	ProviderGitHub OAuthProvider = "github"
)

// OAuthService handles OAuth2 and OIDC authentication flows.
type OAuthService struct {
	userRepo user.Repository
	tokenSvc ports.TokenService
	// In a real implementation, we would inject a map of providers or a factory
	// providers map[OAuthProvider]ports.OAuthProvider
}

// NewOAuthService creates a new OAuth service.
func NewOAuthService(userRepo user.Repository, tokenSvc ports.TokenService) *OAuthService {
	return &OAuthService{
		userRepo: userRepo,
		tokenSvc: tokenSvc,
	}
}

// GetAuthURL returns the URL to redirect the user to for the given provider.
func (s *OAuthService) GetAuthURL(provider OAuthProvider) (string, error) {
	// Mock implementation - in reality, this would use the provider's SDK/config to generate the URL
	switch provider {
	case ProviderGoogle:
		return "https://accounts.google.com/o/oauth2/v2/auth?client_id=MOCK&redirect_uri=MOCK&response_type=code&scope=email", nil
	case ProviderGitHub:
		return "https://github.com/login/oauth/authorize?client_id=MOCK&redirect_uri=MOCK&scope=user:email", nil
	default:
		return "", errors.New("unsupported provider")
	}
}

// Callback handles the OAuth2 callback, exchanges code for token, and logs in/registers the user.
func (s *OAuthService) Callback(ctx context.Context, provider OAuthProvider, code string) (*TokenPair, error) {
	// 1. Exchange code for token (Mock)
	// token, err := s.providers[provider].Exchange(ctx, code)

	// 2. Get User Info from Provider (Mock)
	// userInfo, err := s.providers[provider].GetUserInfo(ctx, token)

	// Mock User Info
	mockEmail := fmt.Sprintf("user_%s@example.com", code) // Use code to simulate different users

	// 3. Find or Create User
	u, err := s.userRepo.GetByEmail(ctx, mockEmail)
	if err != nil {
		// Create new user (Simplified registration for OAuth)
		// ...
		return nil, fmt.Errorf("user not found (auto-registration not implemented in this mock): %w", err)
	}

	// 4. Generate Tokens
	// Default to individual identity and no company role for OAuth login initially
	accessToken, err := s.tokenSvc.GenerateToken(u.ID, u.TenantID, []string{"user"}, "", "", user.IdentityTypeIndividual)
	if err != nil {
		return nil, err
	}

	refreshToken := s.tokenSvc.GenerateRefreshToken()

	// 5. Create Session (Should inject SessionRepo and use it here, similar to AuthService)
	// For now, returning tokens.

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    86400,
	}, nil
}
