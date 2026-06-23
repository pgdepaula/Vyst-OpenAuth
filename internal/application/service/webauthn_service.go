package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/domain/auth"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

type WebAuthnService struct {
	webAuthn *webauthn.WebAuthn
	userRepo user.Repository
	authRepo auth.WebAuthnRepository
}

func NewWebAuthnService(
	userRepo user.Repository,
	authRepo auth.WebAuthnRepository,
	rpID string,
	rpOrigin string,
	rpDisplayName string,
) (*WebAuthnService, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: rpDisplayName,
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize webauthn: %w", err)
	}

	return &WebAuthnService{
		webAuthn: w,
		userRepo: userRepo,
		authRepo: authRepo,
	}, nil
}

// User adapter for webauthn library
type webAuthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webAuthnUser) WebAuthnName() string                       { return u.name }
func (u *webAuthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *webAuthnUser) WebAuthnIcon() string                       { return "" }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

func (s *WebAuthnService) getUserAdapter(ctx context.Context, userID uuid.UUID) (*webAuthnUser, error) {
	u, err := s.userRepo.GetByID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil {
		return nil, fmt.Errorf("user not found")
	}

	creds, err := s.authRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	wCreds := make([]webauthn.Credential, 0, len(creds))
	for _, c := range creds {
		transports := make([]protocol.AuthenticatorTransport, 0, len(c.Transport))
		for _, t := range c.Transport {
			transports = append(transports, protocol.AuthenticatorTransport(t))
		}

		wCreds = append(wCreds, webauthn.Credential{
			ID:              c.WebAuthnID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Transport:       transports,
			Flags: webauthn.CredentialFlags{
				UserPresent:    c.Flags["userPresent"].(bool),
				UserVerified:   c.Flags["userVerified"].(bool),
				BackupEligible: c.Flags["backupEligible"].(bool),
				BackupState:    c.Flags["backupState"].(bool),
			},
			Authenticator: webauthn.Authenticator{
				SignCount: c.SignCount,
			},
		})
	}

	return &webAuthnUser{
		id:          []byte(u.ID),
		name:        u.Email,
		displayName: u.Email, // Could be Name if we had it
		credentials: wCreds,
	}, nil
}

func (s *WebAuthnService) BeginRegistration(ctx context.Context, userID uuid.UUID) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	user, err := s.getUserAdapter(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	options, session, err := s.webAuthn.BeginRegistration(user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin registration: %w", err)
	}

	return options, session, nil
}

func (s *WebAuthnService) FinishRegistration(ctx context.Context, userID uuid.UUID, sessionData webauthn.SessionData, r *http.Request) error {
	user, err := s.getUserAdapter(ctx, userID)
	if err != nil {
		return err
	}

	credential, err := s.webAuthn.FinishRegistration(user, sessionData, r)
	if err != nil {
		return fmt.Errorf("failed to finish registration: %w", err)
	}

	transports := make([]string, 0, len(credential.Transport))
	for _, t := range credential.Transport {
		transports = append(transports, string(t))
	}

	// Save credential
	newCred := &auth.WebAuthnCredential{
		UserID:          userID,
		WebAuthnID:      credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		Transport:       transports,
		Flags: map[string]interface{}{
			"userPresent":    credential.Flags.UserPresent,
			"userVerified":   credential.Flags.UserVerified,
			"backupEligible": credential.Flags.BackupEligible,
			"backupState":    credential.Flags.BackupState,
		},
		SignCount: credential.Authenticator.SignCount,
	}

	if err := s.authRepo.Create(ctx, newCred); err != nil {
		return fmt.Errorf("failed to save credential: %w", err)
	}

	return nil
}

func (s *WebAuthnService) BeginLogin(ctx context.Context, userID uuid.UUID) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	user, err := s.getUserAdapter(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	options, session, err := s.webAuthn.BeginLogin(user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin login: %w", err)
	}

	return options, session, nil
}

func (s *WebAuthnService) BeginLoginByEmail(ctx context.Context, email string) (*protocol.CredentialAssertion, *webauthn.SessionData, uuid.UUID, error) {
	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, uuid.Nil, fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil {
		return nil, nil, uuid.Nil, fmt.Errorf("user not found")
	}

	// We need to parse the ID string back to UUID to use our helper
	// (or just use the helper if we refactor, but let's just parse)
	// Actually, getUserAdapter takes UUID.
	// User.ID is string.
	userID, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, nil, uuid.Nil, fmt.Errorf("invalid user id in db: %w", err)
	}

	options, session, err := s.BeginLogin(ctx, userID)
	if err != nil {
		return nil, nil, uuid.Nil, err
	}

	return options, session, userID, nil
}

func (s *WebAuthnService) FinishLogin(ctx context.Context, userID uuid.UUID, sessionData webauthn.SessionData, r *http.Request) (*webauthn.Credential, error) {
	user, err := s.getUserAdapter(ctx, userID)
	if err != nil {
		return nil, err
	}

	credential, err := s.webAuthn.FinishLogin(user, sessionData, r)
	if err != nil {
		return nil, fmt.Errorf("failed to finish login: %w", err)
	}

	// Update sign count
	credToUpdate := &auth.WebAuthnCredential{
		SignCount: credential.Authenticator.SignCount,
	}

	// Find the credential in the user's list to get the DB ID
	for _, c := range user.credentials {
		if string(c.ID) == string(credential.ID) {
			storedCreds, _ := s.authRepo.GetByUserID(ctx, userID)
			for _, sc := range storedCreds {
				if string(sc.WebAuthnID) == string(credential.ID) {
					credToUpdate.ID = sc.ID
					break
				}
			}
		}
	}

	if credToUpdate.ID != uuid.Nil {
		if err := s.authRepo.Update(ctx, credToUpdate); err != nil {
			// Log error but don't fail login? Or fail?
			// FIDO2 spec says we should update counter.
			return nil, fmt.Errorf("failed to update credential counter: %w", err)
		}
	}

	return credential, nil
}
