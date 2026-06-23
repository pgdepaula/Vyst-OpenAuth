package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
)

type txKey struct{}

// PostgresTransactionManager implements ports.TransactionManager for PostgreSQL.
type PostgresTransactionManager struct {
	pool *pgxpool.Pool
}

// NewTransactionManager creates a new PostgresTransactionManager.
func NewTransactionManager(pool *pgxpool.Pool) ports.TransactionManager {
	return &PostgresTransactionManager{pool: pool}
}

// RunInTransaction executes the function within a transaction.
func (tm *PostgresTransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer rollback in case of panic or error (if commit is not called)
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Inject transaction into context
	ctxWithTx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(ctxWithTx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DBExecutor is an interface that matches both pgxpool.Pool and pgx.Tx
type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// GetExecutor returns the transaction from context if present, otherwise the pool.
func GetExecutor(ctx context.Context, pool *pgxpool.Pool) DBExecutor {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}
