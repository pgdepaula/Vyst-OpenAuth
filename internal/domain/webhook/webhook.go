package webhook

import (
	"errors"
	"net/url"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidURL  = errors.New("invalid webhook URL")
	ErrEmptyEvents = errors.New("at least one event must be subscribed")
	ErrNotFound    = errors.New("webhook not found")
)

type WebhookStatus string

const (
	StatusActive   WebhookStatus = "active"
	StatusInactive WebhookStatus = "inactive"
	StatusFailed   WebhookStatus = "failed" // Temporarily disabled due to failures
)

type Webhook struct {
	ID        string
	TenantID  string
	URL       string
	Secret    string   // For HMAC signature
	Events    []string // List of event types to subscribe to
	Status    WebhookStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewWebhook(tenantID, targetURL string, events []string, secret string) (*Webhook, error) {
	if tenantID == "" {
		return nil, errors.New("tenant id is required")
	}
	if _, err := url.ParseRequestURI(targetURL); err != nil {
		return nil, ErrInvalidURL
	}
	if len(events) == 0 {
		return nil, ErrEmptyEvents
	}

	now := time.Now()
	return &Webhook{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		URL:       targetURL,
		Secret:    secret,
		Events:    events,
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
