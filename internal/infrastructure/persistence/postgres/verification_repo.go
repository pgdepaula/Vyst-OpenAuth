package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/document"
)

type VerificationRepository struct {
	pool *pgxpool.Pool
}

func NewVerificationRepository(pool *pgxpool.Pool) *VerificationRepository {
	return &VerificationRepository{pool: pool}
}

func (r *VerificationRepository) Save(ctx context.Context, record *document.VerificationRecord) error {
	query := `
		INSERT INTO document_verifications (id, document, type, source, valid, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	metaJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx, query,
		record.ID,
		record.Document,
		record.Type,
		record.Source,
		record.Valid,
		metaJSON,
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save verification record: %w", err)
	}

	return nil
}

func (r *VerificationRepository) GetByDocument(ctx context.Context, docType, docValue string) ([]*document.VerificationRecord, error) {
	query := `
		SELECT id, document, type, source, valid, metadata, created_at
		FROM document_verifications
		WHERE type = $1 AND document = $2
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, docType, docValue)
	if err != nil {
		return nil, fmt.Errorf("failed to query verifications: %w", err)
	}
	defer rows.Close()

	var records []*document.VerificationRecord
	for rows.Next() {
		var r document.VerificationRecord
		var metaJSON []byte
		if err := rows.Scan(
			&r.ID,
			&r.Document,
			&r.Type,
			&r.Source,
			&r.Valid,
			&metaJSON,
			&r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan verification record: %w", err)
		}
		if len(metaJSON) > 0 {
			r.Metadata = string(metaJSON)
		}
		records = append(records, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return records, nil
}
