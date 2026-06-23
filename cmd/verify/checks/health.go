// Package checks provides individual verification checks for the systemd.
package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pgdepaula/vyst-openauth/cmd/verify/config"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/runner"
)

// HealthChecks returns all health-related verification checks.
func HealthChecks() []runner.Check {
	return []runner.Check{
		{
			Name:  "GET /health",
			Group: "Health",
			Fn:    checkHealth,
		},
		{
			Name:  "GET /ready",
			Group: "Health",
			Fn:    checkReady,
		},
	}
}

func checkHealth(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
	return doHealthCheck(ctx, cfg, "/health", "healthy")
}

func checkReady(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
	return doHealthCheck(ctx, cfg, "/ready", "ready")
}

func doHealthCheck(ctx context.Context, cfg *config.Config, path, expectedStatus string) (*runner.CheckResult, error) {
	start := time.Now()

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequestWithContext(ctx, "GET", cfg.BaseURL+path, nil)
	if err != nil {
		return &runner.CheckResult{
			Name:     "GET " + path,
			Group:    "Health",
			Passed:   false,
			Duration: time.Since(start),
			Error:    fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return &runner.CheckResult{
			Name:     "GET " + path,
			Group:    "Health",
			Passed:   false,
			Duration: time.Since(start),
			Error:    fmt.Sprintf("request failed: %v", err),
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &runner.CheckResult{
			Name:       "GET " + path,
			Group:      "Health",
			Passed:     false,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("failed to read response: %v", err),
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &runner.CheckResult{
			Name:       "GET " + path,
			Group:      "Health",
			Passed:     false,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, string(body)),
		}, nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return &runner.CheckResult{
			Name:       "GET " + path,
			Group:      "Health",
			Passed:     false,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("invalid JSON response: %v", err),
		}, nil
	}

	if status, _ := response["status"].(string); status != expectedStatus {
		return &runner.CheckResult{
			Name:       "GET " + path,
			Group:      "Health",
			Passed:     false,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("expected status %q, got %q", expectedStatus, status),
		}, nil
	}

	return &runner.CheckResult{
		Name:       "GET " + path,
		Group:      "Health",
		Passed:     true,
		Duration:   time.Since(start),
		StatusCode: resp.StatusCode,
	}, nil
}
