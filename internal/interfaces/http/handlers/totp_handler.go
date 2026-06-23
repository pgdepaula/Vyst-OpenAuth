package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
)

// TOTPHandler handles 2FA-related HTTP requests.
type TOTPHandler struct {
	totpSvc *service.TOTPService
	authSvc *service.AuthService
}

// NewTOTPHandler creates a new TOTP handler.
func NewTOTPHandler(totpSvc *service.TOTPService, authSvc *service.AuthService) *TOTPHandler {
	return &TOTPHandler{
		totpSvc: totpSvc,
		authSvc: authSvc,
	}
}

// SetupRequest is the request for starting 2FA setup.
type SetupRequest struct{}

// SetupResponse contains the QR code and secret for setup.
type SetupResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// Setup handles POST /auth/2fa/setup - initiates 2FA setup.
func (h *TOTPHandler) Setup(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user email
	user, err := h.authSvc.GetUser(r.Context(), userID)
	if err != nil {
		writeError(r.Context(), w, "User not found", http.StatusNotFound)
		return
	}

	result, err := h.totpSvc.GenerateSecret(r.Context(), userID, user.Email)
	if err != nil {
		writeError(r.Context(), w, "Failed to setup 2FA: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, SetupResponse{
		Secret:      result.Secret,
		QRCodeURL:   result.QRCodeURL,
		BackupCodes: result.BackupCodes,
	})
}

// VerifyRequest is the request for verifying 2FA setup.
type VerifyRequest struct {
	Code string `json:"code"`
}

// Verify handles POST /auth/2fa/verify - verifies setup code and enables 2FA.
func (h *TOTPHandler) Verify(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" {
		writeError(r.Context(), w, "Verification code is required", http.StatusBadRequest)
		return
	}

	if err := h.totpSvc.VerifySetup(r.Context(), userID, req.Code); err != nil {
		writeError(r.Context(), w, "Verification failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Two-factor authentication enabled successfully",
	})
}

// StatusResponse contains 2FA status for a user.
type StatusResponse struct {
	Enabled bool `json:"enabled"`
}

// Status handles GET /auth/2fa/status - returns 2FA status.
func (h *TOTPHandler) Status(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	enabled, err := h.totpSvc.IsEnabled(r.Context(), userID)
	if err != nil {
		writeError(r.Context(), w, "Failed to get 2FA status", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, StatusResponse{Enabled: enabled})
}

// Disable handles DELETE /auth/2fa - disables 2FA.
func (h *TOTPHandler) Disable(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Optionally require TOTP code to disable 2FA for security
	code := r.URL.Query().Get("code")
	if code != "" {
		if !h.totpSvc.VerifyCode(r.Context(), userID, code) {
			writeError(r.Context(), w, "Invalid 2FA code", http.StatusUnauthorized)
			return
		}
	}

	if err := h.totpSvc.Disable(r.Context(), userID); err != nil {
		writeError(r.Context(), w, "Failed to disable 2FA: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Two-factor authentication disabled",
	})
}
