// Package company contains the Company domain entity and repository interface.
// This file contains domain event definitions for company-related operations.
package company

// Event types for company domain.
// These constants define the event types published when company-related actions occur.
const (
	// EventCompanyCreated is published when a new company is created.
	EventCompanyCreated = "Identity.CompanyCreated"

	// EventCompanyUserAdded is published when a user is added to a company.
	EventCompanyUserAdded = "Identity.CompanyUserAdded"

	// EventCompanyUserRemoved is published when a user is removed from a company.
	EventCompanyUserRemoved = "Identity.CompanyUserRemoved"

	// EventCompanyUpdated is published when a company is updated.
	EventCompanyUpdated = "Identity.CompanyUpdated"

	// EventCompanySuspended is published when a company is suspended.
	EventCompanySuspended = "Identity.CompanySuspended"

	// EventCompanyActivated is published when a company is activated.
	EventCompanyActivated = "Identity.CompanyActivated"

	// EventCompanyInfoFetched is published when company info is fetched from an external source.
	EventCompanyInfoFetched = "Identity.CompanyInfoFetched"
)

// CompanyCreatedPayload is the payload for CompanyCreated event.
type CompanyCreatedPayload struct {
	CompanyID   string `json:"company_id"`
	CNPJ        string `json:"cnpj"`
	RazaoSocial string `json:"razao_social"`
	TenantID    string `json:"tenant_id"`
	CreatedBy   string `json:"created_by"`
}

// CompanyUserAddedPayload is the payload for CompanyUserAdded event.
type CompanyUserAddedPayload struct {
	CompanyID string `json:"company_id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	InvitedBy string `json:"invited_by"`
}

// CompanyUserRemovedPayload is the payload for CompanyUserRemoved event.
type CompanyUserRemovedPayload struct {
	CompanyID string `json:"company_id"`
	UserID    string `json:"user_id"`
	RemovedBy string `json:"removed_by"`
	Reason    string `json:"reason,omitempty"`
}

// CompanyUpdatedPayload is the payload for CompanyUpdated event.
type CompanyUpdatedPayload struct {
	CompanyID string `json:"company_id"`
	UpdatedBy string `json:"updated_by"`
}

// CompanySuspendedPayload is the payload for CompanySuspended event.
type CompanySuspendedPayload struct {
	CompanyID   string `json:"company_id"`
	SuspendedBy string `json:"suspended_by"`
	Reason      string `json:"reason"`
}

// CompanyActivatedPayload is the payload for CompanyActivated event.
type CompanyActivatedPayload struct {
	CompanyID   string `json:"company_id"`
	ActivatedBy string `json:"activated_by"`
}

// CompanyInfoFetchedPayload is the payload for CompanyInfoFetched event.
type CompanyInfoFetchedPayload struct {
	CNPJ        string `json:"cnpj"`
	TriggeredBy string `json:"triggered_by,omitempty"` // UserID that triggered the search, if any
	Provider    string `json:"provider"`               // Provider name (e.g., "BrasilAPI")
	Found       bool   `json:"found"`
}
