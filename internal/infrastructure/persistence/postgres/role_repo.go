package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
)

// RoleRepository implements policy.RoleRepository for PostgreSQL.
type RoleRepository struct {
	pool *pgxpool.Pool
}

// NewRoleRepository creates a new RoleRepository.
func NewRoleRepository(pool *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{pool: pool}
}

// Create persists a new role.
func (r *RoleRepository) Create(ctx context.Context, role *policy.Role) error {
	query := `
		INSERT INTO roles (id, name, description, permissions, tenant_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.Permissions,
		role.TenantID,
		role.CreatedAt,
		role.UpdatedAt,
	)
	return err
}

// GetByID retrieves a role by ID.
func (r *RoleRepository) GetByID(ctx context.Context, id string) (*policy.Role, error) {
	query := `
		SELECT id, name, description, permissions, tenant_id, created_at, updated_at
		FROM roles
		WHERE id = $1
	`
	row := GetExecutor(ctx, r.pool).QueryRow(ctx, query, id)
	return scanRole(row)
}

// List retrieves all roles for a tenant.
func (r *RoleRepository) List(ctx context.Context, tenantID string) ([]*policy.Role, error) {
	query := `
		SELECT id, name, description, permissions, tenant_id, created_at, updated_at
		FROM roles
		WHERE tenant_id = $1
		ORDER BY name ASC
	`
	rows, err := GetExecutor(ctx, r.pool).Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*policy.Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// Update updates an existing role.
func (r *RoleRepository) Update(ctx context.Context, role *policy.Role) error {
	query := `
		UPDATE roles
		SET name = $2, description = $3, permissions = $4, updated_at = $5
		WHERE id = $1
	`
	cmdTag, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.Permissions,
		role.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return policy.ErrRoleNotFound
	}
	return nil
}

// Delete removes a role.
func (r *RoleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM roles WHERE id = $1`
	cmdTag, err := GetExecutor(ctx, r.pool).Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return policy.ErrRoleNotFound
	}
	return nil
}

func scanRole(row pgx.Row) (*policy.Role, error) {
	var r policy.Role
	err := row.Scan(
		&r.ID,
		&r.Name,
		&r.Description,
		&r.Permissions,
		&r.TenantID,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, policy.ErrRoleNotFound
		}
		return nil, err
	}
	return &r, nil
}
