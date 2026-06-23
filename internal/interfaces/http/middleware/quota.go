// Package middleware contains HTTP middleware functions.
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

// QuotaPlan defines the limits for a billing plan.
type QuotaPlan struct {
	Name          string `json:"name"`
	AuthsPerMonth int64  `json:"auths_per_month"`
	UsersLimit    int64  `json:"users_limit"`
	RatePerMinute int64  `json:"rate_per_minute"`
}

// Default plans
var DefaultPlans = map[string]QuotaPlan{
	"free": {
		Name:          "Free",
		AuthsPerMonth: 1000,
		UsersLimit:    10,
		RatePerMinute: 60,
	},
	"pro": {
		Name:          "Pro",
		AuthsPerMonth: 50000,
		UsersLimit:    100,
		RatePerMinute: 300,
	},
	"enterprise": {
		Name:          "Enterprise",
		AuthsPerMonth: -1, // Unlimited
		UsersLimit:    -1, // Unlimited
		RatePerMinute: 1000,
	},
}

// QuotaEnforcer middleware enforces usage quotas per tenant.
type QuotaEnforcer struct {
	redis *redis.Client
	plans map[string]QuotaPlan
}

// NewQuotaEnforcer creates a new quota enforcer.
func NewQuotaEnforcer(redisClient *redis.Client) *QuotaEnforcer {
	return &QuotaEnforcer{
		redis: redisClient,
		plans: DefaultPlans,
	}
}

// WithPlans allows overriding default plans.
func (q *QuotaEnforcer) WithPlans(plans map[string]QuotaPlan) *QuotaEnforcer {
	q.plans = plans
	return q
}

// QuotaUsage represents current usage for a tenant.
type QuotaUsage struct {
	TenantID       string    `json:"tenant_id"`
	PlanName       string    `json:"plan_name"`
	AuthsThisMonth int64     `json:"auths_this_month"`
	AuthsLimit     int64     `json:"auths_limit"`
	UsersCount     int64     `json:"users_count"`
	UsersLimit     int64     `json:"users_limit"`
	ResetDate      time.Time `json:"reset_date"`
}

// Middleware returns the quota enforcement middleware.
// It checks the tenant's usage against their plan limits.
func (q *QuotaEnforcer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract tenant ID from context (set by auth middleware)
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			// No tenant context, skip quota check (might be health check, etc.)
			next.ServeHTTP(w, r)
			return
		}

		// Get tenant's plan
		plan := q.getTenantPlan(ctx, tenantID)

		// Check if unlimited (enterprise)
		if plan.AuthsPerMonth == -1 {
			next.ServeHTTP(w, r)
			return
		}

		// Get current usage
		usage, err := q.getCurrentUsage(ctx, tenantID)
		if err != nil {
			// Fail open if Redis is down
			next.ServeHTTP(w, r)
			return
		}

		// Check quota
		if usage >= plan.AuthsPerMonth {
			q.writeQuotaExceeded(w, plan, usage)
			return
		}

		// Increment usage counter
		q.incrementUsage(ctx, tenantID)

		// Add quota headers
		remaining := plan.AuthsPerMonth - usage - 1
		if remaining < 0 {
			remaining = 0
		}
		w.Header().Set("X-Quota-Limit", fmt.Sprintf("%d", plan.AuthsPerMonth))
		w.Header().Set("X-Quota-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-Quota-Reset", q.getResetDate().Format(time.RFC3339))

		next.ServeHTTP(w, r)
	})
}

// getTenantPlan retrieves the tenant's plan from Redis or returns default.
func (q *QuotaEnforcer) getTenantPlan(ctx context.Context, tenantID string) QuotaPlan {
	planKey := fmt.Sprintf("tenant:%s:plan", tenantID)
	planName, err := q.redis.Get(ctx, planKey).Result()
	if err != nil || planName == "" {
		return q.plans["free"] // Default to free plan
	}
	if plan, ok := q.plans[planName]; ok {
		return plan
	}
	return q.plans["free"]
}

// getCurrentUsage gets the current month's auth count.
func (q *QuotaEnforcer) getCurrentUsage(ctx context.Context, tenantID string) (int64, error) {
	monthKey := q.getMonthKey(tenantID)
	count, err := q.redis.Get(ctx, monthKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// incrementUsage increments the usage counter.
func (q *QuotaEnforcer) incrementUsage(ctx context.Context, tenantID string) {
	monthKey := q.getMonthKey(tenantID)
	pipe := q.redis.Pipeline()
	pipe.Incr(ctx, monthKey)
	// Expire at end of month + 1 day buffer
	pipe.ExpireAt(ctx, monthKey, q.getResetDate().Add(24*time.Hour))
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("Failed to increment quota usage for tenant %s: %v", tenantID, err)
	}
}

// getMonthKey returns the Redis key for this month's usage.
func (q *QuotaEnforcer) getMonthKey(tenantID string) string {
	now := time.Now()
	return fmt.Sprintf("quota:%s:%d-%02d", tenantID, now.Year(), now.Month())
}

// getResetDate returns the first day of next month.
func (q *QuotaEnforcer) getResetDate() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
}

// writeQuotaExceeded writes a 402 response when quota is exceeded.
func (q *QuotaEnforcer) writeQuotaExceeded(w http.ResponseWriter, plan QuotaPlan, usage int64) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Quota-Limit", fmt.Sprintf("%d", plan.AuthsPerMonth))
	w.Header().Set("X-Quota-Remaining", "0")
	w.Header().Set("X-Quota-Reset", q.getResetDate().Format(time.RFC3339))
	w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(q.getResetDate()).Seconds())))

	w.WriteHeader(http.StatusPaymentRequired)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"error":        "quota_exceeded",
		"message":      "Monthly authentication quota exceeded. Please upgrade your plan.",
		"current_plan": plan.Name,
		"usage":        usage,
		"limit":        plan.AuthsPerMonth,
		"reset_date":   q.getResetDate().Format(time.RFC3339),
	}); err != nil {
		log.Printf("Failed to write quota exceeded response: %v", err)
	}
}

// GetUsage returns the current usage for a tenant (for admin API).
func (q *QuotaEnforcer) GetUsage(ctx context.Context, tenantID string) (*QuotaUsage, error) {
	plan := q.getTenantPlan(ctx, tenantID)
	usage, err := q.getCurrentUsage(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &QuotaUsage{
		TenantID:       tenantID,
		PlanName:       plan.Name,
		AuthsThisMonth: usage,
		AuthsLimit:     plan.AuthsPerMonth,
		UsersLimit:     plan.UsersLimit,
		ResetDate:      q.getResetDate(),
	}, nil
}

// SetTenantPlan sets the plan for a tenant.
func (q *QuotaEnforcer) SetTenantPlan(ctx context.Context, tenantID, planName string) error {
	if _, ok := q.plans[planName]; !ok {
		return fmt.Errorf("unknown plan: %s", planName)
	}
	planKey := fmt.Sprintf("tenant:%s:plan", tenantID)
	return q.redis.Set(ctx, planKey, planName, 0).Err()
}

// AddActiveTenant adds a tenant to the set of active tenants.
func (q *QuotaEnforcer) AddActiveTenant(ctx context.Context, tenantID string) error {
	return q.redis.SAdd(ctx, "tenants:active", tenantID).Err()
}

// GetActiveTenants returns all active tenants.
func (q *QuotaEnforcer) GetActiveTenants(ctx context.Context) ([]string, error) {
	return q.redis.SMembers(ctx, "tenants:active").Result()
}

// QuotaHandler handles quota-related HTTP endpoints.
type QuotaHandler struct {
	enforcer *QuotaEnforcer
}

// NewQuotaHandler creates a new quota handler.
func NewQuotaHandler(enforcer *QuotaEnforcer) *QuotaHandler {
	return &QuotaHandler{enforcer: enforcer}
}

// GetUsage returns the usage for a tenant.
// GET /api/v1/tenants/{id}/usage
func (h *QuotaHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")
	if tenantID == "" {
		http.Error(w, `{"error": "tenant_id required"}`, http.StatusBadRequest)
		return
	}

	usage, err := h.enforcer.GetUsage(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error": "failed to get usage"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(usage); err != nil {
		GetLogger(r.Context()).Warn("Failed to write usage response", "tenant_id", tenantID, "error", err)
	}
}

// GetPlans returns available plans.
// GET /api/v1/plans
func (h *QuotaHandler) GetPlans(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.enforcer.plans); err != nil {
		GetLogger(r.Context()).Warn("Failed to write plans response", "error", err)
	}
}
