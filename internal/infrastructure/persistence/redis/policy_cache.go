package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"github.com/redis/go-redis/v9"
)

// CachedPolicyRepository is a decorator that adds caching to a PolicyRepository.
type CachedPolicyRepository struct {
	delegate policy.Repository
	client   *redis.Client
	ttl      time.Duration
}

// NewCachedPolicyRepository creates a new CachedPolicyRepository.
func NewCachedPolicyRepository(delegate policy.Repository, client *redis.Client, ttl time.Duration) *CachedPolicyRepository {
	return &CachedPolicyRepository{
		delegate: delegate,
		client:   client,
		ttl:      ttl,
	}
}

func (r *CachedPolicyRepository) Check(ctx context.Context, subject, relation, object string) (bool, error) {
	key := fmt.Sprintf("policy:check:%s:%s:%s", subject, relation, object)

	// 1. Check Cache
	val, err := r.client.Get(ctx, key).Result()
	if err == nil {
		// Hit
		return val == "1", nil
	} else if err != redis.Nil {
		// Log error but continue to delegate (fail open/safe depending on strategy, here fail safe -> delegate)
		slog.Error("Redis cache error", "error", err)
	}

	// 2. Miss - Call Delegate
	allowed, err := r.delegate.Check(ctx, subject, relation, object)
	if err != nil {
		return false, err
	}

	// 3. Set Cache
	cacheVal := "0"
	if allowed {
		cacheVal = "1"
	}
	// Fire and forget cache set to not block
	go func() {
		// Create a detached context for the async operation
		_ = r.client.Set(context.Background(), key, cacheVal, r.ttl).Err()
	}()

	return allowed, nil
}

func (r *CachedPolicyRepository) WriteTuple(ctx context.Context, tuple policy.Tuple) error {
	// 1. Write to Delegate
	if err := r.delegate.WriteTuple(ctx, tuple); err != nil {
		return err
	}

	// 2. Invalidate Cache?
	// Invalidation is hard because one tuple can affect many checks (transitive).
	// We rely on TTL (30s) for eventual consistency.
	// We COULD invalidate the direct check key, but that's minimal help.
	// key := fmt.Sprintf("policy:check:%s:%s:%s", tuple.Subject, tuple.Relation, tuple.Object)
	// r.client.Del(ctx, key)

	return nil
}

func (r *CachedPolicyRepository) DeleteTuple(ctx context.Context, tuple policy.Tuple) error {
	if err := r.delegate.DeleteTuple(ctx, tuple); err != nil {
		return err
	}
	return nil
}

func (r *CachedPolicyRepository) GetRolesForUser(ctx context.Context, userID string) ([]string, error) {
	// For now, just delegate without caching as roles might change and are critical
	return r.delegate.GetRolesForUser(ctx, userID)
}
