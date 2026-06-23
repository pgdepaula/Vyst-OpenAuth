package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"github.com/pgdepaula/vyst-openauth/internal/domain/tenant"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

// TenantService handles tenant management operations.
type TenantService struct {
	tenantRepo tenant.Repository
	userRepo   user.Repository
	policyRepo policy.Repository
}

// NewTenantService creates a new TenantService.
func NewTenantService(tenantRepo tenant.Repository, userRepo user.Repository, policyRepo policy.Repository) *TenantService {
	return &TenantService{
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
		policyRepo: policyRepo,
	}
}

// CreateTenant creates a new tenant and assigns the owner role to the user.
func (s *TenantService) CreateTenant(ctx context.Context, name, ownerID string) (*tenant.Tenant, error) {
	// 1. Create Tenant
	t := &tenant.Tenant{
		ID:        uuid.New().String(),
		Name:      name,
		Status:    tenant.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.tenantRepo.Create(ctx, t); err != nil {
		return nil, err
	}

	// 2. Assign Owner Role to User
	// Note: In a real system, we'd use a transaction here.
	// For now, we assume eventual consistency or manual cleanup on failure.
	ownerTuple := policy.Tuple{
		TenantID: t.ID,
		Subject:  "user:" + ownerID,
		Relation: "owner",
		Object:   "tenant:" + t.ID,
	}

	if err := s.policyRepo.WriteTuple(ctx, ownerTuple); err != nil {
		// Log error, potentially rollback tenant creation
		return nil, err
	}

	// 3. Update User's TenantID if they don't have one (optional, or handle multi-tenancy on user side)
	// For this MVP, users belong to one tenant primarily, but can own multiple.
	// Let's assume the user context handles the switch.

	return t, nil
}

// ListTenants retrieves all tenants (Super Admin).
func (s *TenantService) ListTenants(ctx context.Context) ([]*tenant.Tenant, error) {
	return s.tenantRepo.List(ctx)
}

// SuspendTenant suspends a tenant.
func (s *TenantService) SuspendTenant(ctx context.Context, id string) error {
	t, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	t.Status = tenant.StatusSuspended
	t.UpdatedAt = time.Now()

	return s.tenantRepo.Update(ctx, t)
}
