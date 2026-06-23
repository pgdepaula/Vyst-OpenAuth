// Package event contains domain event definitions and the publisher interface.
package event

import (
	"context"
	"time"
)

// EventType defines the type of event.
type EventType string

const (
	// User events
	UserCreated   EventType = "Identity.UserCreated"
	UserSuspended EventType = "Identity.UserSuspended"
	UserUpdated   EventType = "Identity.UserUpdated"
	UserLoggedIn  EventType = "Identity.UserLoggedIn"
	CPFVerified   EventType = "Identity.CPFVerified"

	// Tenant events
	TenantProvisioned EventType = "Identity.TenantProvisioned"
	TenantSuspended   EventType = "Identity.TenantSuspended"

	// Session events
	SessionCreated    EventType = "Identity.SessionCreated"
	SessionTerminated EventType = "Identity.SessionTerminated"

	// Company events
	CompanyCreated     EventType = "Identity.CompanyCreated"
	CompanyUpdated     EventType = "Identity.CompanyUpdated"
	CompanyUserAdded   EventType = "Identity.CompanyUserAdded"
	CompanyUserRemoved EventType = "Identity.CompanyUserRemoved"
	CompanySuspended   EventType = "Identity.CompanySuspended"
	CompanyActivated   EventType = "Identity.CompanyActivated"
	CompanyInfoFetched EventType = "Identity.CompanyInfoFetched"
)

// Event represents a domain event in the system.
type Event struct {
	ID            string      `json:"id"`
	Type          EventType   `json:"type"`
	AggregateType string      `json:"aggregate_type,omitempty"`
	AggregateID   string      `json:"aggregate_id,omitempty"`
	Source        string      `json:"source"`
	Timestamp     time.Time   `json:"timestamp"`
	Payload       interface{} `json:"payload"`
	TenantID      string      `json:"tenant_id,omitempty"`
}

// Publisher defines the interface for publishing events.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// UserCreatedPayload is the payload for UserCreated event.
type UserCreatedPayload struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role,omitempty"`
}

// UserSuspendedPayload is the payload for UserSuspended event.
type UserSuspendedPayload struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason"`
}

// CPFVerifiedPayload is the payload for CPFVerified event.
type CPFVerifiedPayload struct {
	UserID string `json:"user_id"`
	CPF    string `json:"cpf"` // Masked for security
}

// TenantProvisionedPayload is the payload for TenantProvisioned event.
type TenantProvisionedPayload struct {
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
}

// TenantSuspendedPayload is the payload for TenantSuspended event.
type TenantSuspendedPayload struct {
	TenantID string `json:"tenant_id"`
	Reason   string `json:"reason"`
}

// CompanyCreatedPayload is the payload for CompanyCreated event.
type CompanyCreatedPayload struct {
	CompanyID   string `json:"company_id"`
	TenantID    string `json:"tenant_id"`
	CNPJ        string `json:"cnpj"`
	RazaoSocial string `json:"razao_social"`
}

// CompanyUserAddedPayload is the payload for CompanyUserAdded event.
type CompanyUserAddedPayload struct {
	CompanyID string `json:"company_id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
}

// CompanyUserRemovedPayload is the payload for CompanyUserRemoved event.
type CompanyUserRemovedPayload struct {
	CompanyID string `json:"company_id"`
	UserID    string `json:"user_id"`
}
