package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/webhook"
)

type WebhookRepository struct {
	pool *pgxpool.Pool
}

func NewWebhookRepository(pool *pgxpool.Pool) *WebhookRepository {
	return &WebhookRepository{pool: pool}
}

func (r *WebhookRepository) Create(ctx context.Context, w *webhook.Webhook) error {
	query := `
		INSERT INTO webhooks (id, tenant_id, url, secret, events, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		w.ID, w.TenantID, w.URL, w.Secret, w.Events, w.Status, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}
	return nil
}

func (r *WebhookRepository) GetByID(ctx context.Context, id string) (*webhook.Webhook, error) {
	query := `
		SELECT id, tenant_id, url, secret, events, status, created_at, updated_at
		FROM webhooks WHERE id = $1
	`
	var w webhook.Webhook
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&w.ID, &w.TenantID, &w.URL, &w.Secret, &w.Events, &w.Status, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, webhook.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	return &w, nil
}

func (r *WebhookRepository) ListByTenant(ctx context.Context, tenantID string) ([]*webhook.Webhook, error) {
	query := `
		SELECT id, tenant_id, url, secret, events, status, created_at, updated_at
		FROM webhooks WHERE tenant_id = $1
	`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*webhook.Webhook
	for rows.Next() {
		var w webhook.Webhook
		if err := rows.Scan(&w.ID, &w.TenantID, &w.URL, &w.Secret, &w.Events, &w.Status, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, &w)
	}
	return webhooks, nil
}

func (r *WebhookRepository) ListByEvent(ctx context.Context, tenantID, eventType string) ([]*webhook.Webhook, error) {
	query := `
		SELECT id, tenant_id, url, secret, events, status, created_at, updated_at
		FROM webhooks 
		WHERE tenant_id = $1 AND status = 'active' AND $2 = ANY(events)
	`

	rows, err := r.pool.Query(ctx, query, tenantID, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks by event: %w", err)
	}
	defer rows.Close()

	var webhooks []*webhook.Webhook
	for rows.Next() {
		var w webhook.Webhook
		if err := rows.Scan(&w.ID, &w.TenantID, &w.URL, &w.Secret, &w.Events, &w.Status, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, &w)
	}
	return webhooks, nil
}

func (r *WebhookRepository) Update(ctx context.Context, w *webhook.Webhook) error {
	query := `
		UPDATE webhooks
		SET url = $1, secret = $2, events = $3, status = $4, updated_at = $5
		WHERE id = $6
	`
	cmd, err := r.pool.Exec(ctx, query,
		w.URL, w.Secret, w.Events, w.Status, w.UpdatedAt, w.ID)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return webhook.ErrNotFound
	}
	return nil
}

func (r *WebhookRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return webhook.ErrNotFound
	}
	return nil
}
