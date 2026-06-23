package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/tenant"
)

// TenantRepository implements tenant.Repository using PostgreSQL.
type TenantRepository struct {
	pool *pgxpool.Pool
}

// NewTenantRepository creates a new TenantRepository.
func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

// Create persists a new tenant to the database.
func (r *TenantRepository) Create(ctx context.Context, t *tenant.Tenant) error {
	query := `
		INSERT INTO tenants (id, name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		t.ID,
		t.Name,
		string(t.Status),
		t.CreatedAt,
		t.UpdatedAt,
	)
	return err
}

// GetByID retrieves a tenant by their unique identifier.
func (r *TenantRepository) GetByID(ctx context.Context, id string) (*tenant.Tenant, error) {
	query := `
		SELECT id, name, status, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`
	row := GetExecutor(ctx, r.pool).QueryRow(ctx, query, id)
	return r.scanTenant(row)
}

// Update modifies an existing tenant's data.
func (r *TenantRepository) Update(ctx context.Context, t *tenant.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $2, status = $3, updated_at = $4
		WHERE id = $1
	`
	result, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		t.ID,
		t.Name,
		string(t.Status),
		t.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetCurrentTenant sets the current tenant context for RLS.
func (r *TenantRepository) SetCurrentTenant(ctx context.Context, tenantID string) error {
	// Use set_config with is_local=true to scope to the current transaction
	query := `SELECT set_config('app.current_tenant', $1, true)`
	_, err := GetExecutor(ctx, r.pool).Exec(ctx, query, tenantID)
	return err
}

func (r *TenantRepository) scanTenant(row pgx.Row) (*tenant.Tenant, error) {
	t := &tenant.Tenant{}
	var status string
	err := row.Scan(
		&t.ID,
		&t.Name,
		&status,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.Status = tenant.Status(status)
	return t, nil
}

// List retrieves all tenants.
func (r *TenantRepository) List(ctx context.Context) ([]*tenant.Tenant, error) {
	query := `
		SELECT id, name, status, created_at, updated_at
		FROM tenants
		ORDER BY created_at DESC
	`
	rows, err := GetExecutor(ctx, r.pool).Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []*tenant.Tenant
	for rows.Next() {
		t, err := r.scanTenant(rows)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}
