// Package middleware contains HTTP middleware functions.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/apikey"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	TenantKey    contextKey = "tenant_id"
	RolesKey     contextKey = "roles"
	UserAgentKey contextKey = "user_agent"
	IPAddressKey contextKey = "ip_address"
	ClaimsKey    contextKey = "claims"
)

// APIKeyValidator defines the contract for API key validation.
type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, rawKey string) (*apikey.APIKey, error)
}

// Auth returns a middleware that validates JWT tokens or API Keys.
func Auth(tokenSvc ports.TokenService, apiKeyValidator APIKeyValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Check for API Key
			apiKeyHeader := r.Header.Get("X-API-Key")
			if apiKeyHeader != "" {
				if apiKeyValidator == nil {
					http.Error(w, `{"error": "API Key authentication not supported"}`, http.StatusNotImplemented)
					return
				}
				key, err := apiKeyValidator.ValidateAPIKey(r.Context(), apiKeyHeader)
				if err != nil {
					http.Error(w, `{"error": "Invalid API Key"}`, http.StatusUnauthorized)
					return
				}

				// Add claims to context
				ctx := r.Context()
				ctx = context.WithValue(ctx, UserIDKey, key.UserID)
				ctx = context.WithValue(ctx, TenantKey, key.TenantID)
				ctx = context.WithValue(ctx, RolesKey, key.Scopes) // Map scopes to roles for now

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// 2. Check for JWT Token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "Authorization header or X-API-Key required"}`, http.StatusUnauthorized)
				return
			}

			// Parse "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error": "Invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validate token
			claims, err := tokenSvc.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, TenantKey, claims.TenantID)
			ctx = context.WithValue(ctx, RolesKey, claims.Roles)
			ctx = context.WithValue(ctx, ClaimsKey, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the user ID from context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// TenantIDFromContext extracts the tenant ID from context.
func TenantIDFromContext(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(TenantKey).(string)
	return tenantID, ok
}

// RolesFromContext extracts the user roles from context.
func RolesFromContext(ctx context.Context) []string {
	roles, ok := ctx.Value(RolesKey).([]string)
	if !ok {
		return []string{}
	}
	return roles
}

// UserAgentFromContext extracts the user agent from context.
func UserAgentFromContext(ctx context.Context) string {
	ua, ok := ctx.Value(UserAgentKey).(string)
	if !ok {
		return "unknown"
	}
	return ua
}

// IPAddressFromContext extracts the IP address from context.
func IPAddressFromContext(ctx context.Context) string {
	ip, ok := ctx.Value(IPAddressKey).(string)
	if !ok {
		return "unknown"
	}
	return ip
}

// HasRole checks if the user has a specific role.
func HasRole(ctx context.Context, role string) bool {
	roles := RolesFromContext(ctx)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsSuperAdmin checks if the user has the "super_admin" role.
func IsSuperAdmin(ctx context.Context) bool {
	return HasRole(ctx, "super_admin")
}
