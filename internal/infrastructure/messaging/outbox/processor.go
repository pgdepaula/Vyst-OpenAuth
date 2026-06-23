// Package outbox implements the Transactional Outbox pattern for reliable event delivery.
package outbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxEvent represents an event stored in the outbox table.
type OutboxEvent struct {
	ID            string          `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
}

// MessagePublisher defines the interface for external message brokers.
type MessagePublisher interface {
	Publish(ctx context.Context, event OutboxEvent) error
}

// Processor polls the outbox table and publishes events to external systems.
type Processor struct {
	pool      *pgxpool.Pool
	publisher MessagePublisher
	pollRate  time.Duration
	batchSize int
}

// NewProcessor creates a new outbox processor.
func NewProcessor(pool *pgxpool.Pool, publisher MessagePublisher) *Processor {
	return &Processor{
		pool:      pool,
		publisher: publisher,
		pollRate:  5 * time.Second,
		batchSize: 100,
	}
}

// NewProcessorWithConfig creates a processor with custom configuration.
func NewProcessorWithConfig(pool *pgxpool.Pool, publisher MessagePublisher, pollRate time.Duration, batchSize int) *Processor {
	return &Processor{
		pool:      pool,
		publisher: publisher,
		pollRate:  pollRate,
		batchSize: batchSize,
	}
}

// Start begins processing outbox events. Blocks until context is cancelled.
func (p *Processor) Start(ctx context.Context) {
	slog.Info("Outbox processor started")
	ticker := time.NewTicker(p.pollRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Outbox processor stopping...")
			return
		case <-ticker.C:
			if err := p.processBatch(ctx); err != nil {
				slog.Error("Error processing outbox batch", "error", err)
			}
		}
	}
}

func (p *Processor) processBatch(ctx context.Context) error {
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at
		FROM outbox_events
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := p.pool.Query(ctx, query, p.batchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&event.Payload,
			&event.CreatedAt,
		); err != nil {
			return err
		}
		events = append(events, event)
	}

	if len(events) == 0 {
		return nil
	}

	slog.Info("Processing outbox events", "count", len(events))

	for _, event := range events {
		if err := p.publisher.Publish(ctx, event); err != nil {
			slog.Error("Failed to publish event", "id", event.ID, "error", err)
			continue
		}

		// Mark as processed
		updateQuery := `UPDATE outbox_events SET processed_at = NOW() WHERE id = $1`
		if _, err := p.pool.Exec(ctx, updateQuery, event.ID); err != nil {
			slog.Error("Failed to mark event as processed", "id", event.ID, "error", err)
		} else {
			slog.Info("Processed event", "id", event.ID, "type", event.EventType)
		}
	}

	return nil
}

// LogPublisher is a simple publisher that logs events (for development/testing).
type LogPublisher struct{}

// NewLogPublisher creates a publisher that just logs events.
func NewLogPublisher() *LogPublisher {
	return &LogPublisher{}
}

// Publish logs the event details.
func (p *LogPublisher) Publish(ctx context.Context, event OutboxEvent) error {
	slog.Info("[PUBLISH] Event",
		"id", event.ID,
		"type", event.EventType,
		"aggregate", event.AggregateType+":"+event.AggregateID,
	)
	return nil
}
