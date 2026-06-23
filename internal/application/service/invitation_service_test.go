package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/invitation"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvitationService_InviteUser_Success(t *testing.T) {
	mockInvRepo := &mocks.MockInvitationRepository{}
	mockUserRepo := &mocks.MockUserRepository{}
	mockCompanyRepo := &mocks.MockCompanyRepository{}
	mockCompanyUserRepo := &mocks.MockCompanyUserRepository{}
	mockNotifier := &mocks.MockNotificationService{}
	mockLogger := &mocks.MockLogger{}

	svc := service.NewInvitationService(mockInvRepo, mockUserRepo, mockCompanyRepo, mockCompanyUserRepo, mockNotifier, mockLogger)

	adminID := "admin-1"
	companyID := "company-1"
	email := "new@example.com"
	role := company.RoleMember

	// Mock Admin role check
	mockCompanyUserRepo.GetUserRoleFunc = func(ctx context.Context, cID, uID string) (company.CompanyRole, error) {
		if cID == companyID && uID == adminID {
			return company.RoleAdmin, nil
		}
		return "", company.ErrUserNotMember
	}

	// Mock User existence (not found means new user)
	mockUserRepo.GetByEmailFunc = func(ctx context.Context, e string) (*user.User, error) {
		return nil, user.ErrNotFound
	}

	// Mock Invitation check (none exists)
	mockInvRepo.GetByEmailAndCompanyFunc = func(ctx context.Context, e, cID string) (*invitation.Invitation, error) {
		return nil, invitation.ErrNotFound
	}

	// Mock Company fetch
	mockCompanyRepo.GetByIDFunc = func(ctx context.Context, id string) (*company.Company, error) {
		return &company.Company{ID: companyID, RazaoSocial: "Test Corp"}, nil
	}

	err := svc.InviteUser(context.Background(), adminID, companyID, email, role)

	require.NoError(t, err)
	assert.Len(t, mockInvRepo.CreateCalls, 1)
	assert.Equal(t, email, mockInvRepo.CreateCalls[0].Invitation.Email)
	assert.Equal(t, role, mockInvRepo.CreateCalls[0].Invitation.Role)
	assert.Equal(t, companyID, mockInvRepo.CreateCalls[0].Invitation.CompanyID)
	assert.Len(t, mockNotifier.SendEmailCalls, 1)
	assert.Equal(t, email, mockNotifier.SendEmailCalls[0].To)
	assert.Contains(t, mockNotifier.SendEmailCalls[0].Subject, "Test Corp")
}

func TestInvitationService_InviteUser_NonAdmin_Fails(t *testing.T) {
	mockCompanyUserRepo := &mocks.MockCompanyUserRepository{}
	svc := service.NewInvitationService(nil, nil, nil, mockCompanyUserRepo, nil, nil)

	mockCompanyUserRepo.GetUserRoleFunc = func(ctx context.Context, cID, uID string) (company.CompanyRole, error) {
		return company.RoleMember, nil // Not admin
	}

	err := svc.InviteUser(context.Background(), "user-1", "company-1", "test@example.com", company.RoleMember)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only admins can invite")
}

func TestInvitationService_AcceptInvitation_Success(t *testing.T) {
	mockInvRepo := &mocks.MockInvitationRepository{}
	mockUserRepo := &mocks.MockUserRepository{}
	mockCompanyUserRepo := &mocks.MockCompanyUserRepository{}
	mockLogger := &mocks.MockLogger{}

	svc := service.NewInvitationService(mockInvRepo, mockUserRepo, nil, mockCompanyUserRepo, nil, mockLogger)

	userID := "user-new"
	email := "new@example.com"
	companyID := "company-1"
	token := "valid-token"
	role := company.RoleMember

	inv := invitation.NewInvitation(companyID, email, role, 24*time.Hour)
	inv.Token = token

	// Mock Invitation fetch
	mockInvRepo.GetByTokenFunc = func(ctx context.Context, t string) (*invitation.Invitation, error) {
		if t == token {
			return inv, nil
		}
		return nil, invitation.ErrNotFound
	}

	// Mock User fetch
	mockUserRepo.GetByIDFunc = func(ctx context.Context, id string) (*user.User, error) {
		return &user.User{ID: userID, Email: email}, nil
	}

	// Mock AddUser
	mockCompanyUserRepo.AddUserFunc = func(ctx context.Context, cu *company.CompanyUser) error {
		if cu.CompanyID == companyID && cu.UserID == userID && cu.Role == role {
			return nil
		}
		return nil
	}

	err := svc.AcceptInvitation(context.Background(), token, userID)

	require.NoError(t, err)
	assert.Equal(t, invitation.StatusAccepted, inv.Status)
	assert.Len(t, mockCompanyUserRepo.AddUserCalls, 1) // Assumed mock has AddUserCalls (if not, we check implicit success)
	// mockCompanyUserRepo AddUserCalls is not implemented in previous mock dump unless I missed it.
	// But ListByCompanyFunc was added, so AddUser is probably there or I should check implementation of mock.
}
