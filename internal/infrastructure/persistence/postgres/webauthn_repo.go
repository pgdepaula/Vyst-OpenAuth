package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/auth"
)

type WebAuthnRepository struct {
	pool *pgxpool.Pool
}

func NewWebAuthnRepository(pool *pgxpool.Pool) *WebAuthnRepository {
	return &WebAuthnRepository{pool: pool}
}

func (r *WebAuthnRepository) Create(ctx context.Context, cred *auth.WebAuthnCredential) error {
	query := `
		INSERT INTO webauthn_credentials (
			user_id, webauthn_id, public_key, attestation_type, transport, flags, sign_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	flagsJSON, err := json.Marshal(cred.Flags)
	if err != nil {
		return fmt.Errorf("failed to marshal flags: %w", err)
	}

	err = GetExecutor(ctx, r.pool).QueryRow(ctx, query,
		cred.UserID,
		cred.WebAuthnID,
		cred.PublicKey,
		cred.AttestationType,
		cred.Transport,
		flagsJSON,
		cred.SignCount,
	).Scan(&cred.ID, &cred.CreatedAt, &cred.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create webauthn credential: %w", err)
	}

	return nil
}

func (r *WebAuthnRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*auth.WebAuthnCredential, error) {
	query := `
		SELECT id, user_id, webauthn_id, public_key, attestation_type, transport, flags, sign_count, created_at, updated_at
		FROM webauthn_credentials
		WHERE user_id = $1
	`

	rows, err := GetExecutor(ctx, r.pool).Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query webauthn credentials: %w", err)
	}
	defer rows.Close()

	var creds []*auth.WebAuthnCredential
	for rows.Next() {
		var cred auth.WebAuthnCredential
		var flagsJSON []byte

		err := rows.Scan(
			&cred.ID,
			&cred.UserID,
			&cred.WebAuthnID,
			&cred.PublicKey,
			&cred.AttestationType,
			&cred.Transport,
			&flagsJSON,
			&cred.SignCount,
			&cred.CreatedAt,
			&cred.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webauthn credential: %w", err)
		}

		if err := json.Unmarshal(flagsJSON, &cred.Flags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal flags: %w", err)
		}

		creds = append(creds, &cred)
	}

	return creds, nil
}

func (r *WebAuthnRepository) Update(ctx context.Context, cred *auth.WebAuthnCredential) error {
	query := `
		UPDATE webauthn_credentials
		SET sign_count = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := GetExecutor(ctx, r.pool).Exec(ctx, query, cred.SignCount, cred.ID)
	if err != nil {
		return fmt.Errorf("failed to update webauthn credential: %w", err)
	}

	return nil
}
