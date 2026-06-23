package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/domain/webhook"
)

// WebhookService handles webhook subscriptions and delivery.
type WebhookService struct {
	webhookRepo webhook.Repository
	eventBus    event.Bus
	logger      ports.Logger
	httpClient  *http.Client
}

// NewWebhookService creates a new WebhookService and subscribes to relevant events.
func NewWebhookService(
	webhookRepo webhook.Repository,
	eventBus event.Bus,
	logger ports.Logger,
) *WebhookService {
	s := &WebhookService{
		webhookRepo: webhookRepo,
		eventBus:    eventBus,
		logger:      logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Subscribe to all relevant events
	// In a real system, we might subscribe to *everything* or a specific list.
	// For now, let's subscribe to major company events.
	s.subscribeToEvents()

	return s
}

func (s *WebhookService) subscribeToEvents() {
	eventTypes := []event.EventType{
		event.CompanyCreated,
		event.CompanyUserAdded,
		event.CompanyUserRemoved,
		event.UserCreated,
	}

	for _, et := range eventTypes {
		s.eventBus.Subscribe(et, s.handleEvent)
	}
}

// handleEvent is the event handler that dispatches webhooks.
func (s *WebhookService) handleEvent(ctx context.Context, e event.Event) error {
	// Find webhooks subscribed to this event in this tenant
	webhooks, err := s.webhookRepo.ListByEvent(ctx, e.TenantID, string(e.Type))
	if err != nil {
		s.logger.Error("Failed to list webhooks for event", "event_id", e.ID, "error", err)
		return err
	}

	if len(webhooks) == 0 {
		return nil
	}

	for _, w := range webhooks {
		// Fire and forget individual dispatch to avoid blocking
		go s.dispatch(context.Background(), w, e)
	}

	return nil
}

func (s *WebhookService) dispatch(ctx context.Context, w *webhook.Webhook, e event.Event) {
	logger := s.logger.WithContext(ctx)

	payload, err := json.Marshal(e)
	if err != nil {
		logger.Error("Failed to marshal event for webhook", "webhook_id", w.ID, "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.URL, bytes.NewBuffer(payload))
	if err != nil {
		logger.Error("Failed to create webhook request", "webhook_id", w.ID, "error", err)
		return
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Vyst-Identity-Webhook/1.0")
	req.Header.Set("X-Webhook-ID", w.ID)
	req.Header.Set("X-Event-ID", e.ID)
	req.Header.Set("X-Event-Type", string(e.Type))

	// Start with timestamp for replay attack prevention
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	req.Header.Set("X-Vyst-Timestamp", timestamp)

	// Sign payload
	// Signature = HMAC-SHA256(timestamp + "." + payload)
	sigPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))
	signature := computeHMAC(sigPayload, w.Secret)
	req.Header.Set("X-Vyst-Signature", signature)

	// Send
	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.Error("Webhook delivery failed (network)", "webhook_id", w.ID, "url", w.URL, "error", err)
		// Update status to failed? Retry logic?
		// For MVP, just log.
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger.Debug("Webhook delivered successfully", "webhook_id", w.ID, "status", resp.StatusCode)
	} else {
		logger.Warn("Webhook delivery failed (status)", "webhook_id", w.ID, "status", resp.StatusCode)
	}
}

func computeHMAC(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// Management methods

func (s *WebhookService) CreateWebhook(ctx context.Context, tenantID, url string, events []string, secret string) (*webhook.Webhook, error) {
	if secret == "" {
		secret = uuid.New().String()
	}

	w, err := webhook.NewWebhook(tenantID, url, events, secret)
	if err != nil {
		return nil, err
	}

	if err := s.webhookRepo.Create(ctx, w); err != nil {
		return nil, err
	}

	return w, nil
}

func (s *WebhookService) ListWebhooks(ctx context.Context, tenantID string) ([]*webhook.Webhook, error) {
	return s.webhookRepo.ListByTenant(ctx, tenantID)
}

func (s *WebhookService) DeleteWebhook(ctx context.Context, id string) error {
	return s.webhookRepo.Delete(ctx, id)
}
