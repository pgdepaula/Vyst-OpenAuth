package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return client, func() {
		_ = client.Close()
		mr.Close()
	}
}

func TestQuotaEnforcer_AllowsUnderLimit(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	enforcer := NewQuotaEnforcer(redisClient)

	// Handler that gets called if quota passes
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")
	rec := httptest.NewRecorder()

	enforcer.Middleware(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("X-Quota-Limit"), "1000") // Free plan
}

func TestQuotaEnforcer_BlocksOverLimit(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	enforcer := NewQuotaEnforcer(redisClient)
	ctx := context.Background()

	// Pre-fill quota to limit
	monthKey := enforcer.getMonthKey("test-tenant")
	redisClient.Set(ctx, monthKey, 1000, 0) // At limit

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when over limit")
	})

	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")
	rec := httptest.NewRecorder()

	enforcer.Middleware(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusPaymentRequired, rec.Code)
	assert.Contains(t, rec.Body.String(), "quota_exceeded")
}

func TestQuotaEnforcer_EnterprisePlanUnlimited(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	enforcer := NewQuotaEnforcer(redisClient)
	ctx := context.Background()

	// Set tenant to enterprise plan
	redisClient.Set(ctx, "tenant:test-tenant:plan", "enterprise", 0)

	// Pre-fill huge usage
	monthKey := enforcer.getMonthKey("test-tenant")
	redisClient.Set(ctx, monthKey, 999999, 0)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")
	rec := httptest.NewRecorder()

	enforcer.Middleware(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQuotaEnforcer_GetUsage(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	enforcer := NewQuotaEnforcer(redisClient)
	ctx := context.Background()

	// Set some usage
	monthKey := enforcer.getMonthKey("test-tenant")
	redisClient.Set(ctx, monthKey, 500, 0)

	usage, err := enforcer.GetUsage(ctx, "test-tenant")

	require.NoError(t, err)
	assert.Equal(t, "test-tenant", usage.TenantID)
	assert.Equal(t, int64(500), usage.AuthsThisMonth)
	assert.Equal(t, int64(1000), usage.AuthsLimit) // Free plan
	assert.Equal(t, "Free", usage.PlanName)
}

func TestQuotaEnforcer_SetTenantPlan(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	enforcer := NewQuotaEnforcer(redisClient)
	ctx := context.Background()

	err := enforcer.SetTenantPlan(ctx, "test-tenant", "pro")
	require.NoError(t, err)

	usage, _ := enforcer.GetUsage(ctx, "test-tenant")
	assert.Equal(t, "Pro", usage.PlanName)
	assert.Equal(t, int64(50000), usage.AuthsLimit)
}

func TestQuotaEnforcer_NoTenantSkipsCheck(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	enforcer := NewQuotaEnforcer(redisClient)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	// No X-Tenant-ID header
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	enforcer.Middleware(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
}

func TestQuotaEnforcer_ResetDate(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	enforcer := NewQuotaEnforcer(redisClient)

	resetDate := enforcer.getResetDate()
	now := time.Now()

	// Reset date should be first of next month
	assert.Equal(t, 1, resetDate.Day())
	expectedMonth := now.Month() + 1
	if expectedMonth > 12 {
		expectedMonth = 1
	}
	assert.Equal(t, expectedMonth, resetDate.Month())
}
