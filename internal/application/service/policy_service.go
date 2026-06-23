package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
)

// PolicyService handles role and permission management.
type PolicyService struct {
	roleRepo   policy.RoleRepository
	policyRepo policy.Repository
}

// NewPolicyService creates a new PolicyService.
func NewPolicyService(roleRepo policy.RoleRepository, policyRepo policy.Repository) *PolicyService {
	return &PolicyService{
		roleRepo:   roleRepo,
		policyRepo: policyRepo,
	}
}

// CreateRole creates a new role.
func (s *PolicyService) CreateRole(ctx context.Context, name, description, tenantID string, permissions []string) (*policy.Role, error) {
	role := &policy.Role{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Permissions: permissions,
		TenantID:    tenantID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

// GetRole retrieves a role by ID.
func (s *PolicyService) GetRole(ctx context.Context, id string) (*policy.Role, error) {
	return s.roleRepo.GetByID(ctx, id)
}

// ListRoles retrieves all roles for a tenant.
func (s *PolicyService) ListRoles(ctx context.Context, tenantID string) ([]*policy.Role, error) {
	return s.roleRepo.List(ctx, tenantID)
}

// UpdateRole updates an existing role.
func (s *PolicyService) UpdateRole(ctx context.Context, id, name, description string, permissions []string) (*policy.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	role.Name = name
	role.Description = description
	role.Permissions = permissions
	role.UpdatedAt = time.Now()

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

// DeleteRole deletes a role.
func (s *PolicyService) DeleteRole(ctx context.Context, id string) error {
	return s.roleRepo.Delete(ctx, id)
}

// CheckPermission checks if a subject has permission to perform an action on a resource.
func (s *PolicyService) CheckPermission(ctx context.Context, subject, relation, object string) (bool, error) {
	return s.policyRepo.Check(ctx, subject, relation, object)
}
