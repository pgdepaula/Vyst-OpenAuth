package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
)

// PolicyRepository implements policy.Engine using PostgreSQL with recursive CTEs.
type PolicyRepository struct {
	pool *pgxpool.Pool
}

// NewPolicyRepository creates a new PolicyRepository.
func NewPolicyRepository(pool *pgxpool.Pool) *PolicyRepository {
	return &PolicyRepository{pool: pool}
}

// parseEntity splits an entity string like "user:123" into type and id.
func parseEntity(entity string) (entityType, entityID string, err error) {
	parts := strings.SplitN(entity, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid entity format: %s (expected 'type:id')", entity)
	}
	return parts[0], parts[1], nil
}

// Check evaluates if subject has relation to object using recursive graph traversal.
func (r *PolicyRepository) Check(ctx context.Context, subject, relation, object string) (bool, error) {
	subType, subID, err := parseEntity(subject)
	if err != nil {
		return false, err
	}
	objType, objID, err := parseEntity(object)
	if err != nil {
		return false, err
	}

	// Recursive CTE to traverse the permission graph
	query := `
	WITH RECURSIVE search_path AS (
		-- Base Case: Direct tuples on the object with the requested relation
		SELECT 
			subject_type, 
			subject_id, 
			1 as depth
		FROM policy_tuples
		WHERE entity_type = $1 AND entity_id = $2 AND relation = $3
		
		UNION ALL
		
		-- Recursive Step: Expand groups via 'member' relation
		SELECT 
			pt.subject_type, 
			pt.subject_id, 
			sp.depth + 1
		FROM policy_tuples pt
		JOIN search_path sp ON pt.entity_type = sp.subject_type AND pt.entity_id = sp.subject_id
		WHERE pt.relation = 'member'
		AND sp.depth < 10 -- Safety limit to prevent infinite loops
	)
	SELECT EXISTS (
		SELECT 1 FROM search_path 
		WHERE subject_type = $4 AND subject_id = $5
	);
	`

	var allowed bool
	err = GetExecutor(ctx, r.pool).QueryRow(ctx, query, objType, objID, relation, subType, subID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("failed to execute permission check: %w", err)
	}

	return allowed, nil
}

// WriteTuple creates a new relationship in the policy store.
func (r *PolicyRepository) WriteTuple(ctx context.Context, tuple policy.Tuple) error {
	subType, subID, err := parseEntity(tuple.Subject)
	if err != nil {
		return err
	}
	objType, objID, err := parseEntity(tuple.Object)
	if err != nil {
		return err
	}

	// Use provided TenantID or default
	tenantID := tuple.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000000"
	}

	query := `
		INSERT INTO policy_tuples (tenant_id, entity_type, entity_id, relation, subject_type, subject_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, entity_type, entity_id, relation, subject_type, subject_id) DO NOTHING
	`

	_, err = GetExecutor(ctx, r.pool).Exec(ctx, query, tenantID, objType, objID, tuple.Relation, subType, subID)
	return err
}

// DeleteTuple removes a relationship from the policy store.
func (r *PolicyRepository) DeleteTuple(ctx context.Context, tuple policy.Tuple) error {
	subType, subID, err := parseEntity(tuple.Subject)
	if err != nil {
		return err
	}
	objType, objID, err := parseEntity(tuple.Object)
	if err != nil {
		return err
	}

	tenantID := tuple.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000000"
	}

	query := `
		DELETE FROM policy_tuples
		WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3 
		AND relation = $4 AND subject_type = $5 AND subject_id = $6
	`

	_, err = GetExecutor(ctx, r.pool).Exec(ctx, query, tenantID, objType, objID, tuple.Relation, subType, subID)
	return err
}

// GetRolesForUser retrieves all roles assigned to a user.
func (r *PolicyRepository) GetRolesForUser(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT relation
		FROM policy_tuples
		WHERE subject_type = 'user' AND subject_id = $1
		AND entity_type LIKE 'tenant' -- Roles are typically relations on tenants
	`

	rows, err := GetExecutor(ctx, r.pool).Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating roles: %w", err)
	}

	return roles, nil
}
