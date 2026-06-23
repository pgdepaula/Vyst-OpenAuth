package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"github.com/pgdepaula/vyst-openauth/internal/domain/auth"
)

// TOTPRepository implements TOTP secret persistence using PostgreSQL.
type TOTPRepository struct {
	pool *pgxpool.Pool
}

// NewTOTPRepository creates a new TOTP repository.
func NewTOTPRepository(pool *pgxpool.Pool) *TOTPRepository {
	return &TOTPRepository{pool: pool}
}

// Create stores a new TOTP secret.
func (r *TOTPRepository) Create(ctx context.Context, secret *auth.TOTPSecret) error {
	query := `
		INSERT INTO totp_secrets (
			user_id, secret, enabled, backup_codes
		) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := GetExecutor(ctx, r.pool).QueryRow(ctx, query,
		secret.UserID,
		secret.Secret,
		secret.Enabled,
		pq.Array(secret.BackupCodes),
	).Scan(&secret.ID, &secret.CreatedAt, &secret.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create TOTP secret: %w", err)
	}

	return nil
}

// GetByUserID retrieves the TOTP secret for a user.
func (r *TOTPRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*auth.TOTPSecret, error) {
	query := `
		SELECT id, user_id, secret, enabled, backup_codes, created_at, updated_at
		FROM totp_secrets
		WHERE user_id = $1
	`

	var secret auth.TOTPSecret
	var backupCodes []string

	err := GetExecutor(ctx, r.pool).QueryRow(ctx, query, userID).Scan(
		&secret.ID,
		&secret.UserID,
		&secret.Secret,
		&secret.Enabled,
		pq.Array(&backupCodes),
		&secret.CreatedAt,
		&secret.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get TOTP secret: %w", err)
	}

	secret.BackupCodes = backupCodes
	return &secret, nil
}

// Update updates an existing TOTP secret.
func (r *TOTPRepository) Update(ctx context.Context, secret *auth.TOTPSecret) error {
	query := `
		UPDATE totp_secrets
		SET enabled = $1, backup_codes = $2, updated_at = NOW()
		WHERE user_id = $3
	`

	_, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		secret.Enabled,
		pq.Array(secret.BackupCodes),
		secret.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update TOTP secret: %w", err)
	}

	return nil
}

// Delete removes a TOTP secret.
func (r *TOTPRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM totp_secrets WHERE user_id = $1`

	_, err := GetExecutor(ctx, r.pool).Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete TOTP secret: %w", err)
	}

	return nil
}
