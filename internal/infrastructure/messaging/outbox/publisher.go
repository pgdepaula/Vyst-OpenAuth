// Package outbox implements the Transactional Outbox pattern for reliable event delivery.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/persistence/postgres"
)

// Publisher writes events to the outbox table within a database transaction.
// This ensures atomicity: either both business data AND event are saved, or neither.
type Publisher struct {
	pool *pgxpool.Pool
}

// NewPublisher creates a new outbox publisher.
func NewPublisher(pool *pgxpool.Pool) *Publisher {
	return &Publisher{pool: pool}
}

// Publish writes the event to the outbox_events table.
// It uses the transaction from the context if present.
func (p *Publisher) Publish(ctx context.Context, evt event.Event) error {
	query := `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
		VALUES ($1, $2, $3, $4)
	`

	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Extract aggregate info from the event
	aggregateType := "unknown"
	aggregateID := evt.ID

	// Determine aggregate type from event type
	switch evt.Type {
	case event.UserCreated, event.UserSuspended:
		aggregateType = "User"
		if payload, ok := evt.Payload.(event.UserCreatedPayload); ok {
			aggregateID = payload.UserID
		}
	case event.TenantProvisioned:
		aggregateType = "Tenant"
		if payload, ok := evt.Payload.(event.TenantProvisionedPayload); ok {
			aggregateID = payload.TenantID
		}
	}

	_, err = postgres.GetExecutor(ctx, p.pool).Exec(ctx, query, aggregateType, aggregateID, string(evt.Type), payload)
	if err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	return nil
}
