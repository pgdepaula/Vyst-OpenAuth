package ports

import (
	"context"
	"time"
)

// TempTokenStore abstracts temporary token storage (e.g., for 2FA flow).
// This allows the application layer to be independent of Redis or other storage implementations.
type TempTokenStore interface {
	// SaveString stores a string value with a TTL.
	SaveString(ctx context.Context, key, value string, ttl time.Duration) error

	// GetString retrieves a string value by key.
	GetString(ctx context.Context, key string) (string, error)

	// Delete removes a key from the store.
	Delete(ctx context.Context, key string) error
}
