package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/apikey"
)

type APIKeyService struct {
	repo   apikey.Repository
	hasher ports.PasswordHasher
}

func NewAPIKeyService(repo apikey.Repository, hasher ports.PasswordHasher) *APIKeyService {
	return &APIKeyService{
		repo:   repo,
		hasher: hasher,
	}
}

// CreateAPIKey generates a new API key, hashes it, and stores it.
func (s *APIKeyService) CreateAPIKey(ctx context.Context, userID, tenantID, name string) (*apikey.GeneratedKey, error) {
	// Generate random key
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Format: vyst_{base64}
	// Using RawURLEncoding to avoid + and / which might be annoying in URLs/headers
	rawKey := "vyst_" + base64.RawURLEncoding.EncodeToString(randomBytes)

	// Extract prefix (e.g., first 12 chars including 'vyst_')
	// vyst_ + 8 chars = 13 chars
	prefix := rawKey[:13]

	// Hash the key
	hashedKey, err := s.hasher.Hash(rawKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash key: %w", err)
	}

	key := &apikey.APIKey{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		UserID:    userID,
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   hashedKey,
		Scopes:    []string{"all"}, // Default scope for now
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return nil, err
	}

	return &apikey.GeneratedKey{
		APIKey: key,
		RawKey: rawKey,
	}, nil
}

// ValidateAPIKey checks if the provided key is valid.
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, rawKey string) (*apikey.APIKey, error) {
	if !strings.HasPrefix(rawKey, "vyst_") {
		return nil, errors.New("invalid key format")
	}

	if len(rawKey) < 13 {
		return nil, errors.New("invalid key length")
	}

	prefix := rawKey[:13]
	key, err := s.repo.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, errors.New("invalid api key") // Don't leak if it was DB error or not found
	}

	if !s.hasher.Verify(rawKey, key.KeyHash) {
		return nil, errors.New("invalid api key")
	}

	// Update last used asynchronously (fire and forget)
	// In a real high-throughput system, we'd buffer this or use Redis
	go func() {
		_ = s.repo.UpdateLastUsed(context.Background(), key.ID)
	}()

	return key, nil
}

func (s *APIKeyService) ListAPIKeys(ctx context.Context, tenantID string) ([]*apikey.APIKey, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

func (s *APIKeyService) RevokeAPIKey(ctx context.Context, id string) error {
	return s.repo.Revoke(ctx, id)
}
