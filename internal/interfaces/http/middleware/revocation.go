// Package middleware contains HTTP middleware functions.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
)

// RevocationStore defines the interface for checking if a token is revoked.
type RevocationStore interface {
	IsRevoked(ctx context.Context, tokenID string) (bool, error)
}

// InMemoryRevocationStore is a simple in-memory implementation for testing.
type InMemoryRevocationStore struct {
	revokedTokens map[string]struct{}
}

// NewInMemoryRevocationStore creates a new in-memory revocation store.
func NewInMemoryRevocationStore() *InMemoryRevocationStore {
	return &InMemoryRevocationStore{
		revokedTokens: make(map[string]struct{}),
	}
}

// IsRevoked checks if a token is in the revocation list.
func (s *InMemoryRevocationStore) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	_, revoked := s.revokedTokens[tokenID]
	return revoked, nil
}

// Revoke adds a token to the revocation list.
func (s *InMemoryRevocationStore) Revoke(tokenID string) {
	s.revokedTokens[tokenID] = struct{}{}
}

// Revocation returns a middleware that checks if the token is revoked.
func Revocation(tokenSvc ports.TokenService, store RevocationStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := tokenSvc.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Check revocation
			revoked, err := store.IsRevoked(r.Context(), claims.ID)
			if err != nil {
				http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
				return
			}
			if revoked {
				http.Error(w, `{"error": "Token revoked"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
