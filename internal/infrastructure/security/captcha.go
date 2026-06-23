// Package security contains security-related infrastructure services.
package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/captcha"
)

// Ensure TurnstileService implements ports.CaptchaService at compile time.
var _ ports.CaptchaService = (*TurnstileService)(nil)

const (
	// turnstileVerifyURL is the Cloudflare Turnstile verification endpoint.
	turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

	// defaultTimeout is the default HTTP client timeout for Turnstile API calls.
	defaultTimeout = 10 * time.Second

	// defaultScore is the score assigned when Turnstile doesn't return a score.
	// Turnstile managed mode doesn't always provide a score, so we assume success = 1.0.
	defaultScore = 1.0
)

// TurnstileService implements CAPTCHA verification using Cloudflare Turnstile.
// It implements the ports.CaptchaService interface.
type TurnstileService struct {
	secretKey string
	siteKey   string
	enabled   bool
	client    *http.Client
	logger    ports.Logger
}

// NewTurnstileService creates a new Turnstile CAPTCHA service.
// If siteKey or secretKey are empty, the service will be disabled.
func NewTurnstileService(siteKey, secretKey string, logger ports.Logger) *TurnstileService {
	enabled := secretKey != "" && siteKey != ""

	if logger != nil && enabled {
		logger.Info("Turnstile CAPTCHA service initialized", "enabled", enabled)
	}

	return &TurnstileService{
		secretKey: secretKey,
		siteKey:   siteKey,
		enabled:   enabled,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		logger: logger,
	}
}

// ValidateToken verifies a CAPTCHA response token.
// Returns nil if valid, or a domain error if invalid.
func (s *TurnstileService) ValidateToken(ctx context.Context, token, remoteIP string) error {
	_, err := s.ValidateTokenWithResult(ctx, token, remoteIP)
	return err
}

// ValidateTokenWithResult verifies and returns detailed result.
// This provides full validation metadata including score.
func (s *TurnstileService) ValidateTokenWithResult(ctx context.Context, token, remoteIP string) (*captcha.Result, error) {
	// If CAPTCHA is disabled, return success immediately
	if !s.enabled {
		s.logDebug(ctx, "CAPTCHA disabled, skipping validation")
		return &captcha.Result{
			Success:     true,
			Score:       defaultScore,
			ChallengeTS: time.Now(),
		}, nil
	}

	// Validate token is present
	if token == "" {
		s.logWarn(ctx, "CAPTCHA token missing", "remote_ip", remoteIP)
		return nil, captcha.ErrCaptchaTokenMissing
	}

	s.logDebug(ctx, "Validating CAPTCHA token", "remote_ip", remoteIP)

	// Make request to Turnstile API
	result, err := s.verifyWithTurnstile(ctx, token, remoteIP)
	if err != nil {
		s.logError(ctx, "CAPTCHA verification request failed", "error", err, "remote_ip", remoteIP)
		return nil, fmt.Errorf("failed to verify captcha: %w", err)
	}

	// Log result
	if result.Success {
		s.logInfo(ctx, "CAPTCHA verification successful",
			"remote_ip", remoteIP,
			"hostname", result.Hostname,
			"action", result.Action,
		)
	} else {
		s.logWarn(ctx, "CAPTCHA verification failed",
			"remote_ip", remoteIP,
			"error_codes", result.ErrorCodes,
		)
		return result, captcha.ErrCaptchaInvalid
	}

	return result, nil
}

// GetConfig returns the CAPTCHA configuration for frontend.
func (s *TurnstileService) GetConfig() ports.CaptchaConfig {
	return ports.CaptchaConfig{
		SiteKey: s.siteKey,
		Enabled: s.enabled,
	}
}

// IsEnabled returns whether CAPTCHA verification is enabled.
func (s *TurnstileService) IsEnabled() bool {
	return s.enabled
}

// verifyWithTurnstile makes the actual API call to Cloudflare Turnstile.
func (s *TurnstileService) verifyWithTurnstile(ctx context.Context, token, remoteIP string) (*captcha.Result, error) {
	reqBody := turnstileRequest{
		Secret:   s.secretKey,
		Response: token,
		RemoteIP: remoteIP,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call turnstile API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var apiResp turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to domain result
	result := &captcha.Result{
		Success:     apiResp.Success,
		Score:       defaultScore, // Turnstile managed doesn't always return score
		Action:      apiResp.Action,
		Hostname:    apiResp.Hostname,
		ErrorCodes:  apiResp.ErrorCodes,
		ChallengeTS: time.Now(),
	}

	// Try to parse challenge timestamp if provided
	if apiResp.ChallengeTS != "" {
		if ts, err := time.Parse(time.RFC3339, apiResp.ChallengeTS); err == nil {
			result.ChallengeTS = ts
		}
	}

	return result, nil
}

// Logging helpers that handle nil logger gracefully

func (s *TurnstileService) logDebug(ctx context.Context, msg string, args ...any) {
	if s.logger != nil {
		s.logger.WithContext(ctx).Debug(msg, args...)
	}
}

func (s *TurnstileService) logInfo(ctx context.Context, msg string, args ...any) {
	if s.logger != nil {
		s.logger.WithContext(ctx).Info(msg, args...)
	}
}

func (s *TurnstileService) logWarn(ctx context.Context, msg string, args ...any) {
	if s.logger != nil {
		s.logger.WithContext(ctx).Warn(msg, args...)
	}
}

func (s *TurnstileService) logError(ctx context.Context, msg string, args ...any) {
	if s.logger != nil {
		s.logger.WithContext(ctx).Error(msg, args...)
	}
}

// Internal request/response types for Turnstile API

type turnstileRequest struct {
	Secret   string `json:"secret"`
	Response string `json:"response"`
	RemoteIP string `json:"remoteip,omitempty"`
}

type turnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
	Action      string   `json:"action"`
	CData       string   `json:"cdata"`
}

// ============================================================================
// DEPRECATED: Legacy API for backward compatibility
// These functions maintain the old interface while transitioning to the new one.
// TODO: Remove after all callers are updated to use the new interface.
// ============================================================================

// CaptchaService is an alias for TurnstileService for backward compatibility.
// Deprecated: Use TurnstileService instead.
type CaptchaService = TurnstileService

// NewCaptchaService creates a new CAPTCHA service (legacy constructor).
// Deprecated: Use NewTurnstileService instead.
func NewCaptchaService(siteKey, secretKey string) *CaptchaService {
	return NewTurnstileService(siteKey, secretKey, nil)
}

// GetSiteKey returns the public site key for the frontend.
// Deprecated: Use GetConfig().SiteKey instead.
func (s *TurnstileService) GetSiteKey() string {
	return s.siteKey
}

// Verify validates a Turnstile token (legacy method).
// Deprecated: Use ValidateToken instead.
func (s *TurnstileService) Verify(token string, remoteIP string) error {
	return s.ValidateToken(context.Background(), token, remoteIP)
}
