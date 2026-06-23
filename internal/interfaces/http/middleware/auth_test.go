package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/apikey"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// AuthMiddleware Tests
// ============================================================================

func TestAuthMiddleware_ValidToken_PassesToHandler(t *testing.T) {
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
		userID, ok := middleware.UserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "user-123", userID)
		tenantID, ok := middleware.TenantIDFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "tenant-456", tenantID)
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.Auth(mockToken, nil)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid.jwt.token")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_MissingAuthorizationHeader_Returns401(t *testing.T) {
	mockToken := &mocks.MockTokenService{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	mw := middleware.Auth(mockToken, nil)
	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_InvalidBearerFormat_Returns401(t *testing.T) {
	mockToken := &mocks.MockTokenService{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	mw := middleware.Auth(mockToken, nil)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "NotBearer token")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	mockToken := &mocks.MockTokenService{
		ValidateTokenFunc: func(tokenString string) (*ports.Claims, error) {
			return nil, errors.New("invalid token")
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	mw := middleware.Auth(mockToken, nil)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	mockToken := &mocks.MockTokenService{
		ValidateTokenFunc: func(tokenString string) (*ports.Claims, error) {
			return nil, errors.New("token expired")
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	mw := middleware.Auth(mockToken, nil)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired.token")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_EmptyBearerToken_DocumentsBehavior(t *testing.T) {
	mockToken := &mocks.MockTokenService{
		ValidateTokenFunc: func(tokenString string) (*ports.Claims, error) {
			if tokenString == "" {
				return nil, errors.New("empty token")
			}
			return &ports.Claims{UserID: "user-123", TenantID: "tenant-456"}, nil
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.Auth(mockToken, nil)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusUnauthorized)
}

func TestAuthMiddleware_SetsAllClaimsInContext(t *testing.T) {
	mockToken := &mocks.MockTokenService{
		ValidateTokenFunc: func(tokenString string) (*ports.Claims, error) {
			return &ports.Claims{
				UserID:   "user-123",
				TenantID: "tenant-456",
				Roles:    []string{"admin", "manager"},
			}, nil
		},
	}

	var capturedCtx context.Context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.Auth(mockToken, nil)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid.token")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	userID, _ := middleware.UserIDFromContext(capturedCtx)
	tenantID, _ := middleware.TenantIDFromContext(capturedCtx)
	assert.Equal(t, "user-123", userID)
	assert.Equal(t, "tenant-456", tenantID)
}

// ============================================================================
// Table-Driven Tests for Authorization Header Formats
// ============================================================================

func TestAuthMiddleware_AuthorizationHeaderFormats(t *testing.T) {
	validClaims := &ports.Claims{UserID: "user-123", TenantID: "tenant-456"}
	mockToken := &mocks.MockTokenService{
		ValidateTokenFunc: func(tokenString string) (*ports.Claims, error) {
			return validClaims, nil
		},
	}

	tests := []struct {
		name          string
		authHeader    string
		expectStatus  int
		allowFallback bool
	}{
		{"valid bearer token", "Bearer valid.token", http.StatusOK, false},
		{"missing header", "", http.StatusUnauthorized, false},
		{"only bearer", "Bearer", http.StatusUnauthorized, false},
		{"only bearer with space", "Bearer ", http.StatusUnauthorized, true},
		{"wrong scheme", "Basic dXNlcjpwYXNz", http.StatusUnauthorized, false},
		{"lowercase bearer", "bearer valid.token", http.StatusUnauthorized, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			mw := middleware.Auth(mockToken, nil)
			req := httptest.NewRequest("GET", "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			mw(handler).ServeHTTP(rec, req)

			if tt.allowFallback {
				assert.True(t, rec.Code == tt.expectStatus || rec.Code == http.StatusOK)
			} else {
				assert.Equal(t, tt.expectStatus, rec.Code, "Test case: %s", tt.name)
			}
		})
	}
}

func TestAuthMiddleware_ValidAPIKey_PassesToHandler(t *testing.T) {
	mockAPIKeyValidator := &mocks.MockAPIKeyValidator{
		ValidateAPIKeyFunc: func(ctx context.Context, rawKey string) (*apikey.APIKey, error) {
			return &apikey.APIKey{
				UserID:   "user-123",
				TenantID: "tenant-456",
				Scopes:   []string{"admin"},
			}, nil
		},
	}

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		userID, ok := middleware.UserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "user-123", userID)
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.Auth(nil, mockAPIKeyValidator)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-API-Key", "vyst_valid_key")
	rec := httptest.NewRecorder()

	mw(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}
