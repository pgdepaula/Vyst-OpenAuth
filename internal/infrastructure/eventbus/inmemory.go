package eventbus

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"sync"

	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
)

// InMemoryBus is a simple in-memory event bus.
type InMemoryBus struct {
	mu       sync.RWMutex
	handlers map[event.EventType]map[string]event.Handler
	logger   *log.Logger
}

// NewInMemoryBus creates a new in-memory event bus.
func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{
		handlers: make(map[event.EventType]map[string]event.Handler),
		logger:   log.Default(),
	}
}

// Publish publishes an event to all subscribers.
func (b *InMemoryBus) Publish(ctx context.Context, evt event.Event) error {
	// Log it
	data, _ := json.Marshal(evt)
	b.logger.Printf("[EVENT BUS] Publishing %s: %s", evt.Type, string(data))

	b.mu.RLock()
	handlersMap, ok := b.handlers[evt.Type]
	// Create a snapshot of handlers to avoid holding lock during execution
	var handlers []event.Handler
	if ok {
		for _, h := range handlersMap {
			handlers = append(handlers, h)
		}
	}
	b.mu.RUnlock()

	for _, h := range handlers {
		// Run handlers asynchronously to avoid blocking publisher
		go func(handler event.Handler) {
			if err := handler(ctx, evt); err != nil {
				b.logger.Printf("Error handling event %s: %v", evt.Type, err)
			}
		}(h)
	}
	return nil
}

// Subscribe subscribes a handler to an event type.
// Returns a function to unsubscribe.
func (b *InMemoryBus) Subscribe(eventType event.EventType, handler event.Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.handlers[eventType] == nil {
		b.handlers[eventType] = make(map[string]event.Handler)
	}

	id := uuid.New().String()
	b.handlers[eventType][id] = handler

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if m, ok := b.handlers[eventType]; ok {
			delete(m, id)
			if len(m) == 0 {
				delete(b.handlers, eventType)
			}
		}
	}
}
