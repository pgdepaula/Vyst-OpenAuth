// Package user contains the User domain entity and repository interface.
// This is the core domain layer - no external dependencies allowed.
package user

import (
	"context"
	"errors"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/domain/document"
)

var ErrNotFound = errors.New("user not found")

// User represents the core user entity in the system.
// It is a pure domain object with no infrastructure concerns.
type User struct {
	ID                         string       // Unique identifier
	Email                      string       // User email address
	PasswordHash               string       // Bcrypt hash of password
	TenantID                   string       // ID of the tenant this user belongs to
	IdentityType               string       // Type of identity: "individual" or "company"
	CPF                        document.CPF // Validated CPF Value Object. Optional for companies, required for individuals.
	ActiveCompanyID            string       // Current active company context (empty for individual login)
	AvatarURL                  string       // URL to user avatar
	CreatedAt                  time.Time    // Timestamp of creation
	UpdatedAt                  time.Time    // Timestamp of last update
	LastLoginAt                time.Time    // Timestamp of last successful login
	LoginAttempts              int          // Count of failed login attempts
	LockedUntil                time.Time    // Timestamp until which the account is locked
	ResetToken                 string       // Token for password reset
	ResetTokenExpiresAt        time.Time    // Expiration time for reset token
	Status                     string       // User status: "pending", "active", "suspended"
	VerificationToken          string       // Token for email verification
	VerificationTokenExpiresAt time.Time    // Expiration for verification token
	MFAEnabled                 bool         // Whether MFA is enabled
	MFASecret                  string       // Secret for TOTP MFA
}

// User status constants
const (
	StatusPending   = "pending"
	StatusActive    = "active"
	StatusSuspended = "suspended"
)

// Identity type constants
const (
	IdentityTypeIndividual = "individual"
	IdentityTypeCompany    = "company"
)

// NewUser creates a new user with validation.
// By default, users are created as individual (pessoa física) identity type.
func NewUser(id, email, passwordHash, tenantID string) (*User, error) {
	if id == "" {
		return nil, errors.New("user id is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}
	if passwordHash == "" {
		return nil, errors.New("password hash is required")
	}
	if tenantID == "" {
		return nil, errors.New("tenant id is required")
	}

	now := time.Now()
	return &User{
		ID:                         id,
		Email:                      email,
		PasswordHash:               passwordHash,
		TenantID:                   tenantID,
		IdentityType:               IdentityTypeIndividual, // Default to individual
		ActiveCompanyID:            "",                     // No active company by default
		CreatedAt:                  now,
		UpdatedAt:                  now,
		ResetToken:                 "",
		ResetTokenExpiresAt:        time.Time{},
		Status:                     StatusPending,
		VerificationToken:          "", // Will be set by service
		VerificationTokenExpiresAt: time.Time{},
	}, nil
}

// Repository defines the contract for user persistence operations.
// Implementations live in the infrastructure layer.
type Repository interface {
	// Create persists a new user to the storage.
	Create(ctx context.Context, user *User) error

	// GetByEmail retrieves a user by their email address.
	// Returns ErrNotFound if user doesn't exist.
	GetByEmail(ctx context.Context, email string) (*User, error)

	// GetByID retrieves a user by their unique identifier.
	// Returns ErrNotFound if user doesn't exist.
	GetByID(ctx context.Context, id string) (*User, error)

	// Update modifies an existing user's data.
	Update(ctx context.Context, user *User) error

	// GetByResetToken retrieves a user by their reset token.
	GetByResetToken(ctx context.Context, token string) (*User, error)

	// GetByVerificationToken retrieves a user by their verification token.
	GetByVerificationToken(ctx context.Context, token string) (*User, error)
}
