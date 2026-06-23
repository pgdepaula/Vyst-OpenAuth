package checks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pgdepaula/vyst-openauth/cmd/verify/config"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/runner"
)

// RESTChecks returns REST API endpoint verification checks.
func RESTChecks(authCtx *AuthContext) []runner.Check {
	return []runner.Check{
		// Roles
		{
			Name:  "POST /api/v1/roles (create)",
			Group: "REST API",
			Fn:    makeCreateRoleCheck(authCtx),
		},
		{
			Name:  "GET /api/v1/roles (list)",
			Group: "REST API",
			Fn:    makeListRolesCheck(authCtx),
		},
		// Tenants
		{
			Name:  "POST /api/v1/tenants (create)",
			Group: "REST API",
			Fn:    makeCreateTenantCheck(authCtx),
		},
		// API Keys
		{
			Name:  "POST /api/v1/api-keys (create)",
			Group: "REST API",
			Fn:    makeCreateAPIKeyCheck(authCtx),
		},
		{
			Name:  "GET /api/v1/api-keys (list)",
			Group: "REST API",
			Fn:    makeListAPIKeysCheck(authCtx),
		},
		// Stats
		{
			Name:  "GET /api/v1/stats",
			Group: "REST API",
			Fn:    makeStatsCheck(authCtx),
		},
	}
}

func makeCreateRoleCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "POST /api/v1/roles",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		payload := map[string]interface{}{
			"name":        fmt.Sprintf("test_role_%d", time.Now().UnixNano()),
			"description": "Test role created by verify CLI",
			"permissions": []string{"read:test"},
		}

		resp, body, err := doPost(ctx, cfg, "/api/v1/roles", authCtx.Token, payload)
		if err != nil {
			return &runner.CheckResult{
				Name:     "POST /api/v1/roles",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "POST /api/v1/roles",
				Group:      "REST API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 201, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "POST /api/v1/roles",
			Group:      "REST API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeListRolesCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "GET /api/v1/roles",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		resp, body, err := doGet(ctx, cfg, "/api/v1/roles", authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     "GET /api/v1/roles",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "GET /api/v1/roles",
				Group:      "REST API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "GET /api/v1/roles",
			Group:      "REST API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeCreateTenantCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "POST /api/v1/tenants",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		payload := map[string]string{
			"name": fmt.Sprintf("tenant_%d", time.Now().UnixNano()),
		}

		resp, body, err := doPost(ctx, cfg, "/api/v1/tenants", authCtx.Token, payload)
		if err != nil {
			return &runner.CheckResult{
				Name:     "POST /api/v1/tenants",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "POST /api/v1/tenants",
				Group:      "REST API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 201, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "POST /api/v1/tenants",
			Group:      "REST API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeCreateAPIKeyCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "POST /api/v1/api-keys",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		payload := map[string]string{
			"name": fmt.Sprintf("verify_key_%d", time.Now().UnixNano()),
		}

		resp, body, err := doPost(ctx, cfg, "/api/v1/api-keys", authCtx.Token, payload)
		if err != nil {
			return &runner.CheckResult{
				Name:     "POST /api/v1/api-keys",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "POST /api/v1/api-keys",
				Group:      "REST API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 201, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "POST /api/v1/api-keys",
			Group:      "REST API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeListAPIKeysCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "GET /api/v1/api-keys",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		resp, body, err := doGet(ctx, cfg, "/api/v1/api-keys", authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     "GET /api/v1/api-keys",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "GET /api/v1/api-keys",
				Group:      "REST API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "GET /api/v1/api-keys",
			Group:      "REST API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeStatsCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "GET /api/v1/stats",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		resp, body, err := doGet(ctx, cfg, "/api/v1/stats", authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     "GET /api/v1/stats",
				Group:    "REST API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "GET /api/v1/stats",
				Group:      "REST API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "GET /api/v1/stats",
			Group:      "REST API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}
