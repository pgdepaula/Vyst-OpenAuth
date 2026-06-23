// Package role contains Role and Permission domain entities.
// This is the core domain layer - no external dependencies allowed.
package role

import (
	"context"
	"time"
)

// Permission represents a specific action that can be performed in the system.
// Follows the format: "module.resource.action" (e.g., "crm.leads.read").
type Permission struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"` // e.g., "crm.leads.read"
	Description string    `json:"description"`
	Module      string    `json:"module"` // e.g., "crm", "arc", "identity"
	CreatedAt   time.Time `json:"created_at"`
}

// Role represents a collection of permissions that can be assigned to users.
type Role struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// HasPermission checks if the role contains a specific permission.
func (r *Role) HasPermission(permissionName string) bool {
	for _, p := range r.Permissions {
		if p.Name == permissionName {
			return true
		}
	}
	return false
}

// Repository defines the contract for role persistence operations.
// Implementations live in the infrastructure layer.
type Repository interface {
	// CreateRole persists a new role to the storage.
	CreateRole(ctx context.Context, role *Role) error

	// GetRoleByID retrieves a role by its unique identifier.
	GetRoleByID(ctx context.Context, id string) (*Role, error)

	// GetRoleByName retrieves a role by its name.
	GetRoleByName(ctx context.Context, name string) (*Role, error)

	// AssignPermission adds a permission to a role.
	AssignPermission(ctx context.Context, roleID, permissionID string) error

	// GetPermissionsByUserID retrieves all permissions for a user (through their roles).
	GetPermissionsByUserID(ctx context.Context, userID string) ([]string, error)
}
