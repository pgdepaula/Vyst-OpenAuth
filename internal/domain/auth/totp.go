package auth

import (
	"time"

	"github.com/google/uuid"
)

// TOTPSecret represents a TOTP secret for two-factor authentication.
type TOTPSecret struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Secret      string    `json:"secret"`
	Enabled     bool      `json:"enabled"`
	BackupCodes []string  `json:"backup_codes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TOTPRepository defines the interface for TOTP secret persistence.
type TOTPRepository interface {
	// Create stores a new TOTP secret
	Create(ctx interface{}, secret *TOTPSecret) error

	// GetByUserID retrieves the TOTP secret for a user
	GetByUserID(ctx interface{}, userID uuid.UUID) (*TOTPSecret, error)

	// Update updates an existing TOTP secret
	Update(ctx interface{}, secret *TOTPSecret) error

	// Delete removes a TOTP secret
	Delete(ctx interface{}, userID uuid.UUID) error
}
