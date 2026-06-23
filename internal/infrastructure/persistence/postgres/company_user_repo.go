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

// CompanyUserRepository implements company.CompanyUserRepository using PostgreSQL.
type CompanyUserRepository struct {
	pool *pgxpool.Pool
}

// NewCompanyUserRepository creates a new CompanyUserRepository.
func NewCompanyUserRepository(pool *pgxpool.Pool) *CompanyUserRepository {
	return &CompanyUserRepository{pool: pool}
}

// AddUser adds a user to a company.
func (r *CompanyUserRepository) AddUser(ctx context.Context, cu *company.CompanyUser) error {
	query := `
		INSERT INTO company_users (company_id, user_id, role, invited_by, joined_at, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		cu.CompanyID,
		cu.UserID,
		string(cu.Role),
		nullableString(cu.InvitedBy),
		cu.JoinedAt,
		string(cu.Status),
	)
	if err != nil {
		return fmt.Errorf("failed to add user to company: %w", err)
	}

	return nil
}

// RemoveUser removes a user from a company.
func (r *CompanyUserRepository) RemoveUser(ctx context.Context, companyID, userID string) error {
	query := `DELETE FROM company_users WHERE company_id = $1 AND user_id = $2`
	result, err := GetExecutor(ctx, r.pool).Exec(ctx, query, companyID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove user from company: %w", err)
	}

	if result.RowsAffected() == 0 {
		return company.ErrUserNotMember
	}

	return nil
}

// GetUserRole returns the user's role in a company.
func (r *CompanyUserRepository) GetUserRole(ctx context.Context, companyID, userID string) (company.CompanyRole, error) {
	query := `
		SELECT role FROM company_users 
		WHERE company_id = $1 AND user_id = $2 AND status = 'active'
	`

	var role string
	err := GetExecutor(ctx, r.pool).QueryRow(ctx, query, companyID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", company.ErrUserNotMember
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return company.CompanyRole(role), nil
}

// GetCompaniesForUser returns all company memberships for a user.
func (r *CompanyUserRepository) GetCompaniesForUser(ctx context.Context, userID string) ([]*company.CompanyUser, error) {
	query := `
		SELECT company_id, user_id, role, invited_by, joined_at, status
		FROM company_users
		WHERE user_id = $1
		ORDER BY joined_at DESC
	`

	rows, err := GetExecutor(ctx, r.pool).Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query company users: %w", err)
	}
	defer rows.Close()

	var users []*company.CompanyUser
	for rows.Next() {
		cu, err := r.scanCompanyUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, cu)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating company users: %w", err)
	}

	return users, nil
}

// GetUsersForCompany returns all user memberships for a company.
func (r *CompanyUserRepository) GetUsersForCompany(ctx context.Context, companyID string) ([]*company.CompanyUser, error) {
	query := `
		SELECT company_id, user_id, role, invited_by, joined_at, status
		FROM company_users
		WHERE company_id = $1
		ORDER BY role ASC, joined_at ASC
	`

	rows, err := GetExecutor(ctx, r.pool).Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query company users: %w", err)
	}
	defer rows.Close()

	var users []*company.CompanyUser
	for rows.Next() {
		cu, err := r.scanCompanyUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, cu)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating company users: %w", err)
	}

	return users, nil
}

// UpdateUserRole updates a user's role in a company.
func (r *CompanyUserRepository) UpdateUserRole(ctx context.Context, companyID, userID string, role company.CompanyRole) error {
	query := `
		UPDATE company_users
		SET role = $3
		WHERE company_id = $1 AND user_id = $2
	`

	result, err := GetExecutor(ctx, r.pool).Exec(ctx, query, companyID, userID, string(role))
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return company.ErrUserNotMember
	}

	return nil
}

// UpdateUserStatus updates a user's membership status.
func (r *CompanyUserRepository) UpdateUserStatus(ctx context.Context, companyID, userID string, status company.MembershipStatus) error {
	query := `
		UPDATE company_users
		SET status = $3
		WHERE company_id = $1 AND user_id = $2
	`

	result, err := GetExecutor(ctx, r.pool).Exec(ctx, query, companyID, userID, string(status))
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return company.ErrUserNotMember
	}

	return nil
}

// scanCompanyUser scans a row into a CompanyUser struct.
func (r *CompanyUserRepository) scanCompanyUser(rows pgx.Rows) (*company.CompanyUser, error) {
	cu := &company.CompanyUser{}
	var role, status string
	var invitedBy *string

	err := rows.Scan(
		&cu.CompanyID,
		&cu.UserID,
		&role,
		&invitedBy,
		&cu.JoinedAt,
		&status,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan company user: %w", err)
	}

	cu.Role = company.CompanyRole(role)
	cu.Status = company.MembershipStatus(status)
	if invitedBy != nil {
		cu.InvitedBy = *invitedBy
	}

	return cu, nil
}
