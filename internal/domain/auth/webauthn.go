package auth

import (
	"context"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// WebAuthnCredential represents a stored passkey.
type WebAuthnCredential struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	WebAuthnID      []byte
	PublicKey       []byte
	AttestationType string
	Transport       []string
	Flags           map[string]interface{}
	SignCount       uint32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// WebAuthnRepository defines storage operations for credentials.
type WebAuthnRepository interface {
	Create(ctx context.Context, cred *WebAuthnCredential) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*WebAuthnCredential, error)
	Update(ctx context.Context, cred *WebAuthnCredential) error
}

// WebAuthnService defines the business logic for passkey operations.
type WebAuthnService interface {
	BeginRegistration(ctx context.Context, userID uuid.UUID) (*protocol.CredentialCreation, *webauthn.SessionData, error)
	FinishRegistration(ctx context.Context, userID uuid.UUID, sessionData webauthn.SessionData, response *protocol.ParsedCredentialCreationData) error
	BeginLogin(ctx context.Context, userID uuid.UUID) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	FinishLogin(ctx context.Context, userID uuid.UUID, sessionData webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error)
}
