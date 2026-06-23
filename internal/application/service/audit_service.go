package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/audit"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
)

// AuditService listens to domain events and creates audit logs.
type AuditService struct {
	auditRepo audit.Repository
	eventBus  event.Bus
	logger    ports.Logger
}

func NewAuditService(auditRepo audit.Repository, eventBus event.Bus, logger ports.Logger) *AuditService {
	s := &AuditService{
		auditRepo: auditRepo,
		eventBus:  eventBus,
		logger:    logger,
	}

	// Subscribe to all events we want to audit
	s.subscribeToEvents()
	return s
}

func (s *AuditService) subscribeToEvents() {
	// Subscribe to everything for audit purposes? Or specific relevant events?
	// To minimize noise, let's start with critical changes.
	eventTypes := []event.EventType{
		event.CompanyCreated,
		event.CompanyUpdated,
		event.CompanySuspended,
		event.CompanyActivated,
		event.CompanyUserAdded,
		event.CompanyUserRemoved,
		event.UserCreated,
		event.UserSuspended,
		event.UserUpdated,
	}

	for _, et := range eventTypes {
		s.eventBus.Subscribe(et, s.handleEvent)
	}
}

func (s *AuditService) handleEvent(ctx context.Context, e event.Event) error {
	// Map event helper
	// We need ActorID.
	// The Event struct has Source, but usually not ActorID directly unless in payload or metadata.
	// But our Event struct definition:
	/*
		type Event struct {
			...
			Payload       interface{} `json:"payload"`
		}
	*/

	// We will try to extract actor_id from payload if available.
	// Or we might need to enhance Event struct to carry ActorID (User ID who triggered).
	// For now, let's look into map payload.

	payloadMap, ok := e.Payload.(map[string]interface{})
	if !ok {
		// Maybe it was unmarshalled into specific struct, try to convert back to map or inspect
		// This depends on how EventBus publishes. If passing structs, we need JSON reflection.
		// Let's coerce to map via JSON.
		b, _ := json.Marshal(e.Payload)
		_ = json.Unmarshal(b, &payloadMap)
	}

	actorID := "system"
	if val, ok := payloadMap["actor_id"]; ok {
		actorID = fmt.Sprintf("%v", val)
	} else if val, ok := payloadMap["user_id"]; ok {
		// Sometimes user_id is the actor (e.g. self-registration)
		// But in admin actions, user_id is the target.
		// We'll fallback to system if ambiguity.
		// For MVP, relying on payload is brittle but works if we enforce consistent payloads.
		_ = val
	}

	// Simplify changes capture
	changes := make(map[string]interface{})
	for k, v := range payloadMap {
		changes[k] = v
	}

	entry := audit.NewLogEntry(
		e.TenantID,
		actorID,
		string(e.Type),
		e.AggregateType,
		e.AggregateID,
		changes,
	)
	entry.Timestamp = e.Timestamp

	if err := s.auditRepo.Create(context.Background(), entry); err != nil {
		s.logger.Error("Failed to create audit log", "event_id", e.ID, "error", err)
		return err
	}

	return nil
}

// ListLogs provides access to audit logs for admin UI.
func (s *AuditService) ListLogs(ctx context.Context, tenantID string, limit, offset int) ([]*audit.LogEntry, error) {
	return s.auditRepo.ListByTenant(ctx, tenantID, limit, offset)
}
