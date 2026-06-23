package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/invitation"
)

type InvitationRepository struct {
	pool *pgxpool.Pool
}

func NewInvitationRepository(pool *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{pool: pool}
}

func (r *InvitationRepository) Create(ctx context.Context, inv *invitation.Invitation) error {
	query := `
		INSERT INTO invitations (id, company_id, email, role, token, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		inv.ID,
		inv.CompanyID,
		inv.Email,
		inv.Role,
		inv.Token,
		inv.Status,
		inv.ExpiresAt,
		inv.CreatedAt,
		inv.UpdatedAt,
	)
	return err
}

func (r *InvitationRepository) GetByToken(ctx context.Context, token string) (*invitation.Invitation, error) {
	query := `
		SELECT id, company_id, email, role, token, status, expires_at, created_at, updated_at
		FROM invitations
		WHERE token = $1
	`
	var inv invitation.Invitation
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&inv.ID,
		&inv.CompanyID,
		&inv.Email,
		&inv.Role,
		&inv.Token,
		&inv.Status,
		&inv.ExpiresAt,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, invitation.ErrNotFound
		}
		return nil, err
	}
	return &inv, nil
}

func (r *InvitationRepository) GetByEmailAndCompany(ctx context.Context, email, companyID string) (*invitation.Invitation, error) {
	query := `
		SELECT id, company_id, email, role, token, status, expires_at, created_at, updated_at
		FROM invitations
		WHERE email = $1 AND company_id = $2 AND status = 'pending'
	`
	// Note: checking only pending, as rejected/expired ones don't block new invites usually?
	// But requirements said "GetByEmailAndCompany". Let's assume we want the latest active one or simply any.
	// For "pending invitation exists" check, restricting to pending is best.

	var inv invitation.Invitation
	err := r.pool.QueryRow(ctx, query, email, companyID).Scan(
		&inv.ID,
		&inv.CompanyID,
		&inv.Email,
		&inv.Role,
		&inv.Token,
		&inv.Status,
		&inv.ExpiresAt,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, invitation.ErrNotFound
		}
		return nil, err
	}
	return &inv, nil
}

func (r *InvitationRepository) Update(ctx context.Context, inv *invitation.Invitation) error {
	query := `
		UPDATE invitations
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := r.pool.Exec(ctx, query, inv.Status, inv.UpdatedAt, inv.ID)
	return err
}

func (r *InvitationRepository) ListByCompany(ctx context.Context, companyID string) ([]*invitation.Invitation, error) {
	query := `
		SELECT id, company_id, email, role, token, status, expires_at, created_at, updated_at
		FROM invitations
		WHERE company_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []*invitation.Invitation
	for rows.Next() {
		var inv invitation.Invitation
		if err := rows.Scan(
			&inv.ID,
			&inv.CompanyID,
			&inv.Email,
			&inv.Role,
			&inv.Token,
			&inv.Status,
			&inv.ExpiresAt,
			&inv.CreatedAt,
			&inv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		invitations = append(invitations, &inv)
	}
	return invitations, nil
}
