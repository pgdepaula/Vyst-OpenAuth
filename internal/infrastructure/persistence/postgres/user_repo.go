package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

// UserRepository implements user.Repository using PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

const userByIDQuery = `
	SELECT id, email, password_hash, tenant_id, cpf, created_at, updated_at, reset_token, reset_token_expires_at
	FROM users
	WHERE id = $1
`

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create persists a new user to the database.
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, tenant_id, cpf, created_at, updated_at, reset_token, reset_token_expires_at, status, verification_token, verification_token_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		u.ID,
		u.Email,
		u.PasswordHash,
		u.TenantID,
		u.CPF,
		u.CreatedAt,
		u.UpdatedAt,
		u.ResetToken,
		u.ResetTokenExpiresAt,
		u.Status,
		u.VerificationToken,
		u.VerificationTokenExpiresAt,
	)
	return err
}

// GetByEmail retrieves a user by email.
// It uses a SECURITY DEFINER function to bypass RLS since the tenant context
// might not be known at the time of login.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `
		SELECT id, email, password_hash, tenant_id, cpf, created_at, updated_at, reset_token, reset_token_expires_at, status, verification_token, verification_token_expires_at
		FROM get_user_by_email_secure($1)
	`
	row := GetExecutor(ctx, r.pool).QueryRow(ctx, query, email)

	var u user.User
	var resetTokenExpiresAt *time.Time
	var verificationTokenExpiresAt *time.Time

	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.TenantID,
		&u.CPF,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.ResetToken,
		&resetTokenExpiresAt,
		&u.Status,
		&u.VerificationToken,
		&verificationTokenExpiresAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	if resetTokenExpiresAt != nil {
		u.ResetTokenExpiresAt = *resetTokenExpiresAt
	}
	if verificationTokenExpiresAt != nil {
		u.VerificationTokenExpiresAt = *verificationTokenExpiresAt
	}

	return &u, nil
}

// GetByID retrieves a user by their unique identifier.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	tenantID, _ := ctx.Value("tenant_id").(string)
	if tenantID != "" {
		return r.getByIDWithTenant(ctx, id, tenantID)
	}

	return r.scanUser(r.pool.QueryRow(ctx, userByIDQuery, id))
}

func (r *UserRepository) getByIDWithTenant(ctx context.Context, id, tenantID string) (*user.User, error) {
	if err := validateTenantContext(tenantID); err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	u, err := r.scanUser(tx.QueryRow(ctx, userByIDQuery, id))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}
	committed = true
	return u, nil
}

func validateTenantContext(tenantID string) error {
	if len(tenantID) > 36 {
		return fmt.Errorf("invalid tenant_id length")
	}
	for _, r := range tenantID {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '-') {
			return fmt.Errorf("invalid characters in tenant_id")
		}
	}
	return nil
}

// Update modifies an existing user's data.
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users
		SET email = $2, password_hash = $3, cpf = $4, updated_at = $5, reset_token = $6, reset_token_expires_at = $7, status = $8, verification_token = $9, verification_token_expires_at = $10
		WHERE id = $1
	`
	// Handle nullable times for Update
	var resetTokenExpiresAt *time.Time
	if !u.ResetTokenExpiresAt.IsZero() {
		resetTokenExpiresAt = &u.ResetTokenExpiresAt
	}
	var verificationTokenExpiresAt *time.Time
	if !u.VerificationTokenExpiresAt.IsZero() {
		verificationTokenExpiresAt = &u.VerificationTokenExpiresAt
	}

	result, err := GetExecutor(ctx, r.pool).Exec(ctx, query,
		u.ID,
		u.Email,
		u.PasswordHash,
		u.CPF,
		time.Now(),
		u.ResetToken,
		resetTokenExpiresAt,
		u.Status,
		u.VerificationToken,
		verificationTokenExpiresAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *UserRepository) scanUser(row pgx.Row) (*user.User, error) {
	u := &user.User{}
	var resetTokenExpiresAt *time.Time

	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.TenantID,
		&u.CPF,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.ResetToken,
		&resetTokenExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrNotFound
		}
		return nil, err
	}

	if resetTokenExpiresAt != nil {
		u.ResetTokenExpiresAt = *resetTokenExpiresAt
	}

	return u, nil
}

// GetByResetToken retrieves a user by their reset token.
func (r *UserRepository) GetByResetToken(ctx context.Context, token string) (*user.User, error) {
	query := `
		SELECT id, email, password_hash, tenant_id, cpf, created_at, updated_at, reset_token, reset_token_expires_at
		FROM users
		WHERE reset_token = $1
	`
	row := GetExecutor(ctx, r.pool).QueryRow(ctx, query, token)
	return r.scanUser(row)
}

// GetByVerificationToken retrieves a user by their verification token.
func (r *UserRepository) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	query := `
		SELECT id, email, password_hash, tenant_id, cpf, created_at, updated_at, reset_token, reset_token_expires_at, status, verification_token, verification_token_expires_at
		FROM users
		WHERE verification_token = $1
	`
	row := GetExecutor(ctx, r.pool).QueryRow(ctx, query, token)

	var u user.User
	var resetTokenExpiresAt *time.Time
	var verificationTokenExpiresAt *time.Time

	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.TenantID,
		&u.CPF,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.ResetToken,
		&resetTokenExpiresAt,
		&u.Status,
		&u.VerificationToken,
		&verificationTokenExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrNotFound
		}
		return nil, err
	}

	if resetTokenExpiresAt != nil {
		u.ResetTokenExpiresAt = *resetTokenExpiresAt
	}
	if verificationTokenExpiresAt != nil {
		u.VerificationTokenExpiresAt = *verificationTokenExpiresAt
	}

	return &u, nil
}
