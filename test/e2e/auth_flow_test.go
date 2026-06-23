package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// E2E Test: Health Check Endpoints
// ============================================================================

func TestE2E_HealthEndpoint(t *testing.T) {
	healthHandler := handlers.NewHealthHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler.Health(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "healthy")
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestE2E_ReadyEndpoint(t *testing.T) {
	healthHandler := handlers.NewHealthHandler()

	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()

	healthHandler.Ready(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ready")
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

// ============================================================================
// E2E Test: JSON Response Validation
// ============================================================================

func TestE2E_HealthReturnsValidJSON(t *testing.T) {
	healthHandler := handlers.NewHealthHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler.Health(rec, req)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)

	require.NoError(t, err, "Response should be valid JSON")
	assert.Contains(t, response, "status")
}

func TestE2E_ReadyReturnsValidJSON(t *testing.T) {
	healthHandler := handlers.NewHealthHandler()

	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()

	healthHandler.Ready(rec, req)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)

	require.NoError(t, err, "Response should be valid JSON")
	assert.Contains(t, response, "status")
}

// ============================================================================
// E2E Test: Registration Validation
// ============================================================================

func TestE2E_RegisterEndpoint_InvalidJSON_Returns400(t *testing.T) {
	// Test with invalid JSON
	invalidJSON := []byte("{invalid json}")
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil)
	authHandler.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestE2E_LoginEndpoint_InvalidJSON_Returns400(t *testing.T) {
	invalidJSON := []byte("{invalid json}")
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	authHandler := handlers.NewAuthHandler(nil, nil, nil, nil, nil)
	authHandler.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// E2E Test: Content-Type Headers
// ============================================================================

func TestE2E_ContentTypeHeaders(t *testing.T) {
	healthHandler := handlers.NewHealthHandler()

	endpoints := []struct {
		name    string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"health", "/health", healthHandler.Health},
		{"ready", "/ready", healthHandler.Ready},
	}

	for _, ep := range endpoints {
		t.Run(ep.name+"_has_json_content_type", func(t *testing.T) {
			req := httptest.NewRequest("GET", ep.path, nil)
			rec := httptest.NewRecorder()

			ep.handler(rec, req)

			contentType := rec.Header().Get("Content-Type")
			assert.Equal(t, "application/json", contentType)
		})
	}
}
