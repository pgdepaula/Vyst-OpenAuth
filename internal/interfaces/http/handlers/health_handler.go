// Package handler contains HTTP request handlers.
package handlers

import (
	"context"
	"net/http"
)

// HealthChecker defines the contract for checking service health.
type HealthChecker interface {
	// PingDB checks if the database connection is healthy.
	PingDB(ctx context.Context) error
	// PingRedis checks if the Redis connection is healthy.
	PingRedis(ctx context.Context) error
}

// HealthHandler handles health check requests.
type HealthHandler struct {
	checker HealthChecker
}

// NewHealthHandler creates a new health handler.
// If checker is nil, readiness checks will always return ready (for backwards compatibility).
func NewHealthHandler(checker ...HealthChecker) *HealthHandler {
	var c HealthChecker
	if len(checker) > 0 {
		c = checker[0]
	}
	return &HealthHandler{checker: c}
}

// Health handles GET /health - basic liveness probe.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// Ready handles GET /ready - checks all dependencies.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// If no checker configured, assume ready (backwards compatible)
	if h.checker == nil {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ready",
		})
		return
	}

	// Check database connection
	if err := h.checker.PingDB(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"error":  "database connection failed",
		})
		return
	}

	// Check Redis connection
	if err := h.checker.PingRedis(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"error":  "redis connection failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}
