package ports

import (
	"context"

	"github.com/pgdepaula/vyst-openauth/internal/domain/captcha"
)

// CaptchaService defines the contract for CAPTCHA operations.
// This is an output port - implementations live in infrastructure.
type CaptchaService interface {
	// ValidateToken verifies a CAPTCHA response token.
	// Returns nil if valid, or a domain error if invalid.
	// When CAPTCHA is disabled, returns nil without validation.
	ValidateToken(ctx context.Context, token, remoteIP string) error

	// ValidateTokenWithResult verifies and returns detailed result.
	// This provides full validation metadata including score.
	ValidateTokenWithResult(ctx context.Context, token, remoteIP string) (*captcha.Result, error)

	// GetConfig returns the CAPTCHA configuration for frontend.
	// This includes the public site key needed for widget rendering.
	GetConfig() CaptchaConfig

	// IsEnabled returns whether CAPTCHA verification is enabled.
	// When false, ValidateToken always returns nil (success).
	IsEnabled() bool
}

// CaptchaConfig contains public configuration for frontend.
// This is safe to expose to clients via API.
type CaptchaConfig struct {
	// SiteKey is the public site key for the CAPTCHA widget.
	SiteKey string `json:"site_key"`

	// Enabled indicates whether CAPTCHA verification is active.
	Enabled bool `json:"enabled"`
}
