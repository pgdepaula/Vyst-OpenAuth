package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Login(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/login", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(LoginResponse{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		}))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Login(context.Background(), "test@example.com", "password123")

	require.NoError(t, err)
	assert.Equal(t, "test-access-token", client.GetAccessToken())
	assert.Equal(t, "test-refresh-token", client.GetRefreshToken())
	assert.True(t, client.IsAuthenticated())
}

func TestClient_RefreshIfNeeded(t *testing.T) {
	refreshCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/refresh" {
			refreshCalled = true
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(LoginResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
			}))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL,
		WithTokens("old-token", "old-refresh"),
		WithRefreshBuffer(1*time.Hour), // Force refresh
	)
	// Set expiresAt to past
	client.expiresAt = time.Now().Add(-1 * time.Hour)

	err := client.RefreshIfNeeded(context.Background())

	require.NoError(t, err)
	assert.True(t, refreshCalled)
	assert.Equal(t, "new-access-token", client.GetAccessToken())
}

func TestClient_Can(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/authz/check" {
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(PermissionCheckResponse{
				Allowed: true,
				Reason:  "User has role admin",
			}))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL,
		WithTokens("valid-token", "refresh-token"),
		WithAutoRefresh(false),
	)
	client.expiresAt = time.Now().Add(1 * time.Hour)

	allowed, err := client.Can(context.Background(), "user-123", "edit", "invoice:456")

	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestClient_ValidateToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/introspect" {
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(Claims{
				UserID:   "user-123",
				TenantID: "tenant-456",
				Email:    "test@example.com",
				Roles:    []string{"admin", "user"},
				Exp:      time.Now().Add(1 * time.Hour).Unix(),
			}))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	claims, err := client.ValidateToken(context.Background(), "some-token")

	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "tenant-456", claims.TenantID)
	assert.Contains(t, claims.Roles, "admin")
}

func TestClient_Logout(t *testing.T) {
	client := NewClient("http://localhost",
		WithTokens("token", "refresh"),
	)
	client.expiresAt = time.Now().Add(1 * time.Hour)

	assert.True(t, client.IsAuthenticated())

	client.Logout()

	assert.False(t, client.IsAuthenticated())
	assert.Empty(t, client.GetAccessToken())
}

func TestClient_OnTokenRefresh(t *testing.T) {
	var savedAccess, savedRefresh string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(LoginResponse{
			AccessToken:  "callback-access",
			RefreshToken: "callback-refresh",
			ExpiresIn:    3600,
		}))
	}))
	defer server.Close()

	client := NewClient(server.URL,
		WithTokens("old", "old-refresh"),
		WithOnTokenRefresh(func(access, refresh string) {
			savedAccess = access
			savedRefresh = refresh
		}),
	)
	client.expiresAt = time.Now().Add(-1 * time.Hour)

	_ = client.RefreshIfNeeded(context.Background())

	assert.Equal(t, "callback-access", savedAccess)
	assert.Equal(t, "callback-refresh", savedRefresh)
}
