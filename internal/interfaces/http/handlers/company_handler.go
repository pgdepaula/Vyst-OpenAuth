// Package handlers contains HTTP request handlers.
// Handlers are THIN - they only parse requests, call services, and write responses.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
)

// CompanyHandler handles company-related HTTP requests.
type CompanyHandler struct {
	companySvc       *service.CompanyService
	companyLookupSvc *service.CompanyLookupService
	authSvc          *service.AuthService
	invitationSvc    *service.InvitationService
}

// NewCompanyHandler creates a new CompanyHandler.
func NewCompanyHandler(companySvc *service.CompanyService, companyLookupSvc *service.CompanyLookupService, authSvc *service.AuthService, invitationSvc *service.InvitationService) *CompanyHandler {
	return &CompanyHandler{
		companySvc:       companySvc,
		companyLookupSvc: companyLookupSvc,
		authSvc:          authSvc,
		invitationSvc:    invitationSvc,
	}
}

// CreateCompanyRequest is the request body for company creation.
type CreateCompanyRequest struct {
	CNPJ               string                 `json:"cnpj"`
	RazaoSocial        string                 `json:"razao_social"`
	NomeFantasia       string                 `json:"nome_fantasia,omitempty"`
	Endereco           *CompanyAddressRequest `json:"endereco,omitempty"`
	RepresentanteLegal string                 `json:"representante_legal,omitempty"`
}

// CompanyAddressRequest is the address portion of the company creation request.
type CompanyAddressRequest struct {
	Logradouro  string `json:"logradouro,omitempty"`
	Numero      string `json:"numero,omitempty"`
	Complemento string `json:"complemento,omitempty"`
	Bairro      string `json:"bairro,omitempty"`
	Cidade      string `json:"cidade,omitempty"`
	UF          string `json:"uf,omitempty"`
	CEP         string `json:"cep,omitempty"`
}

// CompanyResponse is the response for company operations.
type CompanyResponse struct {
	ID                 string                 `json:"id"`
	CNPJ               string                 `json:"cnpj"`
	CNPJFormatted      string                 `json:"cnpj_formatted"`
	RazaoSocial        string                 `json:"razao_social"`
	NomeFantasia       string                 `json:"nome_fantasia,omitempty"`
	Endereco           *CompanyAddressRequest `json:"endereco,omitempty"`
	RepresentanteLegal string                 `json:"representante_legal,omitempty"`
	Status             string                 `json:"status"`
	CreatedAt          string                 `json:"created_at"`
	UpdatedAt          string                 `json:"updated_at"`
}

// CompanyWithRoleResponse includes the user's role in the company.
type CompanyWithRoleResponse struct {
	Company CompanyResponse `json:"company"`
	Role    string          `json:"role"`
	Status  string          `json:"membership_status"`
}

// companyToResponse converts a domain Company to a response DTO.
func companyToResponse(c *company.Company) CompanyResponse {
	resp := CompanyResponse{
		ID:                 c.ID,
		CNPJ:               c.CNPJ,
		CNPJFormatted:      company.FormatCNPJ(c.CNPJ),
		RazaoSocial:        c.RazaoSocial,
		NomeFantasia:       c.NomeFantasia,
		RepresentanteLegal: c.RepresentanteLegal,
		Status:             string(c.Status),
		CreatedAt:          c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if !c.Endereco.IsEmpty() {
		resp.Endereco = &CompanyAddressRequest{
			Logradouro:  c.Endereco.Logradouro,
			Numero:      c.Endereco.Numero,
			Complemento: c.Endereco.Complemento,
			Bairro:      c.Endereco.Bairro,
			Cidade:      c.Endereco.Cidade,
			UF:          c.Endereco.UF,
			CEP:         c.Endereco.CEP,
		}
	}

	return resp
}

// CreateCompany handles POST /api/v1/companies
// @Summary		Create a new company
// @Description	Creates a new company (pessoa jurídica) and adds the creator as admin
// @Tags		Companies
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		body	body		CreateCompanyRequest	true	"Company data"
// @Success		201		{object}	CompanyResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Router		/api/v1/companies [post]
func (h *CompanyHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var req CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.CNPJ == "" {
		writeError(r.Context(), w, "cnpj is required", http.StatusBadRequest)
		return
	}
	if req.RazaoSocial == "" {
		writeError(r.Context(), w, "razao_social is required", http.StatusBadRequest)
		return
	}

	// Get user context from JWT
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID, _ := middleware.TenantIDFromContext(r.Context())

	// Build address from request
	var addr company.Address
	if req.Endereco != nil {
		addr = company.Address{
			Logradouro:  req.Endereco.Logradouro,
			Numero:      req.Endereco.Numero,
			Complemento: req.Endereco.Complemento,
			Bairro:      req.Endereco.Bairro,
			Cidade:      req.Endereco.Cidade,
			UF:          req.Endereco.UF,
			CEP:         req.Endereco.CEP,
		}
	}

	comp, err := h.companySvc.CreateCompany(
		r.Context(),
		tenantID,
		userID,
		service.CreateCompanyRequest{
			CNPJ:               req.CNPJ,
			RazaoSocial:        req.RazaoSocial,
			NomeFantasia:       req.NomeFantasia,
			Endereco:           addr,
			RepresentanteLegal: req.RepresentanteLegal,
		},
	)
	if err != nil {
		switch err {
		case company.ErrCNPJInvalid:
			writeError(r.Context(), w, "Invalid CNPJ", http.StatusBadRequest)
		case company.ErrCNPJTaken:
			writeError(r.Context(), w, "CNPJ already registered", http.StatusConflict)
		default:
			writeError(r.Context(), w, "Failed to create company", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusCreated, companyToResponse(comp))
}

// ListCompanies handles GET /api/v1/companies
// @Summary		List user's companies
// @Description	Returns all companies the authenticated user belongs to
// @Tags		Companies
// @Produce		json
// @Security	BearerAuth
// @Success		200		{object}	map[string]interface{}
// @Failure		401		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Router		/api/v1/companies [get]
func (h *CompanyHandler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	companiesWithRole, err := h.companySvc.GetCompaniesForUser(r.Context(), userID)
	if err != nil {
		writeError(r.Context(), w, "Failed to list companies", http.StatusInternalServerError)
		return
	}

	// Build response
	responses := make([]CompanyWithRoleResponse, 0, len(companiesWithRole))
	for _, cwr := range companiesWithRole {
		responses = append(responses, CompanyWithRoleResponse{
			Company: companyToResponse(cwr.Company),
			Role:    cwr.Role.String(),
			Status:  cwr.Status,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"companies": responses,
		"count":     len(responses),
	})
}

// GetCompany handles GET /api/v1/companies/{id}
// @Summary		Get company details
// @Description	Returns details of a specific company
// @Tags		Companies
// @Produce		json
// @Security	BearerAuth
// @Param		id		path		string	true	"Company ID"
// @Success		200		{object}	CompanyWithRoleResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router		/api/v1/companies/{id} [get]
func (h *CompanyHandler) GetCompany(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	if companyID == "" {
		writeError(r.Context(), w, "company id is required", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check user has access to this company
	role, err := h.companySvc.GetUserRoleInCompany(r.Context(), companyID, userID)
	if err != nil {
		writeError(r.Context(), w, "You do not have access to this company", http.StatusForbidden)
		return
	}

	comp, err := h.companySvc.GetCompanyByID(r.Context(), companyID)
	if err != nil {
		if err == company.ErrNotFound {
			writeError(r.Context(), w, "Company not found", http.StatusNotFound)
			return
		}
		writeError(r.Context(), w, "Failed to get company", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, CompanyWithRoleResponse{
		Company: companyToResponse(comp),
		Role:    role.String(),
		Status:  "active",
	})
}

// AddUserToCompanyRequest is the request body for adding a user to a company.
type AddUserToCompanyRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// AddUserToCompany handles POST /api/v1/companies/{id}/users
// @Summary		Add user to company
// @Description	Adds a user to the company with the specified role
// @Tags		Companies
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		id		path		string					true	"Company ID"
// @Param		body	body		AddUserToCompanyRequest	true	"User data"
// @Success		201		{object}	MessageResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Router		/api/v1/companies/{id}/users [post]
func (h *CompanyHandler) AddUserToCompany(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	if companyID == "" {
		writeError(r.Context(), w, "company id is required", http.StatusBadRequest)
		return
	}

	var req AddUserToCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		writeError(r.Context(), w, "user_id is required", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check invoker is admin of this company
	invokerRole, err := h.companySvc.GetUserRoleInCompany(r.Context(), companyID, userID)
	if err != nil || invokerRole != company.RoleAdmin {
		writeError(r.Context(), w, "Only company admins can add users", http.StatusForbidden)
		return
	}

	// Parse role
	role := company.CompanyRole(req.Role)
	if !role.IsValid() {
		role = company.RoleMember // Default to member
	}

	err = h.companySvc.AddUserToCompany(r.Context(), service.AddUserToCompanyRequest{
		CompanyID: companyID,
		UserID:    req.UserID,
		Role:      role,
		InvitedBy: userID,
	})
	if err != nil {
		switch err {
		case company.ErrAlreadyMember:
			writeError(r.Context(), w, "User is already a member", http.StatusConflict)
		case company.ErrInvalidRole:
			writeError(r.Context(), w, "Invalid role", http.StatusBadRequest)
		default:
			writeError(r.Context(), w, "Failed to add user", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "User added successfully"})
}

// SwitchCompanyRequest is the request body for switching company context.
type SwitchCompanyRequest struct {
	CompanyID string `json:"company_id"`
}

// SwitchCompany handles POST /api/v1/auth/switch-company
// @Summary		Switch company context
// @Description	Switches the user's active company context for subsequent operations
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		body	body		SwitchCompanyRequest	true	"Company to switch to"
// @Success		200		{object}	MessageResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Router		/api/v1/auth/switch-company [post]
func (h *CompanyHandler) SwitchCompany(w http.ResponseWriter, r *http.Request) {
	var req SwitchCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CompanyID == "" {
		writeError(r.Context(), w, "company_id is required", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.companySvc.SwitchCompany(r.Context(), userID, req.CompanyID)
	if err != nil {
		if err == company.ErrUserNotMember {
			writeError(r.Context(), w, "You are not a member of this company", http.StatusForbidden)
			return
		}
		writeError(r.Context(), w, "Failed to switch company: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate new token with updated context
	tokenPair, err := h.authSvc.GenerateTokenForUser(r.Context(), userID)
	if err != nil {
		writeError(r.Context(), w, "Failed to generate new token context", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "Company context switched successfully",
		"token":         tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken, // Optional, but good practice
		"expires_in":    tokenPair.ExpiresIn,
	})
}

// ClearCompanyContext handles DELETE /api/v1/auth/company-context
// @Summary		Clear company context
// @Description	Clears the user's active company context, switching back to individual mode
// @Tags		Auth
// @Produce		json
// @Security	BearerAuth
// @Success		200		{object}	MessageResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		500		{object}	ErrorResponse
// @Router		/api/v1/auth/company-context [delete]
func (h *CompanyHandler) ClearCompanyContext(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.companySvc.ClearCompanyContext(r.Context(), userID)
	if err != nil {
		writeError(r.Context(), w, "Failed to clear company context", http.StatusInternalServerError)
		return
	}

	// Generate new token with updated context (individual)
	tokenPair, err := h.authSvc.GenerateTokenForUser(r.Context(), userID)
	if err != nil {
		writeError(r.Context(), w, "Failed to generate new token context", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "Company context cleared",
		"token":         tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
	})
}

// RemoveUserFromCompanyRequest is the request body for removing a user.
type RemoveUserFromCompanyRequest struct {
	Reason string `json:"reason,omitempty"`
}

// RemoveUserFromCompany handles DELETE /api/v1/companies/{id}/users/{userId}
// @Summary		Remove user from company
// @Description	Removes a user from the company
// @Tags		Companies
// @Produce		json
// @Security	BearerAuth
// @Param		id		path		string	true	"Company ID"
// @Param		userId	path		string	true	"User ID to remove"
// @Success		200		{object}	MessageResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router		/api/v1/companies/{id}/users/{userId} [delete]
func (h *CompanyHandler) RemoveUserFromCompany(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "userId")

	if companyID == "" || targetUserID == "" {
		writeError(r.Context(), w, "company_id and user_id are required", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check invoker is admin of this company
	invokerRole, err := h.companySvc.GetUserRoleInCompany(r.Context(), companyID, userID)
	if err != nil || invokerRole != company.RoleAdmin {
		writeError(r.Context(), w, "Only company admins can remove users", http.StatusForbidden)
		return
	}

	// Parse optional reason from body
	var req RemoveUserFromCompanyRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // Ignore error, reason is optional

	err = h.companySvc.RemoveUserFromCompany(r.Context(), companyID, targetUserID, userID, req.Reason)
	if err != nil {
		if err == company.ErrUserNotMember {
			writeError(r.Context(), w, "User is not a member", http.StatusNotFound)
			return
		}
		writeError(r.Context(), w, "Failed to remove user", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "User removed successfully"})
}

// RequestJoin handles POST /api/v1/companies/{id}/join-requests
// @Summary		Request to join a company
// @Description	Authenticated user requests to join the specified company
// @Tags		Companies
// @Produce		json
// @Security	BearerAuth
// @Param		id		path		string	true	"Company ID"
// @Success		201		{object}	MessageResponse
// @Failure		401		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse
// @Router		/api/v1/companies/{id}/join-requests [post]
func (h *CompanyHandler) RequestJoin(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	if companyID == "" {
		writeError(r.Context(), w, "company_id is required", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.companySvc.RequestJoin(r.Context(), companyID, userID)
	if err != nil {
		switch err {
		case company.ErrAlreadyMember:
			writeError(r.Context(), w, "Already a member or request pending", http.StatusConflict)
		case company.ErrNotFound:
			writeError(r.Context(), w, "Company not found", http.StatusNotFound)
		default:
			writeError(r.Context(), w, "Failed to submit join request: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "Join request submitted successfully"})
}

// ApproveMember handles POST /api/v1/companies/{id}/members/{userId}/approve
// @Summary		Approve a member join request
// @Description	Approves a pending member (Admin only)
// @Tags		Companies
// @Produce		json
// @Security	BearerAuth
// @Param		id		path		string	true	"Company ID"
// @Param		userId	path		string	true	"Target User ID"
// @Success		200		{object}	MessageResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router		/api/v1/companies/{id}/members/{userId}/approve [post]
func (h *CompanyHandler) ApproveMember(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "userId")
	if companyID == "" || targetUserID == "" {
		writeError(r.Context(), w, "company_id and user_id are required", http.StatusBadRequest)
		return
	}

	adminID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.companySvc.ApproveMember(r.Context(), companyID, adminID, targetUserID)
	if err != nil {
		// Basic error mapping
		writeError(r.Context(), w, "Failed to approve member: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Member approved successfully"})
}

// RejectMember handles POST /api/v1/companies/{id}/members/{userId}/reject
// @Summary		Reject a member join request
// @Description	Rejects/Removes a pending member (Admin only)
// @Tags		Companies
// @Produce		json
// @Security	BearerAuth
// @Param		id		path		string	true	"Company ID"
// @Param		userId	path		string	true	"Target User ID"
// @Success		200		{object}	MessageResponse
// @Failure		403		{object}	ErrorResponse
// @Failure		404		{object}	ErrorResponse
// @Router		/api/v1/companies/{id}/members/{userId}/reject [post]
func (h *CompanyHandler) RejectMember(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "userId")
	if companyID == "" || targetUserID == "" {
		writeError(r.Context(), w, "company_id and user_id are required", http.StatusBadRequest)
		return
	}

	adminID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.companySvc.RejectMember(r.Context(), companyID, adminID, targetUserID)
	if err != nil {
		writeError(r.Context(), w, "Failed to reject member: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Member rejected successfully"})
}

// LookupCompany handles GET /api/v1/companies/lookup
// @Summary		Lookup company information
// @Description	Searches for a company by CNPJ or Name using external APIs and cache
// @Tags		Companies
// @Produce		json
// @Security	BearerAuth
// @Param		q	query		string	true	"Search query (CNPJ or Name)"
// @Success		200	{object}	map[string]interface{}
// @Failure		400	{object}	ErrorResponse
// @Failure		401	{object}	ErrorResponse
// @Failure		500	{object}	ErrorResponse
// @Router		/api/v1/companies/lookup [get]
func (h *CompanyHandler) LookupCompany(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" || len(query) < 3 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}})
		return
	}

	_, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID, _ := middleware.TenantIDFromContext(r.Context())

	searchLimit := 10
	var results []*company.CompanyInfo
	var err error

	results, err = h.companyLookupSvc.Lookup(r.Context(), tenantID, query, searchLimit)

	if err != nil && err != company.ErrCompanyInfoNotFound {
		writeError(r.Context(), w, "Failed to lookup company: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Map domain model to response matching the frontend preview
	type CompanyPreview struct {
		CNPJ          string `json:"cnpj"`
		RazaoSocial   string `json:"razao_social"`
		NomeFantasia  string `json:"nome_fantasia,omitempty"`
		Situacao      string `json:"situacao"`
		CNAEPrincipal string `json:"cnae_principal,omitempty"`
	}

	items := make([]CompanyPreview, 0, len(results))
	for _, res := range results {
		items = append(items, CompanyPreview{
			CNPJ:          res.CNPJ,
			RazaoSocial:   res.RazaoSocial,
			NomeFantasia:  res.NomeFantasia,
			Situacao:      string(res.Situacao),
			CNAEPrincipal: res.CNAEPrincipal,
		})
	}

	// Return empty array if items is nil to ensure proper JSON array
	if items == nil {
		items = []CompanyPreview{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}
