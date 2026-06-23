// Package middleware contains HTTP middleware functions.
package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter provides rate limiting per IP address.
type RateLimiter struct {
	redis  *redis.Client
	limit  int
	window time.Duration
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(redisClient *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		limit:  limit,
		window: window,
	}
}

// Middleware returns the rate limiting middleware.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ip := r.RemoteAddr

		key := fmt.Sprintf("ratelimit:%s", ip)

		// Increment the counter
		count, err := rl.redis.Incr(ctx, key).Result()
		if err != nil {
			// If Redis fails, allow the request (fail open)
			next.ServeHTTP(w, r)
			return
		}

		// Set expiry on first request
		if count == 1 {
			rl.redis.Expire(ctx, key, rl.window)
		}

		// Check if limit exceeded
		if count > int64(rl.limit) {
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.window.Seconds())))
			http.Error(w, `{"error": "Rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rl.limit-int(count)))

		next.ServeHTTP(w, r)
	})
}
