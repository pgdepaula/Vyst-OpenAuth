package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLogger implements ports.Logger for testing
type MockLogger struct{}

func (l *MockLogger) Debug(msg string, args ...any)                {}
func (l *MockLogger) Info(msg string, args ...any)                 {}
func (l *MockLogger) Warn(msg string, args ...any)                 {}
func (l *MockLogger) Error(msg string, args ...any)                {}
func (l *MockLogger) With(args ...any) ports.Logger                { return l }
func (l *MockLogger) WithContext(ctx context.Context) ports.Logger { return l }

// ============================================================================
// Login Tests
// ============================================================================

func TestAuthService_Login_ValidCredentials_ReturnsTokenPair(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{
		GetByEmailFunc: func(ctx context.Context, email string) (*user.User, error) {
			u := mocks.NewTestUser("user-123", email, "tenant-456")
			u.Status = user.StatusActive
			return u, nil
		},
	}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{
		VerifyFunc: func(password, hash string) bool {
			return true
		},
	}
	mockToken := &mocks.MockTokenService{
		GenerateTokenFunc: func(userID, tenantID string, roles []string, activeCompanyID, companyRole, identityType string) (string, error) {
			return "jwt.token.here", nil
		},
	}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	tokenPair, err := authSvc.Login(context.Background(), "test@example.com", "password123")

	require.NoError(t, err)
	assert.NotNil(t, tokenPair)
	assert.Equal(t, "jwt.token.here", tokenPair.AccessToken)
	assert.Equal(t, 86400, tokenPair.ExpiresIn)
}

func TestAuthService_Login_UserNotFound_ReturnsError(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{
		GetByEmailFunc: func(ctx context.Context, email string) (*user.User, error) {
			return nil, user.ErrNotFound
		},
	}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{}
	mockToken := &mocks.MockTokenService{}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	tokenPair, err := authSvc.Login(context.Background(), "unknown@example.com", "password123")

	assert.Nil(t, tokenPair)
	assert.Error(t, err)
	assert.Equal(t, service.ErrInvalidCredentials, err)
}

func TestAuthService_Login_WrongPassword_ReturnsError(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{
		GetByEmailFunc: func(ctx context.Context, email string) (*user.User, error) {
			u := mocks.NewTestUser("user-123", email, "tenant-456")
			u.Status = user.StatusActive
			return u, nil
		},
	}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{
		VerifyFunc: func(password, hash string) bool {
			return false
		},
	}
	mockToken := &mocks.MockTokenService{}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	tokenPair, err := authSvc.Login(context.Background(), "test@example.com", "wrongpassword")

	assert.Nil(t, tokenPair)
	assert.Error(t, err)
	assert.Equal(t, service.ErrInvalidCredentials, err)
}

func TestAuthService_Login_TokenGenerationError_ReturnsError(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{
		GetByEmailFunc: func(ctx context.Context, email string) (*user.User, error) {
			u := mocks.NewTestUser("user-123", email, "tenant-456")
			u.Status = user.StatusActive
			return u, nil
		},
	}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{
		VerifyFunc: func(password, hash string) bool {
			return true
		},
	}
	mockToken := &mocks.MockTokenService{
		GenerateTokenFunc: func(userID, tenantID string, roles []string, activeCompanyID, companyRole, identityType string) (string, error) {
			return "", errors.New("token generation failed")
		},
	}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	tokenPair, err := authSvc.Login(context.Background(), "test@example.com", "password123")

	assert.Nil(t, tokenPair)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token generation failed")
}

func TestAuthService_Login_DatabaseError_ReturnsError(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{
		GetByEmailFunc: func(ctx context.Context, email string) (*user.User, error) {
			return nil, errors.New("database connection error")
		},
	}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{}
	mockToken := &mocks.MockTokenService{}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	tokenPair, err := authSvc.Login(context.Background(), "test@example.com", "password123")

	assert.Nil(t, tokenPair)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection error")
}

// ============================================================================
// ValidateToken Tests
// ============================================================================

func TestAuthService_ValidateToken_ValidToken_ReturnsClaims(t *testing.T) {
	expectedClaims := &ports.Claims{
		UserID:   "user-123",
		TenantID: "tenant-456",
		Roles:    []string{"admin"},
	}
	mockUserRepo := &mocks.MockUserRepository{}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{}
	mockToken := &mocks.MockTokenService{
		ValidateTokenFunc: func(tokenString string) (*ports.Claims, error) {
			return expectedClaims, nil
		},
	}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	claims, err := authSvc.ValidateToken("valid.token.here")

	require.NoError(t, err)
	assert.Equal(t, expectedClaims, claims)
}

func TestAuthService_ValidateToken_InvalidToken_ReturnsError(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{}
	mockToken := &mocks.MockTokenService{
		ValidateTokenFunc: func(tokenString string) (*ports.Claims, error) {
			return nil, errors.New("invalid token")
		},
	}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	claims, err := authSvc.ValidateToken("invalid.token")

	assert.Nil(t, claims)
	assert.Error(t, err)
}

// ============================================================================
// GetUser Tests
// ============================================================================

func TestAuthService_GetUser_ExistingUser_ReturnsUser(t *testing.T) {
	expectedUser := mocks.NewTestUser("user-123", "test@example.com", "tenant-456")
	mockUserRepo := &mocks.MockUserRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*user.User, error) {
			return expectedUser, nil
		},
	}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{}
	mockToken := &mocks.MockTokenService{}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	u, err := authSvc.GetUser(context.Background(), "user-123")

	require.NoError(t, err)
	assert.Equal(t, expectedUser, u)
}

func TestAuthService_GetUser_NonExistingUser_ReturnsError(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*user.User, error) {
			return nil, user.ErrNotFound
		},
	}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{}
	mockToken := &mocks.MockTokenService{}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	u, err := authSvc.GetUser(context.Background(), "non-existing-id")

	assert.Nil(t, u)
	assert.Error(t, err)
}

// ============================================================================
// Edge Cases and Behavior Tests
// ============================================================================

func TestAuthService_Login_CallsCorrectMethods(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{
		GetByEmailFunc: func(ctx context.Context, email string) (*user.User, error) {
			u := mocks.NewTestUser("user-123", email, "tenant-456")
			u.Status = user.StatusActive
			return u, nil
		},
	}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{
		VerifyFunc: func(password, hash string) bool {
			return true
		},
	}

	mockToken := &mocks.MockTokenService{}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	_, _ = authSvc.Login(context.Background(), "test@example.com", "password123")

	assert.Len(t, mockUserRepo.GetByEmailCalls, 1)
	assert.Equal(t, "test@example.com", mockUserRepo.GetByEmailCalls[0])

	assert.Len(t, mockHasher.VerifyCalls, 1)
	assert.Equal(t, "password123", mockHasher.VerifyCalls[0].Password)

	assert.Len(t, mockToken.GenerateTokenCalls, 1)
	assert.Equal(t, "user-123", mockToken.GenerateTokenCalls[0].UserID)
	assert.Equal(t, "tenant-456", mockToken.GenerateTokenCalls[0].TenantID)
}

func TestAuthService_Login_EmptyEmail_FailsAtRepository(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{
		GetByEmailFunc: func(ctx context.Context, email string) (*user.User, error) {
			return nil, user.ErrNotFound
		},
	}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{}
	mockToken := &mocks.MockTokenService{}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockToken, &mocks.MockNotificationService{}, mockLogger)

	tokenPair, err := authSvc.Login(context.Background(), "", "password123")

	assert.Nil(t, tokenPair)
	assert.Error(t, err)
	assert.Equal(t, service.ErrInvalidCredentials, err)
}
