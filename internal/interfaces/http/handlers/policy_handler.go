package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
)

// PolicyHandler handles policy-related HTTP requests.
type PolicyHandler struct {
	policySvc *service.PolicyService
}

// NewPolicyHandler creates a new policy handler.
func NewPolicyHandler(policySvc *service.PolicyService) *PolicyHandler {
	return &PolicyHandler{
		policySvc: policySvc,
	}
}

// RoleRequest is the request body for creating/updating a role.
type RoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// CreateRoleRequest is an alias for RoleRequest (for Swagger docs).
type CreateRoleRequest = RoleRequest

// UpdateRoleRequest is an alias for RoleRequest (for Swagger docs).
type UpdateRoleRequest = RoleRequest

// ListRoles handles GET /api/v1/roles
// @Summary		List all roles
// @Description	Returns all roles for the current tenant
// @Tags		Roles
// @Produce		json
// @Security	BearerAuth
// @Success		200	{array}	policy.Role
// @Failure		401	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router		/api/v1/roles [get]
func (h *PolicyHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roles, err := h.policySvc.ListRoles(r.Context(), tenantID)
	if err != nil {
		writeError(r.Context(), w, "Failed to list roles", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, roles)
}

// GetRole handles GET /api/v1/roles/{id}
// @Summary		Get role by ID
// @Description	Returns a specific role by its ID
// @Tags		Roles
// @Produce		json
// @Security	BearerAuth
// @Param		id	path	string	true	"Role ID"
// @Success		200	{object}	policy.Role
// @Failure		401	{object}	ErrorResponse
// @Failure		404	{object}	ErrorResponse
// @Router		/api/v1/roles/{id} [get]
func (h *PolicyHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, "Role ID required", http.StatusBadRequest)
		return
	}

	role, err := h.policySvc.GetRole(r.Context(), id)
	if err != nil {
		writeError(r.Context(), w, "Role not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, role)
}

// CreateRole handles POST /api/v1/roles
// @Summary		Create a new role
// @Description	Creates a new role in the current tenant
// @Tags		Roles
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		body	body	CreateRoleRequest	true	"Role data"
// @Success		201	{object}	policy.Role
// @Failure		400	{object}	ErrorResponse
// @Failure		401	{object}	ErrorResponse
// @Router		/api/v1/roles [post]
func (h *PolicyHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req RoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeError(r.Context(), w, "Role name is required", http.StatusBadRequest)
		return
	}

	role, err := h.policySvc.CreateRole(r.Context(), req.Name, req.Description, tenantID, req.Permissions)
	if err != nil {
		middleware.GetLogger(r.Context()).Error("Failed to create role", "error", err)
		writeError(r.Context(), w, "Failed to create role", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, role)
}

// UpdateRole handles PUT /api/v1/roles/{id}
// @Summary		Update a role
// @Description	Updates an existing role
// @Tags		Roles
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		id	path	string	true	"Role ID"
// @Param		body	body	UpdateRoleRequest	true	"Updated role data"
// @Success		200	{object}	policy.Role
// @Failure		400	{object}	ErrorResponse
// @Failure		401	{object}	ErrorResponse
// @Router		/api/v1/roles/{id} [put]
func (h *PolicyHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, "Role ID required", http.StatusBadRequest)
		return
	}

	var req RoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	role, err := h.policySvc.UpdateRole(r.Context(), id, req.Name, req.Description, req.Permissions)
	if err != nil {
		writeError(r.Context(), w, "Failed to update role", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, role)
}

// DeleteRole handles DELETE /api/v1/roles/{id}
// @Summary		Delete a role
// @Description	Deletes a role by its ID
// @Tags		Roles
// @Security	BearerAuth
// @Param		id	path	string	true	"Role ID"
// @Success		204	"No Content"
// @Failure		401	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router		/api/v1/roles/{id} [delete]
func (h *PolicyHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, "Role ID required", http.StatusBadRequest)
		return
	}

	if err := h.policySvc.DeleteRole(r.Context(), id); err != nil {
		writeError(r.Context(), w, "Failed to delete role", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CheckPermissionRequest is the request body for checking permissions.
type CheckPermissionRequest struct {
	UserID   string `json:"user_id"`
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

// CheckPermissionResponse is the response for permission check.
type CheckPermissionResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// CheckPermission handles POST /api/v1/authz/check
// @Summary		Check permission
// @Description	Checks if a user has permission to perform an action on a resource
// @Tags		Authz
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		body	body	CheckPermissionRequest	true	"Check request"
// @Success		200	{object}	CheckPermissionResponse
// @Failure		400	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router		/api/v1/authz/check [post]
func (h *PolicyHandler) CheckPermission(w http.ResponseWriter, r *http.Request) {
	var req CheckPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.Action == "" || req.Resource == "" {
		writeError(r.Context(), w, "user_id, action, and resource are required", http.StatusBadRequest)
		return
	}

	// Format subject as "user:{id}" usually, but let's check what the SDK sends.
	// SDK sends: `{"user_id":"%s","action":"%s","resource":"%s"}`
	// The ReBAC system expects subjects like "user:abc".
	// The Service layer expects pure strings.
	// We might need to map it here.
	// Looking at tuple.go: Subject string.
	subject := "user:" + req.UserID
	if req.UserID[:5] == "user:" {
		subject = req.UserID
	}

	// Relation is the Action? Or should we map Actions to Relations?
	// Usually Action == Relation in simple Zanzibar models.
	relation := req.Action

	// Resource is Object.
	object := req.Resource

	allowed, err := h.policySvc.CheckPermission(r.Context(), subject, relation, object)
	if err != nil {
		middleware.GetLogger(r.Context()).Error("Permission check failed", "error", err)
		// Don't return error to client, just say denied (fail close) or internal server error?
		// Better return 500 if DB failed.
		writeError(r.Context(), w, "Failed to check permission", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, CheckPermissionResponse{
		Allowed: allowed,
	})
}
