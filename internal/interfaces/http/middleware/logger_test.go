package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/logger"
)

func TestRequestLogger(t *testing.T) {
	// Setup logger
	logger.SetGlobal(logger.New(logger.Config{Level: "debug", Format: "text"}))

	// Setup router
	r := chi.NewRouter()
	r.Use(RequestLogger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if logger is in context
		l := GetLogger(r.Context())
		if l == nil {
			t.Error("Logger not found in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Create request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Serve request
	r.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetLogger(t *testing.T) {
	// Test with no logger in context (should return global)
	req := httptest.NewRequest("GET", "/", nil)
	l := GetLogger(req.Context())
	if l == nil {
		t.Error("GetLogger returned nil for empty context")
	}
	if l != logger.Global() {
		t.Error("GetLogger did not return global logger for empty context")
	}
}
