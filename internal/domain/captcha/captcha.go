// Package captcha contains the CAPTCHA domain entities and interfaces.
// This is the core domain layer - no external dependencies allowed.
package captcha

import (
	"context"
	"errors"
	"time"
)

// Domain errors for CAPTCHA operations.
// These are used throughout the application to handle CAPTCHA-related failures.
var (
	// ErrCaptchaRequired indicates that CAPTCHA verification is required but was not provided.
	ErrCaptchaRequired = errors.New("captcha verification required")

	// ErrCaptchaExpired indicates that the CAPTCHA challenge has expired.
	ErrCaptchaExpired = errors.New("captcha challenge expired")

	// ErrCaptchaInvalid indicates that the CAPTCHA verification failed.
	ErrCaptchaInvalid = errors.New("captcha verification failed")

	// ErrCaptchaTokenMissing indicates that the CAPTCHA token was not provided.
	ErrCaptchaTokenMissing = errors.New("captcha token is required")
)

// CaptchaType represents the type of CAPTCHA challenge.
type CaptchaType string

const (
	// TypeInvisible is a non-interactive, risk-based CAPTCHA.
	// Used when the system determines low risk.
	TypeInvisible CaptchaType = "invisible"

	// TypeInteractive requires explicit user interaction.
	// Used for higher-risk scenarios.
	TypeInteractive CaptchaType = "interactive"

	// TypeManaged lets the provider decide the challenge type.
	// This is the default mode for Cloudflare Turnstile.
	TypeManaged CaptchaType = "managed"
)

// String returns the string representation of the CaptchaType.
func (t CaptchaType) String() string {
	return string(t)
}

// IsValid checks if the CaptchaType is a valid type.
func (t CaptchaType) IsValid() bool {
	switch t {
	case TypeInvisible, TypeInteractive, TypeManaged:
		return true
	default:
		return false
	}
}

// Challenge represents a CAPTCHA challenge issued to a client.
// This is an immutable value object after creation.
type Challenge struct {
	ID        string      // Unique challenge identifier
	Type      CaptchaType // Type of challenge
	SiteKey   string      // Public site key for frontend
	ExpiresAt time.Time   // Challenge expiration timestamp
	CreatedAt time.Time   // Challenge creation timestamp
}

// NewChallenge creates a new CAPTCHA challenge with validation.
func NewChallenge(id, siteKey string, captchaType CaptchaType, ttl time.Duration) (*Challenge, error) {
	if id == "" {
		return nil, errors.New("challenge id is required")
	}
	if siteKey == "" {
		return nil, errors.New("site key is required")
	}
	if !captchaType.IsValid() {
		captchaType = TypeManaged // Default to managed
	}

	now := time.Now()
	return &Challenge{
		ID:        id,
		Type:      captchaType,
		SiteKey:   siteKey,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}, nil
}

// IsExpired checks if the challenge has expired.
func (c *Challenge) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// Result represents the outcome of a CAPTCHA validation.
// This contains detailed information from the CAPTCHA provider.
type Result struct {
	Success     bool      // Whether validation passed
	Score       float64   // Risk score (0.0-1.0, higher = more likely human)
	Action      string    // The action that was protected
	ChallengeTS time.Time // When the challenge was solved
	Hostname    string    // Hostname where challenge was solved
	ErrorCodes  []string  // Any error codes from provider
}

// NewResult creates a new CAPTCHA result.
func NewResult(success bool, score float64) *Result {
	return &Result{
		Success:     success,
		Score:       score,
		ChallengeTS: time.Now(),
	}
}

// IsHuman returns true if the score indicates a human user.
// A score of 0.5 or higher is considered human.
func (r *Result) IsHuman() bool {
	return r.Success && r.Score >= 0.5
}

// IsBot returns true if the score indicates a bot.
func (r *Result) IsBot() bool {
	return !r.Success || r.Score < 0.5
}

// HasErrors returns true if there are error codes from the provider.
func (r *Result) HasErrors() bool {
	return len(r.ErrorCodes) > 0
}

// Validator defines the contract for CAPTCHA validation.
// Implementations live in the infrastructure layer.
type Validator interface {
	// Validate verifies a CAPTCHA response token.
	// Returns the result with score and metadata, or an error.
	Validate(ctx context.Context, token, remoteIP string) (*Result, error)

	// GetSiteKey returns the public site key for frontend integration.
	GetSiteKey() string

	// IsEnabled returns whether CAPTCHA verification is enabled.
	IsEnabled() bool
}
