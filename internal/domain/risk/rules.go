package risk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// VelocityRule checks for high frequency logins.
type VelocityRule struct {
	redisClient *redis.Client
	limit       int
	window      time.Duration
}

func NewVelocityRule(redisClient *redis.Client, limit int, window time.Duration) *VelocityRule {
	return &VelocityRule{
		redisClient: redisClient,
		limit:       limit,
		window:      window,
	}
}

func (r *VelocityRule) Name() string {
	return "Velocity Check"
}

func (r *VelocityRule) Evaluate(ctx context.Context, userID uuid.UUID, ip string, userAgent string) (float64, string, error) {
	key := fmt.Sprintf("risk:velocity:%s", userID.String())

	// Increment counter
	count, err := r.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return 0, "", fmt.Errorf("redis error: %w", err)
	}

	// Set expiration on first increment
	if count == 1 {
		r.redisClient.Expire(ctx, key, r.window)
	}

	if count > int64(r.limit) {
		return 1.0, fmt.Sprintf("Exceeded login limit: %d in %s", count, r.window), nil
	}

	return 0, "", nil
}

// ImpossibleTravelRule checks for physically impossible travel between logins.
type ImpossibleTravelRule struct {
	historyRepo LoginHistoryRepository
}

func NewImpossibleTravelRule(historyRepo LoginHistoryRepository) *ImpossibleTravelRule {
	return &ImpossibleTravelRule{historyRepo: historyRepo}
}

func (r *ImpossibleTravelRule) Name() string {
	return "Impossible Travel"
}

func (r *ImpossibleTravelRule) Evaluate(ctx context.Context, userID uuid.UUID, ip string, userAgent string) (float64, string, error) {
	lastLogin, err := r.historyRepo.GetLastLogin(ctx, userID)
	if err != nil {
		return 0, "", err
	}
	if lastLogin == nil {
		return 0, "", nil // First login, no history
	}

	// If IPs are different, check time difference
	if lastLogin.IPAddress != ip {
		// Simple heuristic: If different IP and < 1 second, it's suspicious.
		// In a real system, we'd use GeoIP to calculate distance / time.
		// For this MVP, we assume any IP change in < 5 seconds is impossible travel (e.g. VPN hopping or bot).

		timeDiff := time.Since(lastLogin.LoginAt)
		if timeDiff < 5*time.Second {
			return 1.0, fmt.Sprintf("Impossible travel detected: IP changed from %s to %s in %v", lastLogin.IPAddress, ip, timeDiff), nil
		}
	}

	return 0, "", nil
}
