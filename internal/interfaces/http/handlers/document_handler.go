package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pgdepaula/vyst-openauth/internal/application/service"
)

// DocumentHandler handles document-related requests.
type DocumentHandler struct {
	documentSvc *service.DocumentService
}

// NewDocumentHandler creates a new document handler.
func NewDocumentHandler(documentSvc *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{
		documentSvc: documentSvc,
	}
}

// ValidateCPFRequest represents the request to validate a CPF.
type ValidateCPFRequest struct {
	CPF string `json:"cpf"`
}

// ValidateCPFResponse represents the response of CPF validation.
type ValidateCPFResponse struct {
	Valid     bool   `json:"valid"`
	Formatted string `json:"formatted,omitempty"`
	Message   string `json:"message,omitempty"`
}

// ValidateCPF handles POST /documents/validate-cpf
// @Summary		Validate a CPF
// @Description	Checks if a CPF is valid and returning its formatted version if so.
// @Tags		Documents
// @Accept		json
// @Produce		json
// @Param		body	body		ValidateCPFRequest	true	"CPF to validate"
// @Success		200		{object}	ValidateCPFResponse
// @Failure		400		{object}	ErrorResponse
// @Router		/documents/validate-cpf [post]
func (h *DocumentHandler) ValidateCPF(w http.ResponseWriter, r *http.Request) {
	var req ValidateCPFRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CPF == "" {
		writeError(r.Context(), w, "CPF is required", http.StatusBadRequest)
		return
	}

	cpfVO, err := h.documentSvc.ValidateAndNormalizeCPF(r.Context(), req.CPF)
	if err != nil {
		// We return 200 OK even for invalid CPF, but with Valid: false
		writeJSON(w, http.StatusOK, ValidateCPFResponse{
			Valid:   false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, ValidateCPFResponse{
		Valid:     true,
		Formatted: cpfVO.String(),
	})
}
