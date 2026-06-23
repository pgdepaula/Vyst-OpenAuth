package ports

import (
	"context"

	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
)

// OutboxPublisher defines the contract for publishing events to the outbox.
type OutboxPublisher interface {
	// Publish writes the event to the outbox table within a database transaction.
	Publish(ctx context.Context, evt event.Event) error
}
