// Package tenant contains the Tenant domain entity and repository interface.
// This is the core domain layer - no external dependencies allowed.
package tenant

import (
	"context"
	"time"
)

// Status represents the possible states of a tenant.
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusPending   Status = "pending"
)

// Tenant represents an organization/company in the multi-tenant system.
// Each tenant has isolated data and users.
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsActive returns true if the tenant is in active status.
func (t *Tenant) IsActive() bool {
	return t.Status == StatusActive
}

// Repository defines the contract for tenant persistence operations.
// Implementations live in the infrastructure layer.
type Repository interface {
	// Create persists a new tenant to the storage.
	Create(ctx context.Context, tenant *Tenant) error

	// GetByID retrieves a tenant by their unique identifier.
	// Returns ErrNotFound if tenant doesn't exist.
	GetByID(ctx context.Context, id string) (*Tenant, error)

	// Update modifies an existing tenant's data.
	Update(ctx context.Context, tenant *Tenant) error

	// SetCurrentTenant sets the current tenant context for RLS.
	// This is required when performing operations that must be scoped to a specific tenant.
	SetCurrentTenant(ctx context.Context, tenantID string) error

	// List retrieves all tenants (for Super Admin).
	List(ctx context.Context) ([]*Tenant, error)
}
