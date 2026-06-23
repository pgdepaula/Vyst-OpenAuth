package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
)

type APIKeyHandler struct {
	apiKeySvc *service.APIKeyService
}

func NewAPIKeyHandler(apiKeySvc *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{apiKeySvc: apiKeySvc}
}

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

type CreateAPIKeyResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Prefix    string `json:"prefix"`
	RawKey    string `json:"raw_key"` // Only returned here!
	CreatedAt string `json:"created_at"`
}

func (h *APIKeyHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeError(r.Context(), w, "Name is required", http.StatusBadRequest)
		return
	}

	generatedKey, err := h.apiKeySvc.CreateAPIKey(r.Context(), userID, tenantID, req.Name)
	if err != nil {
		writeError(r.Context(), w, "Failed to create API key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, CreateAPIKeyResponse{
		ID:        generatedKey.APIKey.ID,
		Name:      generatedKey.APIKey.Name,
		Prefix:    generatedKey.APIKey.KeyPrefix,
		RawKey:    generatedKey.RawKey,
		CreatedAt: generatedKey.APIKey.CreatedAt.String(),
	})
}

func (h *APIKeyHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	keys, err := h.apiKeySvc.ListAPIKeys(r.Context(), tenantID)
	if err != nil {
		writeError(r.Context(), w, "Failed to list API keys", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, keys)
}

func (h *APIKeyHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, "ID is required", http.StatusBadRequest)
		return
	}

	if err := h.apiKeySvc.RevokeAPIKey(r.Context(), id); err != nil {
		writeError(r.Context(), w, "Failed to revoke API key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
