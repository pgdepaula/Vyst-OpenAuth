// Package handlers contains HTTP handlers for the Identity API.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// StatsHandler handles dashboard statistics endpoints.
type StatsHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewStatsHandler creates a new StatsHandler.
func NewStatsHandler(db *pgxpool.Pool, redis *redis.Client) *StatsHandler {
	return &StatsHandler{
		db:    db,
		redis: redis,
	}
}

// DashboardStats represents the statistics for the admin dashboard.
type DashboardStats struct {
	TotalUsers     int64     `json:"total_users"`
	ActiveSessions int64     `json:"active_sessions"`
	RiskEvents24h  int64     `json:"risk_events_24h"`
	AuthsToday     int64     `json:"auths_today"`
	BlockedLogins  int64     `json:"blocked_logins"`
	LastUpdated    time.Time `json:"last_updated"`
}

// GetStats returns dashboard statistics.
// GET /api/v1/stats
func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats := DashboardStats{
		LastUpdated: time.Now(),
	}

	// Get total users count
	var err error
	stats.TotalUsers, err = h.getTotalUsers(ctx)
	if err != nil {
		stats.TotalUsers = 0 // Continue with 0 if error
	}

	// Get active sessions from Redis
	stats.ActiveSessions, err = h.getActiveSessions(ctx)
	if err != nil {
		stats.ActiveSessions = 0
	}

	// Get risk events in last 24h
	stats.RiskEvents24h, err = h.getRiskEvents24h(ctx)
	if err != nil {
		stats.RiskEvents24h = 0
	}

	// Get auths today from Redis counter
	stats.AuthsToday, err = h.getAuthsToday(ctx)
	if err != nil {
		stats.AuthsToday = 0
	}

	// Get blocked logins today
	stats.BlockedLogins, err = h.getBlockedLogins(ctx)
	if err != nil {
		stats.BlockedLogins = 0
	}

	writeJSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) getTotalUsers(ctx context.Context) (int64, error) {
	var count int64
	err := h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (h *StatsHandler) getActiveSessions(ctx context.Context) (int64, error) {
	// Count keys matching session pattern
	keys, err := h.redis.Keys(ctx, "session:*").Result()
	if err != nil {
		return 0, err
	}
	return int64(len(keys)), nil
}

func (h *StatsHandler) getRiskEvents24h(ctx context.Context) (int64, error) {
	// Count from outbox_events where type is risk-related in last 24h
	var count int64
	err := h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events 
		WHERE event_type LIKE '%Risk%' 
		AND created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&count)
	return count, err
}

func (h *StatsHandler) getAuthsToday(ctx context.Context) (int64, error) {
	today := time.Now().Format("2006-01-02")
	key := "stats:auths:" + today
	count, err := h.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

func (h *StatsHandler) getBlockedLogins(ctx context.Context) (int64, error) {
	today := time.Now().Format("2006-01-02")
	key := "stats:blocked:" + today
	count, err := h.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// IncrementAuthCounter increments the daily auth counter.
func (h *StatsHandler) IncrementAuthCounter(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")
	key := "stats:auths:" + today
	pipe := h.redis.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 48*time.Hour) // Keep for 48h
	_, err := pipe.Exec(ctx)
	return err
}

// IncrementBlockedCounter increments the daily blocked logins counter.
func (h *StatsHandler) IncrementBlockedCounter(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")
	key := "stats:blocked:" + today
	pipe := h.redis.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 48*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}
