package ports

import "context"

// RateLimiter defines the contract for rate limiting operations.
type RateLimiter interface {
	// Allow checks if the current request is allowed for the given key (e.g., target feature + tenantID).
	// Returns true if allowed, false if rate limit is exceeded.
	Allow(ctx context.Context, key string) (bool, error)
}
