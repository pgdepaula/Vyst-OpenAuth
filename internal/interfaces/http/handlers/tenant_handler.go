package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
)

// TenantHandler handles tenant-related HTTP requests.
type TenantHandler struct {
	tenantSvc *service.TenantService
}

// NewTenantHandler creates a new tenant handler.
func NewTenantHandler(tenantSvc *service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantSvc: tenantSvc,
	}
}

// CreateTenantRequest is the request body for creating a tenant.
type CreateTenantRequest struct {
	Name string `json:"name"`
}

// CreateTenant handles POST /api/v1/tenants
// @Summary		Create a new tenant
// @Description	Creates a new tenant with the current user as owner
// @Tags		Tenants
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		body	body	CreateTenantRequest	true	"Tenant data"
// @Success		201	{object}	tenant.Tenant
// @Failure		400	{object}	ErrorResponse
// @Failure		401	{object}	ErrorResponse
// @Router		/api/v1/tenants [post]
func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	// User must be authenticated to create a tenant
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeError(r.Context(), w, "Tenant name is required", http.StatusBadRequest)
		return
	}

	tenant, err := h.tenantSvc.CreateTenant(r.Context(), req.Name, userID)
	if err != nil {
		writeError(r.Context(), w, "Failed to create tenant", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, tenant)
}

// ListTenants handles GET /api/v1/admin/tenants (Super Admin only)
// @Summary		List all tenants
// @Description	Returns all tenants (Super Admin only)
// @Tags		Tenants
// @Produce		json
// @Security	BearerAuth
// @Success		200	{array}	tenant.Tenant
// @Failure		401	{object}	ErrorResponse
// @Failure		403	{object}	ErrorResponse
// @Router		/api/v1/admin/tenants [get]
func (h *TenantHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	// Ensure the authenticated user has super_admin role
	if !middleware.IsSuperAdmin(r.Context()) {
		writeError(r.Context(), w, "Forbidden: Super Admin access required", http.StatusForbidden)
		return
	}

	tenants, err := h.tenantSvc.ListTenants(r.Context())
	if err != nil {
		writeError(r.Context(), w, "Failed to list tenants", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, tenants)
}

// SuspendTenant handles POST /api/v1/admin/tenants/{id}/suspend (Super Admin only)
// @Summary		Suspend a tenant
// @Description	Suspends a tenant by ID (Super Admin only)
// @Tags		Tenants
// @Security	BearerAuth
// @Param		id	path	string	true	"Tenant ID"
// @Success		200	{object}	MessageResponse
// @Failure		401	{object}	ErrorResponse
// @Failure		403	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router		/api/v1/admin/tenants/{id}/suspend [post]
func (h *TenantHandler) SuspendTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	if err := h.tenantSvc.SuspendTenant(r.Context(), id); err != nil {
		writeError(r.Context(), w, "Failed to suspend tenant", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Tenant suspended"})
}
