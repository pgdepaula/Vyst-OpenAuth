package policy

import (
	"context"
	"errors"
	"time"
)

var (
	ErrRoleNotFound = errors.New("role not found")
)

// Role represents a high-level role definition.
// It groups a set of permissions (which might be translated to ReBAC tuples or checked directly).
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"` // e.g., ["user:read", "user:write"]
	TenantID    string    `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleRepository defines the storage contract for Roles.
type RoleRepository interface {
	Create(ctx context.Context, role *Role) error
	GetByID(ctx context.Context, id string) (*Role, error)
	List(ctx context.Context, tenantID string) ([]*Role, error)
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id string) error
}
