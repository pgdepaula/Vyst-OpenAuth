package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
)

// InviteUserRequest is the request body for inviting a user.
type InviteUserRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// InviteUser handles POST /api/v1/companies/{id}/invitations
// @Summary		Invite a user to the company
// @Description	Sends an invitation email to the user
// @Tags		Companies
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		id		path		string				true	"Company ID"
// @Param		body	body		InviteUserRequest	true	"Invitation data"
// @Success		201		{object}	MessageResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Router		/api/v1/companies/{id}/invitations [post]
func (h *CompanyHandler) InviteUser(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	if companyID == "" {
		writeError(r.Context(), w, "company id is required", http.StatusBadRequest)
		return
	}

	var req InviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		writeError(r.Context(), w, "email is required", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	role := company.CompanyRole(req.Role)
	if !role.IsValid() {
		role = company.RoleMember
	}

	err := h.invitationSvc.InviteUser(r.Context(), userID, companyID, req.Email, role)
	if err != nil {
		writeError(r.Context(), w, err.Error(), http.StatusBadRequest) // Simplify error handling for now
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "Invitation sent successfully"})
}

// AcceptInvitation handles POST /api/v1/invitations/{token}/accept
// @Summary		Accept an invitation
// @Description	Accepts an invitation to join a company
// @Tags		Invitations
// @Produce		json
// @Security	BearerAuth
// @Param		token	path		string	true	"Invitation token"
// @Success		200		{object}	MessageResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router		/api/v1/invitations/{token}/accept [post]
func (h *CompanyHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(r.Context(), w, "token is required", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.invitationSvc.AcceptInvitation(r.Context(), token, userID)
	if err != nil {
		writeError(r.Context(), w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation accepted successfully"})
}
