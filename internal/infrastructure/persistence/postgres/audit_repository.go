package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/audit"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Create(ctx context.Context, entry *audit.LogEntry) error {
	query := `
		INSERT INTO audit_logs (id, tenant_id, actor_id, action, entity, entity_id, metadata, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		entry.ID, entry.TenantID, entry.ActorID, entry.Action,
		entry.Entity, entry.EntityID, entry.Metadata, entry.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}
	return nil
}

func (r *AuditRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*audit.LogEntry, error) {
	query := `
		SELECT id, tenant_id, actor_id, action, entity, entity_id, metadata, timestamp
		FROM audit_logs
		WHERE tenant_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*audit.LogEntry
	for rows.Next() {
		var l audit.LogEntry
		// pgx automatically handles JSONB to map[string]interface{}
		err := rows.Scan(
			&l.ID, &l.TenantID, &l.ActorID, &l.Action,
			&l.Entity, &l.EntityID, &l.Metadata, &l.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		// In some cases timestamp might be scanned as UTC but time.Now is local. Ensure consistency.
		l.Timestamp = l.Timestamp.UTC()
		logs = append(logs, &l)
	}

	return logs, nil
}
