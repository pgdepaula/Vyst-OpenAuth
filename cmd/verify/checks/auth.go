package checks

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/config"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/runner"
)

// AuthContext holds authentication state for dependent checks.
type AuthContext struct {
	Email         string
	Password      string
	TenantName    string
	Token         string
	RefreshToken  string
	UserID        string
	TenantID      string
	CompanyID     string // ID of a test company for company checks
	EmailVerified bool   // Flag to avoid multiple verification attempts
}

// NewAuthContext creates a new authentication context with random credentials.
func NewAuthContext() *AuthContext {
	suffix := uuid.NewString()
	return &AuthContext{
		Email:         fmt.Sprintf("verify_%s@example.com", suffix),
		Password:      "VerifyPassword123!",
		TenantName:    fmt.Sprintf("verify_tenant_%s", suffix),
		EmailVerified: false,
	}
}

// AuthChecks returns all authentication-related verification checks.
// These checks depend on each other and must run in sequence.
func AuthChecks(authCtx *AuthContext) []runner.Check {
	return []runner.Check{
		{
			Name:  "POST /auth/register",
			Group: "Auth",
			Fn:    makeRegisterCheck(authCtx),
		},
		{
			Name:  "POST /auth/login",
			Group: "Auth",
			Fn:    makeLoginCheck(authCtx),
		},
		{
			Name:  "GET /auth/me",
			Group: "Auth",
			Fn:    makeMeCheck(authCtx),
		},
		{
			Name:  "GET /auth/2fa/status",
			Group: "Auth",
			Fn:    make2FAStatusCheck(authCtx),
		},
		{
			Name:  "GET /auth/captcha-config",
			Group: "Auth",
			Fn:    makeCaptchaConfigCheck(authCtx),
		},
	}
}

func makeRegisterCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		payload := map[string]string{
			"email":         authCtx.Email,
			"password":      authCtx.Password,
			"tenant_name":   authCtx.TenantName,
			"captcha_token": "dummy-token",
		}

		resp, body, err := doPost(ctx, cfg, "/auth/register", "", payload)
		if err != nil {
			return &runner.CheckResult{
				Name:     "POST /auth/register",
				Group:    "Auth",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "POST /auth/register",
				Group:      "Auth",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 201, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "POST /auth/register",
			Group:      "Auth",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeLoginCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		// Verify email only once (on first attempt)
		if !authCtx.EmailVerified {
			// Small delay to ensure register has completed
			time.Sleep(3 * time.Second)

			if err := verifyEmailDirect(ctx, cfg, authCtx.Email); err != nil {
				if cfg.Verbose {
					fmt.Printf("    ⚠️  Email verification skipped: %v\n", err)
				}
			} else {
				authCtx.EmailVerified = true
			}
		}

		payload := map[string]string{
			"email":         authCtx.Email,
			"password":      authCtx.Password,
			"captcha_token": "dummy-token",
		}

		resp, body, err := doPost(ctx, cfg, "/auth/login", "", payload)
		if err != nil {
			return &runner.CheckResult{
				Name:     "POST /auth/login",
				Group:    "Auth",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "POST /auth/login",
				Group:      "Auth",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		// Extract token from response
		var loginResp struct {
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
			User         struct {
				ID       string `json:"id"`
				TenantID string `json:"tenant_id"`
			} `json:"user"`
		}
		if err := json.Unmarshal(body, &loginResp); err != nil {
			return &runner.CheckResult{
				Name:       "POST /auth/login",
				Group:      "Auth",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("failed to parse login response: %v", err),
			}, nil
		}

		authCtx.Token = loginResp.Token
		authCtx.RefreshToken = loginResp.RefreshToken
		authCtx.UserID = loginResp.User.ID
		authCtx.TenantID = loginResp.User.TenantID

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:       "POST /auth/login",
				Group:      "Auth",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      "login succeeded but no token returned",
			}, nil
		}

		return &runner.CheckResult{
			Name:       "POST /auth/login",
			Group:      "Auth",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeMeCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "GET /auth/me",
				Group:    "Auth",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available (login must succeed first)",
			}, nil
		}

		resp, body, err := doGet(ctx, cfg, "/auth/me", authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     "GET /auth/me",
				Group:    "Auth",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "GET /auth/me",
				Group:      "Auth",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		// Extract user_id and tenant_id from /auth/me response
		var meResp struct {
			ID       string `json:"id"`
			TenantID string `json:"tenant_id"`
		}
		if err := json.Unmarshal(body, &meResp); err == nil {
			if meResp.ID != "" {
				authCtx.UserID = meResp.ID
			}
			if meResp.TenantID != "" {
				authCtx.TenantID = meResp.TenantID
			}
		}

		return &runner.CheckResult{
			Name:       "GET /auth/me",
			Group:      "Auth",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func make2FAStatusCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "GET /auth/2fa/status",
				Group:    "Auth",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		resp, body, err := doGet(ctx, cfg, "/auth/2fa/status", authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     "GET /auth/2fa/status",
				Group:    "Auth",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "GET /auth/2fa/status",
				Group:      "Auth",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "GET /auth/2fa/status",
			Group:      "Auth",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeCaptchaConfigCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		resp, body, err := doGet(ctx, cfg, "/auth/captcha-config", "")
		if err != nil {
			return &runner.CheckResult{
				Name:     "GET /auth/captcha-config",
				Group:    "Auth",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "GET /auth/captcha-config",
				Group:      "Auth",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "GET /auth/captcha-config",
			Group:      "Auth",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

// verifyEmailDirect attempts to verify email by directly updating the database.
// This is a testing helper - in production, users would click a link.
func verifyEmailDirect(ctx context.Context, cfg *config.Config, email string) error {
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("no database URL configured")
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Start a transaction to bypass RLS
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) && cfg.Verbose {
			fmt.Printf("    Failed to rollback email verification transaction: %v\n", err)
		}
	}()

	// Bypass RLS - as superuser (postgres), we can use SET LOCAL
	if _, err := tx.ExecContext(ctx, "SET LOCAL app.bypass_rls = 'on'"); err != nil {
		// Try alternative: just proceed as superuser bypasses RLS by default
		if cfg.Verbose {
			fmt.Printf("    RLS bypass setting skipped: %v\n", err)
		}
	}

	// Set status to 'active' and clear verification token
	// This mimics what VerifyEmail service does
	result, err := tx.ExecContext(ctx,
		"UPDATE users SET status = 'active', verification_token = '', verification_token_expires_at = NULL WHERE email = $1",
		email,
	)
	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found: %s", email)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// HTTP helpers

func doPost(ctx context.Context, cfg *config.Config, path, token string, payload interface{}) (*http.Response, []byte, error) {
	client := &http.Client{Timeout: cfg.Timeout}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.BaseURL+path, bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp, respBody, nil
}

func doGet(ctx context.Context, cfg *config.Config, path, token string) (*http.Response, []byte, error) {
	client := &http.Client{Timeout: cfg.Timeout}

	req, err := http.NewRequestWithContext(ctx, "GET", cfg.BaseURL+path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp, respBody, nil
}

func doDelete(ctx context.Context, cfg *config.Config, path, token string) (*http.Response, []byte, error) {
	client := &http.Client{Timeout: cfg.Timeout}

	req, err := http.NewRequestWithContext(ctx, "DELETE", cfg.BaseURL+path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp, respBody, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
