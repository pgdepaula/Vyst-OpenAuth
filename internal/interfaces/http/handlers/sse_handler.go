// Package handlers contains HTTP handlers for the Identity API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// SSEEvent represents a server-sent event.
type SSEEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Time    time.Time   `json:"time"`
}

// SSEHandler handles Server-Sent Events for real-time updates.
type SSEHandler struct {
	redisClient *redis.Client
}

// NewSSEHandler creates a new SSEHandler.
func NewSSEHandler(redisClient *redis.Client) *SSEHandler {
	return &SSEHandler{
		redisClient: redisClient,
	}
}

// StreamEvents streams events to connected clients via SSE.
// GET /api/v1/events/stream
func (h *SSEHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Flush headers
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to Redis channels
	ctx := r.Context()
	pubsub := h.redisClient.PSubscribe(ctx, "vyst:events:*")
	defer func() { _ = pubsub.Close() }()

	// Channel for Redis messages
	ch := pubsub.Channel()

	// Send initial connection event
	h.sendEvent(w, flusher, SSEEvent{
		Type:    "connected",
		Payload: map[string]string{"message": "Connected to Vyst Identity event stream"},
		Time:    time.Now(),
	})

	// Keep-alive ticker (every 30 seconds)
	keepAlive := time.NewTicker(30 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			slog.Info("SSE client disconnected")
			return

		case msg := <-ch:
			// Redis message received
			event := h.parseRedisMessage(msg)
			h.sendEvent(w, flusher, event)

		case <-keepAlive.C:
			// Send keep-alive comment
			_, _ = fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

// parseRedisMessage converts a Redis message to an SSEEvent.
func (h *SSEHandler) parseRedisMessage(msg *redis.Message) SSEEvent {
	var payload interface{}
	if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
		payload = msg.Payload
	}

	// Extract event type from channel name
	// e.g., "vyst:events:killswitch" -> "killswitch"
	eventType := msg.Channel
	if len(msg.Channel) > 13 {
		eventType = msg.Channel[13:] // Skip "vyst:events:"
	}

	return SSEEvent{
		Type:    eventType,
		Payload: payload,
		Time:    time.Now(),
	}
}

// sendEvent sends an SSE event to the client.
func (h *SSEHandler) sendEvent(w http.ResponseWriter, flusher http.Flusher, event SSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("Failed to marshal SSE event", "error", err)
		return
	}

	_, _ = fmt.Fprintf(w, "event: %s\n", event.Type)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// RiskEvent represents a risk event for the dashboard.
type RiskEvent struct {
	UserID    string    `json:"user_id"`
	Score     float64   `json:"score"`
	Reasons   []string  `json:"reasons"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"` // "blocked", "warned", "allowed"
}

// PublishRiskEvent publishes a risk event to Redis for SSE consumers.
func (h *SSEHandler) PublishRiskEvent(ctx context.Context, event RiskEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return h.redisClient.Publish(ctx, "vyst:events:risk", data).Err()
}
