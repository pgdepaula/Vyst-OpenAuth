package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// LogEntry represents a single audit log entry.
type LogEntry struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	ActorID   string                 `json:"actor_id"` // User ID who performed the action
	Action    string                 `json:"action"`   // created, updated, deleted, login, etc.
	Entity    string                 `json:"entity"`   // company, user, etc.
	EntityID  string                 `json:"entity_id"`
	Metadata  map[string]interface{} `json:"metadata"` // diffs, details, etc.
	Timestamp time.Time              `json:"timestamp"`
}

// NewLogEntry creates a new audit log entry.
func NewLogEntry(tenantID, actorID, action, entity, entityID string, metadata map[string]interface{}) *LogEntry {
	return &LogEntry{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		ActorID:   actorID,
		Action:    action,
		Entity:    entity,
		EntityID:  entityID,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}
}

// Repository defines the interface for audit log persistence.
type Repository interface {
	// Create persists a new audit log entry.
	Create(ctx context.Context, entry *LogEntry) error

	// ListByTenant retrieves audit logs for a tenant with pagination.
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*LogEntry, error)
}
