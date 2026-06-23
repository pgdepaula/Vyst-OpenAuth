package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/redis/go-redis/v9"
)

// CachedDocumentVerificationPort is a decorator that caches verification results.
type CachedDocumentVerificationPort struct {
	delegate ports.DocumentVerificationPort
	client   *redis.Client
	ttl      time.Duration
	logger   ports.Logger
}

// NewCachedDocumentVerificationPort creates a new CachedDocumentVerificationPort.
// ttl defaults to 24 hours (86400 seconds) as per roadmap if not provided.
func NewCachedDocumentVerificationPort(delegate ports.DocumentVerificationPort, client *redis.Client, ttl time.Duration, logger ports.Logger) *CachedDocumentVerificationPort {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &CachedDocumentVerificationPort{
		delegate: delegate,
		client:   client,
		ttl:      ttl,
		logger:   logger,
	}
}

// VerifyCPF checks cache first, then calls delegate, then caches result.
func (c *CachedDocumentVerificationPort) VerifyCPF(ctx context.Context, cpf string) (*ports.DocumentVerificationResult, error) {
	key := fmt.Sprintf("document:verification:cpf:%s", cpf)

	// 1. Try Cache
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		var cachedResult ports.DocumentVerificationResult
		if err := json.Unmarshal([]byte(val), &cachedResult); err == nil {
			c.logger.Debug("CPF verification cache hit", "cpf", cpf)
			return &cachedResult, nil
		}
		// If unmarshal fails, log and proceed to re-fetch
		c.logger.Warn("Failed to unmarshal cached verification result", "key", key, "error", err)
	} else if err != redis.Nil {
		c.logger.Warn("Redis cache error during verification lookup", "key", key, "error", err)
	}

	// 2. Delegate (External API Call)
	result, err := c.delegate.VerifyCPF(ctx, cpf)
	if err != nil {
		return nil, err
	}

	// 3. Update Cache
	// We only cache if result is valid or if it's a "definitive" invalid.
	// Transient errors should not be cached (delegated usually shouldn't return result on error but error itself).

	marshaled, err := json.Marshal(result)
	if err != nil {
		c.logger.Error("Failed to marshal verification result for cache", "error", err)
		return result, nil
	}

	if err := c.client.Set(ctx, key, marshaled, c.ttl).Err(); err != nil {
		c.logger.Error("Redis set error for verification result", "key", key, "error", err)
	}

	return result, nil
}
