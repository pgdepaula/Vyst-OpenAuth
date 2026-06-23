// Package postgres provides PostgreSQL implementations for domain repositories.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a requested entity doesn't exist.
var ErrNotFound = errors.New("entity not found")

// DB wraps the connection pool and provides helper methods.
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB creates a new database connection pool.
func NewDB(databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	// Ensure search_path is set to public
	config.ConnConfig.RuntimeParams["search_path"] = "public"

	config.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

// SetTenant sets the current tenant for the transaction (RLS).
// This must be called at the beginning of any transaction that accesses RLS-protected tables.
func SetTenant(ctx context.Context, tx pgx.Tx, tenantID string) error {
	// Validate UUID format to prevent SQL injection
	if _, err := uuid.Parse(tenantID); err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}

	_, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID)
	return err
}

// RunInTx executes a function within a transaction with tenant context set.
func (db *DB) RunInTx(ctx context.Context, tenantID string, fn func(pgx.Tx) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := SetTenant(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RunInTxWithoutTenant executes a function within a transaction without tenant context.
// Use this for cross-tenant operations or system-level queries.
func (db *DB) RunInTxWithoutTenant(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
