// Package handler contains HTTP request handlers.
// Handlers are THIN - they only parse requests, call services, and write responses.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/captcha"
	"github.com/pgdepaula/vyst-openauth/internal/domain/document"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	registrationSvc *service.RegistrationService
	authSvc         *service.AuthService
	totpSvc         *service.TOTPService
	captchaSvc      ports.CaptchaService
	quotaEnforcer   *middleware.QuotaEnforcer
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(
	registrationSvc *service.RegistrationService,
	authSvc *service.AuthService,
	totpSvc *service.TOTPService,
	captchaSvc ports.CaptchaService,
	quotaEnforcer *middleware.QuotaEnforcer,
) *AuthHandler {
	return &AuthHandler{
		registrationSvc: registrationSvc,
		authSvc:         authSvc,
		totpSvc:         totpSvc,
		captchaSvc:      captchaSvc,
		quotaEnforcer:   quotaEnforcer,
	}
}

// RegisterRequest is the request body for user registration.
type RegisterRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	TenantName   string `json:"tenant_name"`
	CaptchaToken string `json:"captcha_token"`
	CPF          string `json:"cpf"`
}

// RegisterResponse is the response for successful registration.
type RegisterResponse struct {
	Message  string `json:"message"`
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
}

// Register handles POST /auth/register
// @Summary		Register a new user
// @Description	Creates a new user account with a new tenant
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body	body		RegisterRequest	true	"Registration data"
// @Success		201		{object}	RegisterResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Router		/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Email == "" || req.Password == "" || req.TenantName == "" {
		writeError(r.Context(), w, "Email, password, and tenant_name are required", http.StatusBadRequest)
		return
	}

	// Verify CAPTCHA
	if err := h.validateCaptcha(r, req.CaptchaToken); err != nil {
		h.handleCaptchaError(w, r, err)
		return
	}

	// Call service
	result, err := h.registrationSvc.RegisterWithTenant(r.Context(), service.RegisterCommand{
		Email:      req.Email,
		Password:   req.Password,
		TenantName: req.TenantName,
		CPF:        req.CPF,
	})
	if err != nil {
		if errors.Is(err, document.ErrCPFInvalid) || errors.Is(err, document.ErrCPFBlacklisted) {
			writeError(r.Context(), w, err.Error(), http.StatusBadRequest)
			return
		}
		writeError(r.Context(), w, "Registration failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Track active tenant for billing
	if h.quotaEnforcer != nil {
		if err := h.quotaEnforcer.AddActiveTenant(r.Context(), result.Tenant.ID); err != nil {
			middleware.GetLogger(r.Context()).Warn("Failed to track active tenant", "tenant_id", result.Tenant.ID, "error", err)
		}
	}

	middleware.GetLogger(r.Context()).Info("User registered", "user_id", result.User.ID, "tenant_id", result.Tenant.ID)

	writeJSON(w, http.StatusCreated, RegisterResponse{
		Message:  "User registered successfully",
		UserID:   result.User.ID,
		TenantID: result.Tenant.ID,
	})
}

// LoginRequest is the request body for login.
type LoginRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captcha_token"`
	TOTPCode     string `json:"totp_code,omitempty"`
	TempToken    string `json:"temp_token,omitempty"`
}

// LoginResponse is the response for successful login.
type LoginResponse struct {
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	Requires2FA  bool   `json:"requires_2fa,omitempty"`
	TempToken    string `json:"temp_token,omitempty"`
}

// Login handles POST /auth/login
// @Summary		Authenticate user
// @Description	Authenticates a user with email and password, optionally with 2FA
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body	body		LoginRequest	true	"Login credentials"
// @Success		200		{object}	LoginResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Router		/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Handle 2FA verification step (temp token provided)
	if req.TempToken != "" && req.TOTPCode != "" {
		h.verifyTOTPLogin(w, r, req.TempToken, req.TOTPCode)
		return
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		writeError(r.Context(), w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Verify CAPTCHA
	if err := h.validateCaptcha(r, req.CaptchaToken); err != nil {
		h.handleCaptchaError(w, r, err)
		return
	}

	// Call service
	loginResult, err := h.authSvc.LoginWithUser(r.Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			writeError(r.Context(), w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		if err == service.ErrUserNotActive {
			writeError(r.Context(), w, "User is not active. Please verify your email.", http.StatusForbidden)
			return
		}
		writeError(r.Context(), w, "Login failed", http.StatusInternalServerError)
		return
	}

	if h.handleTOTPRequirement(w, r, req, loginResult.User.ID) {
		return
	}

	// Track active tenant for billing
	if h.quotaEnforcer != nil && loginResult.User.TenantID != "" {
		if err := h.quotaEnforcer.AddActiveTenant(r.Context(), loginResult.User.TenantID); err != nil {
			middleware.GetLogger(r.Context()).Warn("Failed to track active tenant", "tenant_id", loginResult.User.TenantID, "error", err)
		}
	}

	middleware.GetLogger(r.Context()).Info("User logged in", "user_id", loginResult.User.ID)

	writeJSON(w, http.StatusOK, LoginResponse{
		Token:        loginResult.Token.AccessToken,
		RefreshToken: loginResult.Token.RefreshToken,
		ExpiresIn:    loginResult.Token.ExpiresIn,
	})
}

func (h *AuthHandler) handleTOTPRequirement(w http.ResponseWriter, r *http.Request, req LoginRequest, userID string) bool {
	if h.totpSvc == nil {
		return false
	}

	has2FA, err := h.totpSvc.IsEnabled(r.Context(), userID)
	if err != nil || !has2FA {
		return false
	}

	if req.TOTPCode == "" {
		tempToken, err := h.totpSvc.GenerateTempToken(userID)
		if err != nil {
			writeError(r.Context(), w, "Failed to generate 2FA token", http.StatusInternalServerError)
			return true
		}
		writeJSON(w, http.StatusOK, LoginResponse{
			Requires2FA: true,
			TempToken:   tempToken,
		})
		return true
	}

	if !h.totpSvc.VerifyCode(r.Context(), userID, req.TOTPCode) {
		writeError(r.Context(), w, "Invalid 2FA code", http.StatusUnauthorized)
		return true
	}

	return false
}

// Me handles GET /auth/me - returns the current user info.
// @Summary		Get current user
// @Description	Returns the currently authenticated user's information
// @Tags		Auth
// @Produce		json
// @Security	BearerAuth
// @Success		200	{object}	UserResponse
// @Failure		401	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
// @Router		/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	// UserID should be set by auth middleware in context
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.authSvc.GetUser(r.Context(), userID)
	if err != nil {
		writeError(r.Context(), w, "User not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, ToUserResponse(user))
}

// IntrospectRequest is the request body for token introspection.
type IntrospectRequest struct {
	Token string `json:"token"`
}

// IntrospectResponse is the response from the token introspection endpoint.
// Active is false when the token is invalid, expired, or revoked.
type IntrospectResponse struct {
	Active          bool     `json:"active"`
	UserID          string   `json:"user_id,omitempty"`
	TenantID        string   `json:"tenant_id,omitempty"`
	Roles           []string `json:"roles,omitempty"`
	ActiveCompanyID string   `json:"active_company_id,omitempty"`
	CompanyRole     string   `json:"company_role,omitempty"`
	IdentityType    string   `json:"identity_type,omitempty"`
	ExpiresAt       int64    `json:"exp,omitempty"`
}

// IntrospectToken handles POST /api/v1/auth/introspect
//
// Validates a JWT and returns its claims if active. Returns {"active": false}
// for expired, invalid, or revoked tokens. This endpoint does not require
// authentication — it is designed for service-to-service token validation.
//
// @Summary		Introspect a token
// @Description	Validates a JWT and returns its claims if the token is active
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body	body	IntrospectRequest	true	"Token to introspect"
// @Success		200	{object}	IntrospectResponse
// @Failure		400	{object}	ErrorResponse
// @Router		/api/v1/auth/introspect [post]
func (h *AuthHandler) IntrospectToken(w http.ResponseWriter, r *http.Request) {
	var req IntrospectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		writeError(r.Context(), w, "token is required", http.StatusBadRequest)
		return
	}

	claims, err := h.authSvc.IntrospectToken(r.Context(), req.Token)
	if err != nil {
		// Return active=false for invalid/expired tokens — do not return 401,
		// as introspection itself succeeded (the token is simply not active).
		writeJSON(w, http.StatusOK, IntrospectResponse{Active: false})
		return
	}

	writeJSON(w, http.StatusOK, IntrospectResponse{
		Active:          true,
		UserID:          claims.UserID,
		TenantID:        claims.TenantID,
		Roles:           claims.Roles,
		ActiveCompanyID: claims.ActiveCompanyID,
		CompanyRole:     claims.CompanyRole,
		IdentityType:    claims.IdentityType,
		ExpiresAt:       claims.ExpiresAt.Unix(),
	})
}

// VerifyEmail handles GET /auth/verify-email
// @Summary		Verify email address
// @Description	Verifies a user's email address using the provided token
// @Tags		Auth
// @Produce		json
// @Param		token	query	string	true	"Verification token"
// @Success		200	{object}	MessageResponse
// @Failure		400	{object}	ErrorResponse
// @Router		/auth/verify-email [get]
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(r.Context(), w, "Token is required", http.StatusBadRequest)
		return
	}

	if err := h.authSvc.VerifyEmail(r.Context(), token); err != nil {
		writeError(r.Context(), w, "Verification failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Email verified successfully"})
}

// verifyTOTPLogin handles the second step of 2FA login.
func (h *AuthHandler) verifyTOTPLogin(w http.ResponseWriter, r *http.Request, tempToken, totpCode string) {
	if h.totpSvc == nil {
		writeError(r.Context(), w, "2FA not configured", http.StatusInternalServerError)
		return
	}

	// Validate temp token
	userID, err := h.totpSvc.ValidateTempToken(r.Context(), tempToken)
	if err != nil {
		writeError(r.Context(), w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	// Verify TOTP code
	if !h.totpSvc.VerifyCode(r.Context(), userID, totpCode) {
		writeError(r.Context(), w, "Invalid 2FA code", http.StatusUnauthorized)
		return
	}

	// Generate token for the already-authenticated user
	tokenPair, err := h.authSvc.GenerateTokenForUser(r.Context(), userID)
	if err != nil {
		writeError(r.Context(), w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Token:     tokenPair.AccessToken,
		ExpiresIn: tokenPair.ExpiresIn,
	})
}

// GetCaptchaSiteKey returns the Turnstile site key for the frontend.
// @Summary		Get CAPTCHA configuration
// @Description	Returns the CAPTCHA site key and enabled status
// @Tags		Auth
// @Produce		json
// @Success		200	{object}	map[string]interface{}
// @Router		/auth/captcha-config [get]
func (h *AuthHandler) GetCaptchaSiteKey(w http.ResponseWriter, r *http.Request) {
	if h.captchaSvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"site_key": "",
			"enabled":  false,
		})
		return
	}

	config := h.captchaSvc.GetConfig()
	writeJSON(w, http.StatusOK, config)
}

// RefreshTokenRequest is the request body for refreshing a token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshToken handles POST /auth/refresh
// @Summary		Refresh access token
// @Description	Generates a new access token using a valid refresh token
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body	body	RefreshTokenRequest	true	"Refresh token"
// @Success		200	{object}	LoginResponse
// @Failure		400	{object}	ErrorResponse
// @Failure		401	{object}	ErrorResponse
// @Router		/auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		writeError(r.Context(), w, "Refresh token is required", http.StatusBadRequest)
		return
	}

	tokenPair, err := h.authSvc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(r.Context(), w, "Refresh failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Token:     tokenPair.AccessToken,
		ExpiresIn: tokenPair.ExpiresIn,
		// We don't return a new refresh token here unless we implement rotation,
		// but AuthService.RefreshToken returns the *same* refresh token currently.
		// Let's include it for consistency if the client expects it.
		// LoginResponse struct doesn't have RefreshToken field, let's add it or just return Token/ExpiresIn.
		// The proto definition returns both. Let's stick to what LoginResponse has for now or update it.
		// LoginResponse has Token (access token).
	})
}

// LogoutRequest is the request body for logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout handles POST /auth/logout
// @Summary		Logout user
// @Description	Invalidates the refresh token and logs out the user
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body	body	LogoutRequest	true	"Refresh token to invalidate"
// @Success		200	{object}	MessageResponse
// @Failure		400	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router		/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		writeError(r.Context(), w, "Refresh token is required", http.StatusBadRequest)
		return
	}

	if err := h.authSvc.Logout(r.Context(), req.RefreshToken); err != nil {
		writeError(r.Context(), w, "Logout failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

func writeError(ctx context.Context, w http.ResponseWriter, message string, status int) {
	middleware.GetLogger(ctx).Error("HTTP Error", "status", status, "error", message)
	writeJSON(w, status, map[string]string{"error": message})
}

// validateCaptcha validates the CAPTCHA token if the service is enabled.
// Returns nil if validation passes or CAPTCHA is disabled.
func (h *AuthHandler) validateCaptcha(r *http.Request, token string) error {
	if h.captchaSvc == nil || !h.captchaSvc.IsEnabled() {
		return nil
	}
	remoteIP := getRemoteIP(r)
	return h.captchaSvc.ValidateToken(r.Context(), token, remoteIP)
}

// handleCaptchaError writes an appropriate HTTP error response for CAPTCHA errors.
func (h *AuthHandler) handleCaptchaError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, captcha.ErrCaptchaTokenMissing):
		writeError(r.Context(), w, "CAPTCHA token is required", http.StatusBadRequest)
	case errors.Is(err, captcha.ErrCaptchaInvalid):
		writeError(r.Context(), w, "CAPTCHA verification failed", http.StatusBadRequest)
	case errors.Is(err, captcha.ErrCaptchaExpired):
		writeError(r.Context(), w, "CAPTCHA challenge expired, please try again", http.StatusBadRequest)
	default:
		middleware.GetLogger(r.Context()).Error("CAPTCHA error", "error", err)
		writeError(r.Context(), w, "CAPTCHA verification error", http.StatusInternalServerError)
	}
}

// getRemoteIP extracts the client IP address from the request,
// checking X-Forwarded-For and X-Real-IP headers first.
func getRemoteIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}
