package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/risk"
)

type PostgresLoginHistoryRepository struct {
	pool *pgxpool.Pool
}

func NewLoginHistoryRepository(pool *pgxpool.Pool) *PostgresLoginHistoryRepository {
	return &PostgresLoginHistoryRepository{pool: pool}
}

func (r *PostgresLoginHistoryRepository) Save(ctx context.Context, history *risk.LoginHistory) error {
	query := `
		INSERT INTO login_history (user_id, ip_address, user_agent, login_at, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		history.UserID,
		history.IPAddress,
		history.UserAgent,
		history.LoginAt,
		history.Latitude,
		history.Longitude,
	).Scan(&history.ID)

	if err != nil {
		return fmt.Errorf("failed to save login history: %w", err)
	}
	return nil
}

func (r *PostgresLoginHistoryRepository) GetLastLogin(ctx context.Context, userID uuid.UUID) (*risk.LoginHistory, error) {
	query := `
		SELECT id, user_id, ip_address, user_agent, login_at, latitude, longitude
		FROM login_history
		WHERE user_id = $1
		ORDER BY login_at DESC
		LIMIT 1
	`
	var h risk.LoginHistory
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&h.ID,
		&h.UserID,
		&h.IPAddress,
		&h.UserAgent,
		&h.LoginAt,
		&h.Latitude,
		&h.Longitude,
	)
	if err != nil {
		// Handle no rows
		return nil, nil // Or specific error if preferred
	}
	return &h, nil
}
