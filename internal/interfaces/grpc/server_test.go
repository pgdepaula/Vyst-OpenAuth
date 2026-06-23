package grpc

import (
	"context"
	"testing"

	pb "github.com/pgdepaula/vyst-openauth/api/proto"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ValidateToken Tests
// ============================================================================

func TestValidateToken_ValidToken_ReturnsUserInfo(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{
		ValidateTokenFunc: func(token string) (*ports.Claims, error) {
			return &ports.Claims{
				UserID:   "user-123",
				TenantID: "tenant-456",
				Roles:    []string{"admin", "user"},
			}, nil
		},
	}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.ValidateToken(context.Background(), &pb.ValidateTokenRequest{
		Token: "valid-token",
	})

	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "user-123", resp.UserId)
	assert.Equal(t, "tenant-456", resp.TenantId)
	assert.Equal(t, []string{"admin", "user"}, resp.Roles)
}

func TestValidateToken_InvalidToken_ReturnsFalse(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{
		ValidateTokenFunc: func(token string) (*ports.Claims, error) {
			return nil, assert.AnError
		},
	}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.ValidateToken(context.Background(), &pb.ValidateTokenRequest{
		Token: "invalid-token",
	})

	require.NoError(t, err) // Method doesn't return error, just sets Valid=false
	assert.False(t, resp.Valid)
}

func TestValidateToken_EmptyToken_ReturnsFalse(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.ValidateToken(context.Background(), &pb.ValidateTokenRequest{
		Token: "",
	})

	require.NoError(t, err)
	assert.False(t, resp.Valid)
}

// ============================================================================
// GetUserRoles Tests
// ============================================================================

func TestGetUserRoles_ValidUser_ReturnsRoles(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{
		GetRolesForUserFunc: func(ctx context.Context, userID string) ([]string, error) {
			assert.Equal(t, "user-123", userID)
			return []string{"admin", "manager", "user"}, nil
		},
	}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.GetUserRoles(context.Background(), &pb.GetUserRolesRequest{
		UserId:   "user-123",
		TenantId: "tenant-456",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"admin", "manager", "user"}, resp.Roles)
}

func TestGetUserRoles_MissingUserID_ReturnsError(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.GetUserRoles(context.Background(), &pb.GetUserRolesRequest{
		UserId:   "",
		TenantId: "tenant-456",
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "user_id is required")
}

func TestGetUserRoles_UserWithNoRoles_ReturnsEmptyList(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{
		GetRolesForUserFunc: func(ctx context.Context, userID string) ([]string, error) {
			return []string{}, nil
		},
	}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.GetUserRoles(context.Background(), &pb.GetUserRolesRequest{
		UserId:   "new-user",
		TenantId: "tenant-456",
	})

	require.NoError(t, err)
	assert.Empty(t, resp.Roles)
}

// ============================================================================
// GetUserPermissions Tests
// ============================================================================

func TestGetUserPermissions_ValidUser_ReturnsPermissions(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{
		GetRolesForUserFunc: func(ctx context.Context, userID string) ([]string, error) {
			return []string{"admin", "viewer"}, nil
		},
	}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.GetUserPermissions(context.Background(), &pb.GetUserPermissionsRequest{
		UserId:   "user-123",
		TenantId: "tenant-456",
	})

	require.NoError(t, err)
	assert.Len(t, resp.Permissions, 2)
	// Current implementation maps roles to permissions
	assert.Equal(t, "admin", resp.Permissions[0].Action)
	assert.Equal(t, "viewer", resp.Permissions[1].Action)
}

func TestGetUserPermissions_MissingUserID_ReturnsError(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.GetUserPermissions(context.Background(), &pb.GetUserPermissionsRequest{
		UserId:   "",
		TenantId: "tenant-456",
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "user_id is required")
}

// ============================================================================
// RevokeUserSessions Tests
// ============================================================================

func TestRevokeUserSessions_ValidUser_ReturnsSuccess(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.RevokeUserSessions(context.Background(), &pb.RevokeUserSessionsRequest{
		UserId: "user-123",
		Reason: "Security incident",
	})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	// No active streams, so revoked count is 0
	assert.Equal(t, int32(0), resp.RevokedCount)
}

func TestRevokeUserSessions_MissingUserID_ReturnsError(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.RevokeUserSessions(context.Background(), &pb.RevokeUserSessionsRequest{
		UserId: "",
		Reason: "Test",
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "user_id is required")
}

// ============================================================================
// TriggerKillSwitch Tests
// ============================================================================

func TestTriggerKillSwitch_NoActiveStreams_ReturnsZero(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	// No streams registered, should return 0
	count := server.triggerKillSwitch("user-123")
	assert.Equal(t, 0, count)
}

// ============================================================================
// Server Constructor Tests
// ============================================================================

func TestNewServer_Initializes(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}

	companyUserRepo := &mocks.MockCompanyUserRepository{}
	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	assert.NotNil(t, server)
	assert.NotNil(t, server.streams)
}

// ============================================================================
// ValidateCompanyAccess Tests
// ============================================================================

func TestValidateCompanyAccess_ValidAccess_ReturnsRole(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{
		ValidateTokenFunc: func(token string) (*ports.Claims, error) {
			return &ports.Claims{UserID: "user-123"}, nil
		},
	}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}
	companyUserRepo := &mocks.MockCompanyUserRepository{
		GetUserRoleFunc: func(ctx context.Context, companyID, userID string) (company.CompanyRole, error) {
			return company.RoleAdmin, nil
		},
	}

	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.ValidateCompanyAccess(context.Background(), &pb.ValidateCompanyAccessRequest{
		Token:     "valid-token",
		CompanyId: "comp-123",
	})

	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "user-123", resp.UserId)
	assert.Equal(t, "admin", resp.Role)
}

func TestValidateCompanyAccess_UserNotMember_ReturnsInvalid(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{
		ValidateTokenFunc: func(token string) (*ports.Claims, error) {
			return &ports.Claims{UserID: "user-123"}, nil
		},
	}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}
	companyUserRepo := &mocks.MockCompanyUserRepository{
		GetUserRoleFunc: func(ctx context.Context, companyID, userID string) (company.CompanyRole, error) {
			return "", company.ErrUserNotMember
		},
	}

	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.ValidateCompanyAccess(context.Background(), &pb.ValidateCompanyAccessRequest{
		Token:     "valid-token",
		CompanyId: "comp-123",
	})

	require.NoError(t, err)
	assert.False(t, resp.Valid)
}

// ============================================================================
// GetCompanyRoles Tests
// ============================================================================

func TestGetCompanyRoles_ReturnsList(t *testing.T) {
	tokenSvc := &mocks.MockTokenService{}
	policyRepo := &mocks.MockPolicyRepository{}
	logger := &mocks.MockLogger{}
	companyUserRepo := &mocks.MockCompanyUserRepository{
		GetCompaniesForUserFunc: func(ctx context.Context, userID string) ([]*company.CompanyUser, error) {
			return []*company.CompanyUser{
				{CompanyID: "c1", Role: company.RoleAdmin},
				{CompanyID: "c2", Role: company.RoleMember},
			}, nil
		},
	}

	server := NewServer(tokenSvc, policyRepo, companyUserRepo, logger)

	resp, err := server.GetCompanyRoles(context.Background(), &pb.GetCompanyRolesRequest{
		UserId: "user-123",
	})

	require.NoError(t, err)
	assert.Len(t, resp.Roles, 2)
	assert.Equal(t, "c1", resp.Roles[0].CompanyId)
	assert.Equal(t, "admin", resp.Roles[0].Role)
}
