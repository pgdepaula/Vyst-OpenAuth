package event

import (
	"context"
)

// Handler is a function that handles an event.
type Handler func(ctx context.Context, event Event) error

// Bus defines the interface for an event bus (Pub/Sub).
type Bus interface {
	Publisher
	// Subscribe registers a handler for an event type.
	// Returns a function to unsubscribe/remove the handler.
	Subscribe(eventType EventType, handler Handler) func()
}
