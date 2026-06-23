package apikey

import (
	"context"
	"time"
)

// APIKey represents an API key for programmatic access.
type APIKey struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	KeyHash    string     `json:"-"` // Never return the hash
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// GeneratedKey represents a newly created key with the raw secret.
type GeneratedKey struct {
	APIKey *APIKey `json:"api_key"`
	RawKey string  `json:"raw_key"` // Shown only once
}

// Repository defines the interface for API key persistence.
type Repository interface {
	Create(ctx context.Context, key *APIKey) error
	GetByPrefix(ctx context.Context, prefix string) (*APIKey, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*APIKey, error)
	Revoke(ctx context.Context, id string) error
	UpdateLastUsed(ctx context.Context, id string) error
}
