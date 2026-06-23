// Package policy contains the ReBAC (Relationship-Based Access Control) domain types.
// This implements a Zanzibar-lite model for fine-grained authorization.
package policy

import (
	"context"
)

// Tuple represents a relationship: "Subject has Relation to Object".
// Example: "user:123" is "owner" of "tenant:456"
// This is the core primitive of the ReBAC system.
type Tuple struct {
	TenantID string // For multi-tenancy isolation
	Subject  string // e.g., "user:123" or "group:admins"
	Relation string // e.g., "owner", "viewer", "member"
	Object   string // e.g., "tenant:456", "document:789"
}

// Repository defines the contract for policy evaluation and persistence operations.
// Implementations can be in-memory (for testing) or PostgreSQL (for production).
type Repository interface {
	// Check evaluates if subject has relation to object.
	// Returns true if the relationship exists (directly or through graph traversal).
	Check(ctx context.Context, subject, relation, object string) (bool, error)

	// WriteTuple creates a new relationship in the policy store.
	WriteTuple(ctx context.Context, tuple Tuple) error

	// DeleteTuple removes a tuple from the store.
	DeleteTuple(ctx context.Context, tuple Tuple) error

	// GetRolesForUser retrieves all roles assigned to a user.
	// This is a high-level helper to bridge ReBAC tuples to RBAC-style roles.
	GetRolesForUser(ctx context.Context, userID string) ([]string, error)
}

// CheckRequest represents a permission check request.
// Used for batch checking or audit logging.
type CheckRequest struct {
	Subject  string
	Relation string
	Object   string
}

// CheckResult represents the result of a permission check.
type CheckResult struct {
	Allowed bool
	Reason  string // For audit/debugging
}
