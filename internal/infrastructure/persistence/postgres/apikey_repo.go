package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/apikey"
)

type PostgresAPIKeyRepository struct {
	db *pgxpool.Pool
}

func NewPostgresAPIKeyRepository(db *pgxpool.Pool) *PostgresAPIKeyRepository {
	return &PostgresAPIKeyRepository{db: db}
}

func (r *PostgresAPIKeyRepository) Create(ctx context.Context, key *apikey.APIKey) error {
	query := `
		INSERT INTO api_keys (id, tenant_id, user_id, name, key_prefix, key_hash, scopes, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query,
		key.ID,
		key.TenantID,
		key.UserID,
		key.Name,
		key.KeyPrefix,
		key.KeyHash,
		key.Scopes,
		key.ExpiresAt,
		key.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create api key: %w", err)
	}
	return nil
}

func (r *PostgresAPIKeyRepository) GetByPrefix(ctx context.Context, prefix string) (*apikey.APIKey, error) {
	query := `
		SELECT id, tenant_id, user_id, name, key_prefix, key_hash, scopes, expires_at, last_used_at, created_at
		FROM api_keys
		WHERE key_prefix = $1
	`
	row := r.db.QueryRow(ctx, query, prefix)
	return scanAPIKey(row)
}

func (r *PostgresAPIKeyRepository) ListByTenant(ctx context.Context, tenantID string) ([]*apikey.APIKey, error) {
	query := `
		SELECT id, tenant_id, user_id, name, key_prefix, key_hash, scopes, expires_at, last_used_at, created_at
		FROM api_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}
	defer rows.Close()

	var keys []*apikey.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *PostgresAPIKeyRepository) Revoke(ctx context.Context, id string) error {
	query := `DELETE FROM api_keys WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}
	return nil
}

func (r *PostgresAPIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	query := `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}
	return nil
}

func scanAPIKey(row pgx.Row) (*apikey.APIKey, error) {
	var k apikey.APIKey
	err := row.Scan(
		&k.ID,
		&k.TenantID,
		&k.UserID,
		&k.Name,
		&k.KeyPrefix,
		&k.KeyHash,
		&k.Scopes,
		&k.ExpiresAt,
		&k.LastUsedAt,
		&k.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan api key: %w", err)
	}
	return &k, nil
}
