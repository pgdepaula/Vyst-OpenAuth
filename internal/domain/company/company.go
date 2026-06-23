// Package company contains the Company domain entity and repository interface.
// This is the core domain layer - no external dependencies allowed.
package company

import (
	"context"
	"errors"
	"time"
)

// Domain errors for company operations.
var (
	// ErrNotFound is returned when a company is not found.
	ErrNotFound = errors.New("company not found")

	// ErrCNPJInvalid is returned when a CNPJ fails validation.
	ErrCNPJInvalid = errors.New("invalid CNPJ")

	// ErrCNPJTaken is returned when a CNPJ is already registered.
	ErrCNPJTaken = errors.New("CNPJ already registered")

	// ErrUserNotMember is returned when a user is not a member of a company.
	ErrUserNotMember = errors.New("user is not a member of this company")

	// ErrInvalidRole is returned when an invalid company role is provided.
	ErrInvalidRole = errors.New("invalid company role")

	// ErrAlreadyMember is returned when trying to add a user who is already a member.
	ErrAlreadyMember = errors.New("user is already a member of this company")
)

// CompanyStatus represents the status of a company.
type CompanyStatus string

const (
	// StatusActive indicates the company is active and operational.
	StatusActive CompanyStatus = "active"

	// StatusPending indicates the company is pending verification.
	StatusPending CompanyStatus = "pending"

	// StatusSuspended indicates the company has been suspended.
	StatusSuspended CompanyStatus = "suspended"
)

// IsValid checks if the company status is valid.
func (s CompanyStatus) IsValid() bool {
	return s == StatusActive || s == StatusPending || s == StatusSuspended
}

// CompanyRole defines the user's role within a company.
type CompanyRole string

const (
	// RoleAdmin has full administrative access to the company.
	RoleAdmin CompanyRole = "admin"

	// RoleMember has standard member access to the company.
	RoleMember CompanyRole = "member"

	// RoleViewer has read-only access to the company.
	RoleViewer CompanyRole = "viewer"
)

// IsValid checks if the company role is valid.
func (r CompanyRole) IsValid() bool {
	return r == RoleAdmin || r == RoleMember || r == RoleViewer
}

// String returns the string representation of the company role.
func (r CompanyRole) String() string {
	return string(r)
}

// Address is a value object representing a company's address.
// It is immutable and compared by value.
type Address struct {
	Logradouro  string // Street name
	Numero      string // Street number
	Complemento string // Additional address info (apt, suite, etc.)
	Bairro      string // Neighborhood
	Cidade      string // City
	UF          string // State (2 characters)
	CEP         string // Postal code (8 digits)
}

// IsEmpty returns true if the address has no data.
func (a Address) IsEmpty() bool {
	return a.Logradouro == "" && a.Cidade == "" && a.UF == "" && a.CEP == ""
}

// Company represents a legal entity (pessoa jurídica) in the system.
// It is the aggregate root for company-related operations.
type Company struct {
	ID                 string        // Unique identifier (UUID)
	TenantID           string        // ID of the tenant this company belongs to
	CNPJ               string        // Unique, validated (14 digits, no formatting)
	RazaoSocial        string        // Legal name (razão social)
	NomeFantasia       string        // Trade name (nome fantasia)
	Endereco           Address       // Company address
	RepresentanteLegal string        // Legal representative name
	Status             CompanyStatus // Company status
	CreatedAt          time.Time     // Timestamp of creation
	UpdatedAt          time.Time     // Timestamp of last update
}

// NewCompany creates a new Company with validation.
func NewCompany(id, tenantID, cnpj, razaoSocial string) (*Company, error) {
	if id == "" {
		return nil, errors.New("company id is required")
	}
	if tenantID == "" {
		return nil, errors.New("tenant id is required")
	}
	if cnpj == "" {
		return nil, errors.New("CNPJ is required")
	}
	if razaoSocial == "" {
		return nil, errors.New("razão social is required")
	}

	// Validate CNPJ format and algorithm
	if !ValidateCNPJ(cnpj) {
		return nil, ErrCNPJInvalid
	}

	now := time.Now()
	return &Company{
		ID:          id,
		TenantID:    tenantID,
		CNPJ:        NormalizeCNPJ(cnpj),
		RazaoSocial: razaoSocial,
		Status:      StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// IsActive returns true if the company is active.
func (c *Company) IsActive() bool {
	return c.Status == StatusActive
}

// Suspend suspends the company.
func (c *Company) Suspend() {
	c.Status = StatusSuspended
	c.UpdatedAt = time.Now()
}

// Activate activates the company.
func (c *Company) Activate() {
	c.Status = StatusActive
	c.UpdatedAt = time.Now()
}

// MembershipStatus represents the status of a user's membership in a company.
type MembershipStatus string

const (
	// MembershipPending indicates the user has been invited but not yet accepted.
	MembershipPending MembershipStatus = "pending"

	// MembershipActive indicates the user is an active member.
	MembershipActive MembershipStatus = "active"

	// MembershipRevoked indicates the user's membership has been revoked.
	MembershipRevoked MembershipStatus = "revoked"
)

// IsValid checks if the membership status is valid.
func (s MembershipStatus) IsValid() bool {
	return s == MembershipPending || s == MembershipActive || s == MembershipRevoked
}

// CompanyUser represents the N:N relationship between User and Company.
// It is a join entity with additional metadata about the relationship.
type CompanyUser struct {
	CompanyID string           // ID of the company
	UserID    string           // ID of the user
	Role      CompanyRole      // User's role in the company
	InvitedBy string           // User ID who invited this user
	JoinedAt  time.Time        // Timestamp when user joined
	Status    MembershipStatus // Membership status
}

// NewCompanyUser creates a new CompanyUser with validation.
func NewCompanyUser(companyID, userID string, role CompanyRole, invitedBy string) (*CompanyUser, error) {
	if companyID == "" {
		return nil, errors.New("company id is required")
	}
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	if !role.IsValid() {
		return nil, ErrInvalidRole
	}

	return &CompanyUser{
		CompanyID: companyID,
		UserID:    userID,
		Role:      role,
		InvitedBy: invitedBy,
		JoinedAt:  time.Now(),
		Status:    MembershipActive,
	}, nil
}

// IsActive returns true if the membership is active.
func (cu *CompanyUser) IsActive() bool {
	return cu.Status == MembershipActive
}

// Revoke revokes the user's membership.
func (cu *CompanyUser) Revoke() {
	cu.Status = MembershipRevoked
}

// Repository defines the contract for company persistence operations.
// Implementations live in the infrastructure layer.
type Repository interface {
	// Create persists a new company to the storage.
	Create(ctx context.Context, company *Company) error

	// GetByID retrieves a company by their unique identifier.
	// Returns ErrNotFound if company doesn't exist.
	GetByID(ctx context.Context, id string) (*Company, error)

	// GetByCNPJ retrieves a company by their CNPJ.
	// Returns ErrNotFound if company doesn't exist.
	GetByCNPJ(ctx context.Context, cnpj string) (*Company, error)

	// Update modifies an existing company's data.
	Update(ctx context.Context, company *Company) error

	// GetByTenantID retrieves all companies for a tenant.
	GetByTenantID(ctx context.Context, tenantID string) ([]*Company, error)

	// Delete removes a company from storage.
	Delete(ctx context.Context, id string) error

	// ListAllActive retrieves all active companies processing (with pagination).
	ListAllActive(ctx context.Context, limit, offset int) ([]*Company, error)
}

// CompanyUserRepository defines the contract for company-user relationship persistence.
// Implementations live in the infrastructure layer.
type CompanyUserRepository interface {
	// AddUser adds a user to a company.
	AddUser(ctx context.Context, cu *CompanyUser) error

	// RemoveUser removes a user from a company.
	RemoveUser(ctx context.Context, companyID, userID string) error

	// GetUserRole returns the user's role in a company.
	// Returns ErrUserNotMember if user is not a member.
	GetUserRole(ctx context.Context, companyID, userID string) (CompanyRole, error)

	// GetCompaniesForUser returns all company memberships for a user.
	GetCompaniesForUser(ctx context.Context, userID string) ([]*CompanyUser, error)

	// GetUsersForCompany returns all user memberships for a company.
	GetUsersForCompany(ctx context.Context, companyID string) ([]*CompanyUser, error)

	// UpdateUserRole updates a user's role in a company.
	UpdateUserRole(ctx context.Context, companyID, userID string, role CompanyRole) error

	// UpdateUserStatus updates a user's membership status.
	UpdateUserStatus(ctx context.Context, companyID, userID string, status MembershipStatus) error
}
