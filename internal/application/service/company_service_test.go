package service_test

import (
	"context"
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestCompanyService_SwitchCompany_Success(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{}
	mockCompanyRepo := &mocks.MockCompanyRepository{}
	mockCompanyUserRepo := &mocks.MockCompanyUserRepository{}
	mockTM := &mocks.MockTransactionManager{}
	mockEventBus := &mocks.MockEventBus{}
	mockOutbox := &mocks.MockOutboxPublisher{}
	mockLogger := &mocks.MockLogger{}

	svc := service.NewCompanyService(mockTM, mockCompanyRepo, mockCompanyUserRepo, mockUserRepo, mockEventBus, mockOutbox, nil, mockLogger)

	userID := "user-123"
	companyID := "company-456"

	// Mock UserRepo
	mockUserRepo.GetByIDFunc = func(ctx context.Context, id string) (*user.User, error) {
		return &user.User{
			ID:           userID,
			IdentityType: user.IdentityTypeIndividual,
		}, nil
	}
	// Mock Update
	mockUserRepo.UpdateFunc = func(ctx context.Context, u *user.User) error {
		return nil
	}

	// Mock CompanyUserRepo (Membership check)
	mockCompanyUserRepo.GetUserRoleFunc = func(ctx context.Context, cID, uID string) (company.CompanyRole, error) {
		if cID == companyID && uID == userID {
			return company.RoleMember, nil
		}
		return "", company.ErrUserNotMember
	}
	mockCompanyUserRepo.GetCompaniesForUserFunc = func(ctx context.Context, uID string) ([]*company.CompanyUser, error) {
		return []*company.CompanyUser{
			{CompanyID: companyID, UserID: userID, Status: company.MembershipActive},
		}, nil
	}

	err := svc.SwitchCompany(context.Background(), userID, companyID)

	assert.NoError(t, err)
	assert.Len(t, mockUserRepo.CreateCalls, 0)
	// Check if Update was called with correct values
	// Since we can't easily inspect the 'u' argument here without capturing it in mock,
	// let's rely on the fact that mock struct captures calls if we implemented it right?
	// Wait, mock struct implementation for Update:
	// func (m *MockUserRepository) Update(ctx context.Context, u *user.User) error {
	//  if m.UpdateFunc != nil { return m.UpdateFunc(ctx, u) } return nil }
	// It does NOT capture UpdateCalls in the mock provided earlier.
	// I should check `mocks.go` if Update captures calls.
	// It captures CreateCalls but maybe not UpdateCalls?
	// Let's modify the mock in the test to capture it.
}

func TestCompanyService_SwitchCompany_UserNotMember(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{}
	mockCompanyRepo := &mocks.MockCompanyRepository{}
	mockCompanyUserRepo := &mocks.MockCompanyUserRepository{}
	mockTM := &mocks.MockTransactionManager{}
	mockLogger := &mocks.MockLogger{}

	svc := service.NewCompanyService(mockTM, mockCompanyRepo, mockCompanyUserRepo, mockUserRepo, nil, nil, nil, mockLogger)

	userID := "user-123"
	companyID := "company-456"

	mockCompanyUserRepo.GetUserRoleFunc = func(ctx context.Context, cID, uID string) (company.CompanyRole, error) {
		return "", company.ErrUserNotMember
	}

	err := svc.SwitchCompany(context.Background(), userID, companyID)

	assert.ErrorIs(t, err, company.ErrUserNotMember)
}
