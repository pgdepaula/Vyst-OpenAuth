package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) FetchUnprocessed(ctx context.Context, limit int) ([]event.Event, error) {
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at
		FROM outbox_events
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	// FOR UPDATE SKIP LOCKED ensures multiple workers don't process the same event

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query outbox: %w", err)
	}
	defer rows.Close()

	var events []event.Event
	for rows.Next() {
		var id uuid.UUID
		var aggType, aggID, evtType string
		var payloadBytes []byte
		var createdAt time.Time

		if err := rows.Scan(&id, &aggType, &aggID, &evtType, &payloadBytes, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		var payload interface{}
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			payload = map[string]string{"raw": string(payloadBytes)}
		}

		events = append(events, event.Event{
			ID:            id.String(),
			AggregateType: aggType,
			AggregateID:   aggID,
			Type:          event.EventType(evtType),
			Payload:       payload,
			Timestamp:     createdAt,
		})
	}
	return events, nil
}

func (r *Repository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE outbox_events
		SET processed_at = NOW()
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
