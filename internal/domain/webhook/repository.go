package webhook

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, webhook *Webhook) error
	GetByID(ctx context.Context, id string) (*Webhook, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*Webhook, error)
	ListByEvent(ctx context.Context, tenantID, eventType string) ([]*Webhook, error) // Optimized lookup for event dispatch
	Update(ctx context.Context, webhook *Webhook) error
	Delete(ctx context.Context, id string) error
}
