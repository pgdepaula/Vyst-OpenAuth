package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"github.com/pgdepaula/vyst-openauth/internal/domain/tenant"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// RegisterWithTenant Success Tests
// ============================================================================

func TestRegistrationService_RegisterWithTenant_Success(t *testing.T) {
	mockTM := &mocks.MockTransactionManager{}
	mockUserRepo := &mocks.MockUserRepository{}
	mockTenantRepo := &mocks.MockTenantRepository{}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{
		HashFunc: func(password string) (string, error) {
			return "hashed_" + password, nil
		},
	}
	mockOutbox := &mocks.MockOutboxPublisher{}
	mockEventBus := &mocks.MockEventBus{}
	mockNotifier := &mocks.MockNotificationService{}

	svc := service.NewRegistrationService(
		mockTM,
		mockUserRepo,
		mockTenantRepo,
		mockPolicyRepo,
		mockHasher,
		mockOutbox,
		mockEventBus,
		mockNotifier,
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "securepassword",
		TenantName: "Test Company",
	}

	result, err := svc.RegisterWithTenant(context.Background(), cmd)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test@example.com", result.User.Email)
	assert.Equal(t, "Test Company", result.Tenant.Name)
	assert.Equal(t, tenant.StatusActive, result.Tenant.Status)
	assert.NotEmpty(t, result.User.ID)
	assert.NotEmpty(t, result.Tenant.ID)
}

func TestRegistrationService_RegisterWithTenant_HashesPassword(t *testing.T) {
	var hashedPassword string
	mockHasher := &mocks.MockPasswordHasher{
		HashFunc: func(password string) (string, error) {
			hashedPassword = "bcrypt_hash_of_" + password
			return hashedPassword, nil
		},
	}

	svc := service.NewRegistrationService(
		&mocks.MockTransactionManager{},
		&mocks.MockUserRepository{},
		&mocks.MockTenantRepository{},
		&mocks.MockPolicyRepository{},
		mockHasher,
		&mocks.MockOutboxPublisher{},
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "mypassword",
		TenantName: "Test",
	}

	result, err := svc.RegisterWithTenant(context.Background(), cmd)

	require.NoError(t, err)
	assert.Len(t, mockHasher.HashCalls, 1)
	assert.Equal(t, "mypassword", mockHasher.HashCalls[0])
	assert.Equal(t, hashedPassword, result.User.PasswordHash)
}

func TestRegistrationService_RegisterWithTenant_CreatesTenant(t *testing.T) {
	mockTenantRepo := &mocks.MockTenantRepository{}

	svc := service.NewRegistrationService(
		&mocks.MockTransactionManager{},
		&mocks.MockUserRepository{},
		mockTenantRepo,
		&mocks.MockPolicyRepository{},
		&mocks.MockPasswordHasher{},
		&mocks.MockOutboxPublisher{},
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "password",
		TenantName: "Acme Corp",
	}

	_, err := svc.RegisterWithTenant(context.Background(), cmd)

	require.NoError(t, err)
	assert.Len(t, mockTenantRepo.CreateCalls, 1)
	assert.Equal(t, "Acme Corp", mockTenantRepo.CreateCalls[0].Name)
	assert.Equal(t, tenant.StatusActive, mockTenantRepo.CreateCalls[0].Status)
}

func TestRegistrationService_RegisterWithTenant_CreatesUser(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{}

	svc := service.NewRegistrationService(
		&mocks.MockTransactionManager{},
		mockUserRepo,
		&mocks.MockTenantRepository{},
		&mocks.MockPolicyRepository{},
		&mocks.MockPasswordHasher{},
		&mocks.MockOutboxPublisher{},
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "user@test.com",
		Password:   "password",
		TenantName: "Test",
	}

	_, err := svc.RegisterWithTenant(context.Background(), cmd)

	require.NoError(t, err)
	assert.Len(t, mockUserRepo.CreateCalls, 1)
	assert.Equal(t, "user@test.com", mockUserRepo.CreateCalls[0].User.Email)
}

func TestRegistrationService_RegisterWithTenant_CreatesReBACTuple(t *testing.T) {
	mockPolicyRepo := &mocks.MockPolicyRepository{}

	svc := service.NewRegistrationService(
		&mocks.MockTransactionManager{},
		&mocks.MockUserRepository{},
		&mocks.MockTenantRepository{},
		mockPolicyRepo,
		&mocks.MockPasswordHasher{},
		&mocks.MockOutboxPublisher{},
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "password",
		TenantName: "Test",
	}

	result, err := svc.RegisterWithTenant(context.Background(), cmd)

	require.NoError(t, err)
	assert.Len(t, mockPolicyRepo.WriteTupleCalls, 1)

	tuple := mockPolicyRepo.WriteTupleCalls[0]
	assert.Equal(t, "user:"+result.User.ID, tuple.Subject)
	assert.Equal(t, "owner", tuple.Relation)
	assert.Equal(t, "tenant:"+result.Tenant.ID, tuple.Object)
	assert.Equal(t, result.Tenant.ID, tuple.TenantID)
}

func TestRegistrationService_RegisterWithTenant_PublishesOutboxEvents(t *testing.T) {
	mockOutbox := &mocks.MockOutboxPublisher{}

	svc := service.NewRegistrationService(
		&mocks.MockTransactionManager{},
		&mocks.MockUserRepository{},
		&mocks.MockTenantRepository{},
		&mocks.MockPolicyRepository{},
		&mocks.MockPasswordHasher{},
		mockOutbox,
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "password",
		TenantName: "Test",
	}

	_, err := svc.RegisterWithTenant(context.Background(), cmd)

	require.NoError(t, err)
	assert.Len(t, mockOutbox.PublishCalls, 2, "Should publish TenantProvisioned and UserCreated events")

	assert.Equal(t, event.TenantProvisioned, mockOutbox.PublishCalls[0].Type)
	assert.Equal(t, event.UserCreated, mockOutbox.PublishCalls[1].Type)
}

func TestRegistrationService_RegisterWithTenant_SetsRLSContext(t *testing.T) {
	mockTenantRepo := &mocks.MockTenantRepository{}

	svc := service.NewRegistrationService(
		&mocks.MockTransactionManager{},
		&mocks.MockUserRepository{},
		mockTenantRepo,
		&mocks.MockPolicyRepository{},
		&mocks.MockPasswordHasher{},
		&mocks.MockOutboxPublisher{},
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "password",
		TenantName: "Test",
	}

	result, err := svc.RegisterWithTenant(context.Background(), cmd)

	require.NoError(t, err)
	assert.Len(t, mockTenantRepo.SetCurrentTenantCalls, 1)
	assert.Equal(t, result.Tenant.ID, mockTenantRepo.SetCurrentTenantCalls[0])
}

// ============================================================================
// RegisterWithTenant Error Cases
// ============================================================================

func TestRegistrationService_RegisterWithTenant_HashError_ReturnsError(t *testing.T) {
	mockHasher := &mocks.MockPasswordHasher{
		HashFunc: func(password string) (string, error) {
			return "", errors.New("hashing failed")
		},
	}

	svc := service.NewRegistrationService(
		&mocks.MockTransactionManager{},
		&mocks.MockUserRepository{},
		&mocks.MockTenantRepository{},
		&mocks.MockPolicyRepository{},
		mockHasher,
		&mocks.MockOutboxPublisher{},
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "password",
		TenantName: "Test",
	}

	result, err := svc.RegisterWithTenant(context.Background(), cmd)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hash")
}

func TestRegistrationService_RegisterWithTenant_TenantCreationError_RollsBack(t *testing.T) {
	mockTenantRepo := &mocks.MockTenantRepository{
		CreateFunc: func(ctx context.Context, t *tenant.Tenant) error {
			return errors.New("tenant creation failed")
		},
	}

	svc := service.NewRegistrationService(
		&mocks.MockTransactionManager{},
		&mocks.MockUserRepository{},
		mockTenantRepo,
		&mocks.MockPolicyRepository{},
		&mocks.MockPasswordHasher{},
		&mocks.MockOutboxPublisher{},
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "password",
		TenantName: "Test",
	}

	result, err := svc.RegisterWithTenant(context.Background(), cmd)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant")
}

func TestRegistrationService_RegisterWithTenant_PolicyWriteError_RollsBack(t *testing.T) {
	mockPolicyRepo := &mocks.MockPolicyRepository{
		WriteTupleFunc: func(ctx context.Context, tuple policy.Tuple) error {
			return errors.New("policy write failed")
		},
	}

	svc := service.NewRegistrationService(
		&mocks.MockTransactionManager{},
		&mocks.MockUserRepository{},
		&mocks.MockTenantRepository{},
		mockPolicyRepo,
		&mocks.MockPasswordHasher{},
		&mocks.MockOutboxPublisher{},
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "password",
		TenantName: "Test",
	}

	result, err := svc.RegisterWithTenant(context.Background(), cmd)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission")
}

// ============================================================================
// Transaction Manager Tests
// ============================================================================

func TestRegistrationService_RegisterWithTenant_UsesTransactionManager(t *testing.T) {
	mockTM := &mocks.MockTransactionManager{}

	svc := service.NewRegistrationService(
		mockTM,
		&mocks.MockUserRepository{},
		&mocks.MockTenantRepository{},
		&mocks.MockPolicyRepository{},
		&mocks.MockPasswordHasher{},
		&mocks.MockOutboxPublisher{},
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "password",
		TenantName: "Test",
	}

	_, err := svc.RegisterWithTenant(context.Background(), cmd)

	require.NoError(t, err)
	assert.Equal(t, 1, mockTM.RunInTransactionCalls, "Should run in transaction exactly once")
}

func TestRegistrationService_RegisterWithTenant_TransactionError_ReturnsError(t *testing.T) {
	mockTM := &mocks.MockTransactionManager{
		RunInTransactionFunc: func(ctx context.Context, fn func(ctx context.Context) error) error {
			_ = fn(ctx)
			return errors.New("transaction commit failed")
		},
	}

	svc := service.NewRegistrationService(
		mockTM,
		&mocks.MockUserRepository{},
		&mocks.MockTenantRepository{},
		&mocks.MockPolicyRepository{},
		&mocks.MockPasswordHasher{},
		&mocks.MockOutboxPublisher{},
		&mocks.MockEventBus{},
		&mocks.MockNotificationService{},
		service.NewDocumentService(&mocks.MockLogger{}, nil, nil),
	)

	cmd := service.RegisterCommand{
		Email:      "test@example.com",
		Password:   "password",
		TenantName: "Test",
	}

	result, err := svc.RegisterWithTenant(context.Background(), cmd)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction")
}
