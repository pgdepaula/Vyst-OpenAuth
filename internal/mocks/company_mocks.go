package mocks

import (
	"context"

	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
)

// --- Company Repository Mock ---

type MockCompanyRepository struct {
	GetByIDFunc       func(ctx context.Context, id string) (*company.Company, error)
	GetByCNPJFunc     func(ctx context.Context, cnpj string) (*company.Company, error)
	CreateFunc        func(ctx context.Context, c *company.Company) error
	UpdateFunc        func(ctx context.Context, c *company.Company) error
	GetByTenantIDFunc func(ctx context.Context, tenantID string) ([]*company.Company, error)
	DeleteFunc        func(ctx context.Context, id string) error
	ListAllActiveFunc func(ctx context.Context, limit, offset int) ([]*company.Company, error)

	GetByIDCalls []string
	CreateCalls  []*company.Company
}

func (m *MockCompanyRepository) GetByID(ctx context.Context, id string) (*company.Company, error) {
	m.GetByIDCalls = append(m.GetByIDCalls, id)
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockCompanyRepository) GetByCNPJ(ctx context.Context, cnpj string) (*company.Company, error) {
	if m.GetByCNPJFunc != nil {
		return m.GetByCNPJFunc(ctx, cnpj)
	}
	return nil, nil
}

func (m *MockCompanyRepository) Create(ctx context.Context, c *company.Company) error {
	m.CreateCalls = append(m.CreateCalls, c)
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, c)
	}
	return nil
}

func (m *MockCompanyRepository) Update(ctx context.Context, c *company.Company) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, c)
	}
	return nil
}

func (m *MockCompanyRepository) GetByTenantID(ctx context.Context, tenantID string) ([]*company.Company, error) {
	if m.GetByTenantIDFunc != nil {
		return m.GetByTenantIDFunc(ctx, tenantID)
	}
	return nil, nil
}

func (m *MockCompanyRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockCompanyRepository) ListAllActive(ctx context.Context, limit, offset int) ([]*company.Company, error) {
	if m.ListAllActiveFunc != nil {
		return m.ListAllActiveFunc(ctx, limit, offset)
	}
	return nil, nil
}

// --- Company User Repository Mock ---

type MockCompanyUserRepository struct {
	AddUserFunc             func(ctx context.Context, cu *company.CompanyUser) error
	RemoveUserFunc          func(ctx context.Context, companyID, userID string) error
	GetUserRoleFunc         func(ctx context.Context, companyID, userID string) (company.CompanyRole, error)
	GetCompaniesForUserFunc func(ctx context.Context, userID string) ([]*company.CompanyUser, error)
	GetUsersForCompanyFunc  func(ctx context.Context, companyID string) ([]*company.CompanyUser, error)
	UpdateUserRoleFunc      func(ctx context.Context, companyID, userID string, newRole company.CompanyRole) error
	UpdateUserStatusFunc    func(ctx context.Context, companyID, userID string, status company.MembershipStatus) error

	AddUserCalls []company.CompanyUser
}

func (m *MockCompanyUserRepository) AddUser(ctx context.Context, cu *company.CompanyUser) error {
	if cu != nil {
		m.AddUserCalls = append(m.AddUserCalls, *cu)
	}
	if m.AddUserFunc != nil {
		return m.AddUserFunc(ctx, cu)
	}
	return nil
}

func (m *MockCompanyUserRepository) RemoveUser(ctx context.Context, companyID, userID string) error {
	if m.RemoveUserFunc != nil {
		return m.RemoveUserFunc(ctx, companyID, userID)
	}
	return nil
}

func (m *MockCompanyUserRepository) GetUserRole(ctx context.Context, companyID, userID string) (company.CompanyRole, error) {
	if m.GetUserRoleFunc != nil {
		return m.GetUserRoleFunc(ctx, companyID, userID)
	}
	return "", nil
}

func (m *MockCompanyUserRepository) GetCompaniesForUser(ctx context.Context, userID string) ([]*company.CompanyUser, error) {
	if m.GetCompaniesForUserFunc != nil {
		return m.GetCompaniesForUserFunc(ctx, userID)
	}
	return []*company.CompanyUser{}, nil
}

func (m *MockCompanyUserRepository) GetUsersForCompany(ctx context.Context, companyID string) ([]*company.CompanyUser, error) {
	if m.GetUsersForCompanyFunc != nil {
		return m.GetUsersForCompanyFunc(ctx, companyID)
	}
	return []*company.CompanyUser{}, nil
}

func (m *MockCompanyUserRepository) UpdateUserRole(ctx context.Context, companyID, userID string, newRole company.CompanyRole) error {
	if m.UpdateUserRoleFunc != nil {
		return m.UpdateUserRoleFunc(ctx, companyID, userID, newRole)
	}
	return nil
}

func (m *MockCompanyUserRepository) UpdateUserStatus(ctx context.Context, companyID, userID string, status company.MembershipStatus) error {
	if m.UpdateUserStatusFunc != nil {
		return m.UpdateUserStatusFunc(ctx, companyID, userID, status)
	}
	return nil
}
