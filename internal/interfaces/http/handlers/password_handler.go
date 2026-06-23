package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/pgdepaula/vyst-openauth/internal/application/service"
)

type PasswordHandler struct {
	passwordSvc *service.PasswordService
}

func NewPasswordHandler(passwordSvc *service.PasswordService) *PasswordHandler {
	return &PasswordHandler{
		passwordSvc: passwordSvc,
	}
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

func (h *PasswordHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// Call service - it handles "not found" gracefully
	if err := h.passwordSvc.RequestReset(r.Context(), req.Email); err != nil {
		// Log internally but don't expose to client
		// Ideally, we should return 200 OK even if user not found.
		log.Printf("Password reset request failed: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "If an account with that email exists, a password reset link has been sent."}); err != nil {
		log.Printf("Failed to write password reset response: %v", err)
	}
}

// ResetPasswordRequest is the request body for resetting a password.
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResetPassword handles POST /auth/reset-password
// @Summary		Reset password
// @Description	Resets the user's password using a valid reset token
// @Tags		Password
// @Accept		json
// @Produce		json
// @Param		body	body	ResetPasswordRequest	true	"Reset token and new password"
// @Success		200	{object}	MessageResponse
// @Failure		400	{object}	ErrorResponse
// @Router		/auth/reset-password [post]
func (h *PasswordHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		http.Error(w, "Token and new password are required", http.StatusBadRequest)
		return
	}

	if err := h.passwordSvc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		if err == service.ErrInvalidResetToken {
			http.Error(w, "Invalid or expired reset token", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to reset password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Password has been successfully reset."}); err != nil {
		log.Printf("Failed to write password reset response: %v", err)
	}
}
