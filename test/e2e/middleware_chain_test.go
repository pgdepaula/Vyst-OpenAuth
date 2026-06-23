package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// E2E Test: Middleware Chain
// ============================================================================

func TestE2E_MiddlewareChain_SecurityHeaders(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		assert.NoError(t, err)
	})

	// Apply security headers middleware
	protected := middleware.SecurityHeaders(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	// Verify all security headers are set
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.Contains(t, rec.Header().Get("Strict-Transport-Security"), "max-age=")
}

func TestE2E_MiddlewareChain_AuthWithToken(t *testing.T) {
	mockToken := &mocks.MockTokenService{
		ValidateTokenFunc: func(tokenString string) (*ports.Claims, error) {
			return &ports.Claims{
				UserID:   "user-123",
				TenantID: "tenant-456",
				Roles:    []string{"admin"},
			}, nil
		},
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		// Verify context has user info
		userID, ok := middleware.UserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "user-123", userID)
		w.WriteHeader(http.StatusOK)
	})

	authMiddleware := middleware.Auth(mockToken, nil)
	protected := authMiddleware(handler)

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer valid.jwt.token")
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestE2E_MiddlewareChain_RateLimiting(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	limiter := middleware.NewRateLimiter(client, 3, time.Minute)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protected := limiter.Middleware(handler)

	// Make 3 requests (should all pass)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "Request %d should succeed", i+1)
	}

	// 4th request should be blocked
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

// ============================================================================
// E2E Test: Complete Protected Endpoint Flow
// ============================================================================

func TestE2E_ProtectedEndpoint_CompleteFlow(t *testing.T) {
	// Setup: Rate limiter + Auth + Security Headers + Handler
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	mockToken := &mocks.MockTokenService{
		ValidateTokenFunc: func(tokenString string) (*ports.Claims, error) {
			if tokenString == "valid.token" {
				return &ports.Claims{
					UserID:   "user-123",
					TenantID: "tenant-456",
					Roles:    []string{"admin"},
				}, nil
			}
			return nil, context.DeadlineExceeded
		},
	}

	// Final handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := middleware.UserIDFromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"user":"` + userID + `"}`))
		assert.NoError(t, err)
	})

	// Chain middlewares
	limiter := middleware.NewRateLimiter(client, 100, time.Minute)
	authMiddleware := middleware.Auth(mockToken, nil)

	protected := middleware.SecurityHeaders(
		limiter.Middleware(
			authMiddleware(handler),
		),
	)

	t.Run("Valid_Token_Passes_All_Middleware", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user", nil)
		req.Header.Set("Authorization", "Bearer valid.token")
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Contains(t, rec.Body.String(), "user-123")
	})

	t.Run("No_Token_Blocked_By_Auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/user", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		// Security headers should still be present
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	})
}

// ============================================================================
// E2E Test: Multi-Tenant Isolation
// ============================================================================

func TestE2E_MultiTenant_UserIsolation(t *testing.T) {
	mockToken := &mocks.MockTokenService{
		ValidateTokenFunc: func(tokenString string) (*ports.Claims, error) {
			switch tokenString {
			case "tenant-a-token":
				return &ports.Claims{UserID: "user-a", TenantID: "tenant-A", Roles: []string{"user"}}, nil
			case "tenant-b-token":
				return &ports.Claims{UserID: "user-b", TenantID: "tenant-B", Roles: []string{"user"}}, nil
			default:
				return nil, context.DeadlineExceeded
			}
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantIDFromContext(r.Context())
		_, err := w.Write([]byte(tenantID))
		assert.NoError(t, err)
	})

	authMiddleware := middleware.Auth(mockToken, nil)
	protected := authMiddleware(handler)

	t.Run("TenantA_Gets_Own_Context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/data", nil)
		req.Header.Set("Authorization", "Bearer tenant-a-token")
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		assert.Equal(t, "tenant-A", rec.Body.String())
	})

	t.Run("TenantB_Gets_Own_Context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/data", nil)
		req.Header.Set("Authorization", "Bearer tenant-b-token")
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		assert.Equal(t, "tenant-B", rec.Body.String())
	})
}
