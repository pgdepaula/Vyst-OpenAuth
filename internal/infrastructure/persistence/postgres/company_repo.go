// Package postgres provides PostgreSQL implementations of domain repositories.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
)

// CompanyRepository implements company.Repository using PostgreSQL.
type CompanyRepository struct {
	pool *pgxpool.Pool
}

// NewCompanyRepository creates a new CompanyRepository.
func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{pool: pool}
}

// Create persists a new company to the database.
func (r *CompanyRepository) Create(ctx context.Context, c *company.Company) error {
	query := `
		INSERT INTO companies (
			id, tenant_id, cnpj, razao_social, nome_fantasia,
			logradouro, numero, complemento, bairro, cidade, uf, cep,
			representante_legal, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		c.ID,
		c.TenantID,
		c.CNPJ,
		c.RazaoSocial,
		nullableString(c.NomeFantasia),
		nullableString(c.Endereco.Logradouro),
		nullableString(c.Endereco.Numero),
		nullableString(c.Endereco.Complemento),
		nullableString(c.Endereco.Bairro),
		nullableString(c.Endereco.Cidade),
		nullableString(c.Endereco.UF),
		nullableString(c.Endereco.CEP),
		nullableString(c.RepresentanteLegal),
		string(c.Status),
		c.CreatedAt,
		c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert company: %w", err)
	}

	return nil
}

// GetByID retrieves a company by their unique identifier.
func (r *CompanyRepository) GetByID(ctx context.Context, id string) (*company.Company, error) {
	query := `
		SELECT id, tenant_id, cnpj, razao_social, nome_fantasia,
		       logradouro, numero, complemento, bairro, cidade, uf, cep,
		       representante_legal, status, created_at, updated_at
		FROM companies
		WHERE id = $1
	`
	row := GetExecutor(ctx, r.pool).QueryRow(ctx, query, id)
	return r.scanCompany(row)
}

// GetByCNPJ retrieves a company by their CNPJ.
func (r *CompanyRepository) GetByCNPJ(ctx context.Context, cnpj string) (*company.Company, error) {
	query := `
		SELECT id, tenant_id, cnpj, razao_social, nome_fantasia,
		       logradouro, numero, complemento, bairro, cidade, uf, cep,
		       representante_legal, status, created_at, updated_at
		FROM companies
		WHERE cnpj = $1
	`
	row := GetExecutor(ctx, r.pool).QueryRow(ctx, query, cnpj)
	return r.scanCompany(row)
}

// Update modifies an existing company's data.
func (r *CompanyRepository) Update(ctx context.Context, c *company.Company) error {
	query := `
		UPDATE companies
		SET razao_social = $2,
		    nome_fantasia = $3,
		    logradouro = $4,
		    numero = $5,
		    complemento = $6,
		    bairro = $7,
		    cidade = $8,
		    uf = $9,
		    cep = $10,
		    representante_legal = $11,
		    status = $12,
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		c.ID,
		c.RazaoSocial,
		nullableString(c.NomeFantasia),
		nullableString(c.Endereco.Logradouro),
		nullableString(c.Endereco.Numero),
		nullableString(c.Endereco.Complemento),
		nullableString(c.Endereco.Bairro),
		nullableString(c.Endereco.Cidade),
		nullableString(c.Endereco.UF),
		nullableString(c.Endereco.CEP),
		nullableString(c.RepresentanteLegal),
		string(c.Status),
	)
	if err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}

	if result.RowsAffected() == 0 {
		return company.ErrNotFound
	}

	return nil
}

// GetByTenantID retrieves all companies for a tenant.
func (r *CompanyRepository) GetByTenantID(ctx context.Context, tenantID string) ([]*company.Company, error) {
	query := `
		SELECT id, tenant_id, cnpj, razao_social, nome_fantasia,
		       logradouro, numero, complemento, bairro, cidade, uf, cep,
		       representante_legal, status, created_at, updated_at
		FROM companies
		WHERE tenant_id = $1
		ORDER BY razao_social ASC
	`

	rows, err := GetExecutor(ctx, r.pool).Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query companies: %w", err)
	}
	defer rows.Close()

	var companies []*company.Company
	for rows.Next() {
		c, err := r.scanCompanyFromRows(rows)
		if err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating companies: %w", err)
	}

	return companies, nil
}

// ListAllActive retrieves all active companies processing (with pagination).
func (r *CompanyRepository) ListAllActive(ctx context.Context, limit, offset int) ([]*company.Company, error) {
	query := `
		SELECT id, tenant_id, cnpj, razao_social, nome_fantasia,
		       logradouro, numero, complemento, bairro, cidade, uf, cep,
		       representante_legal, status, created_at, updated_at
		FROM companies
		WHERE status != 'suspended'
		ORDER BY created_at ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := GetExecutor(ctx, r.pool).Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query active companies: %w", err)
	}
	defer rows.Close()

	var companies []*company.Company
	for rows.Next() {
		c, err := r.scanCompanyFromRows(rows)
		if err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating companies: %w", err)
	}

	return companies, nil
}

// Delete removes a company from storage.
func (r *CompanyRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM companies WHERE id = $1`
	result, err := GetExecutor(ctx, r.pool).Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	if result.RowsAffected() == 0 {
		return company.ErrNotFound
	}

	return nil
}

// scanCompany scans a single row into a Company struct.
func (r *CompanyRepository) scanCompany(row pgx.Row) (*company.Company, error) {
	c := &company.Company{}
	var nomeFantasia, logradouro, numero, complemento, bairro, cidade, uf, cep, representanteLegal *string
	var status string

	err := row.Scan(
		&c.ID,
		&c.TenantID,
		&c.CNPJ,
		&c.RazaoSocial,
		&nomeFantasia,
		&logradouro,
		&numero,
		&complemento,
		&bairro,
		&cidade,
		&uf,
		&cep,
		&representanteLegal,
		&status,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, company.ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan company: %w", err)
	}

	// Populate optional fields
	c.Status = company.CompanyStatus(status)
	if nomeFantasia != nil {
		c.NomeFantasia = *nomeFantasia
	}
	if logradouro != nil {
		c.Endereco.Logradouro = *logradouro
	}
	if numero != nil {
		c.Endereco.Numero = *numero
	}
	if complemento != nil {
		c.Endereco.Complemento = *complemento
	}
	if bairro != nil {
		c.Endereco.Bairro = *bairro
	}
	if cidade != nil {
		c.Endereco.Cidade = *cidade
	}
	if uf != nil {
		c.Endereco.UF = *uf
	}
	if cep != nil {
		c.Endereco.CEP = *cep
	}
	if representanteLegal != nil {
		c.RepresentanteLegal = *representanteLegal
	}

	return c, nil
}

// scanCompanyFromRows scans from pgx.Rows into a Company struct.
func (r *CompanyRepository) scanCompanyFromRows(rows pgx.Rows) (*company.Company, error) {
	c := &company.Company{}
	var nomeFantasia, logradouro, numero, complemento, bairro, cidade, uf, cep, representanteLegal *string
	var status string

	err := rows.Scan(
		&c.ID,
		&c.TenantID,
		&c.CNPJ,
		&c.RazaoSocial,
		&nomeFantasia,
		&logradouro,
		&numero,
		&complemento,
		&bairro,
		&cidade,
		&uf,
		&cep,
		&representanteLegal,
		&status,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan company row: %w", err)
	}

	// Populate optional fields (same as scanCompany)
	c.Status = company.CompanyStatus(status)
	if nomeFantasia != nil {
		c.NomeFantasia = *nomeFantasia
	}
	if logradouro != nil {
		c.Endereco.Logradouro = *logradouro
	}
	if numero != nil {
		c.Endereco.Numero = *numero
	}
	if complemento != nil {
		c.Endereco.Complemento = *complemento
	}
	if bairro != nil {
		c.Endereco.Bairro = *bairro
	}
	if cidade != nil {
		c.Endereco.Cidade = *cidade
	}
	if uf != nil {
		c.Endereco.UF = *uf
	}
	if cep != nil {
		c.Endereco.CEP = *cep
	}
	if representanteLegal != nil {
		c.RepresentanteLegal = *representanteLegal
	}

	return c, nil
}

// nullableString returns nil for empty strings, otherwise returns pointer to string.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
