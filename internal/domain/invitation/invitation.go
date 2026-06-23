package invitation

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
	StatusExpired  Status = "expired"
	StatusRevoked  Status = "revoked"
)

type Invitation struct {
	ID        string
	CompanyID string
	Email     string
	Role      company.CompanyRole
	Token     string
	Status    Status
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrNotFound = errors.New("invitation not found")
	ErrExpired  = errors.New("invitation expired")
	ErrInvalid  = errors.New("invitation invalid")
)

func NewInvitation(companyID, email string, role company.CompanyRole, expiresIn time.Duration) *Invitation {
	now := time.Now()
	return &Invitation{
		ID:        uuid.New().String(),
		CompanyID: companyID,
		Email:     email,
		Role:      role,
		Token:     uuid.New().String(), // In real app, consider a cryptographically secure random string
		Status:    StatusPending,
		ExpiresAt: now.Add(expiresIn),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (i *Invitation) Accept() error {
	if i.Status != StatusPending {
		return ErrInvalid
	}
	if time.Now().After(i.ExpiresAt) {
		i.Status = StatusExpired
		return ErrExpired
	}
	i.Status = StatusAccepted
	i.UpdatedAt = time.Now()
	return nil
}

func (i *Invitation) Revoke() error {
	if i.Status != StatusPending {
		return ErrInvalid
	}
	i.Status = StatusRevoked
	i.UpdatedAt = time.Now()
	return nil
}
