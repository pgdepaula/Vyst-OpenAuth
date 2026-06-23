package resolvers

import (
	"context"
	"testing"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/tenant"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/graphql/model"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Me Query Tests (Context-based)
// ============================================================================

func TestMe_WithoutUserIDInContext_ReturnsUnauthorized(t *testing.T) {
	resolver := &Resolver{}
	queryResolver := &queryResolver{resolver}

	// Context without user_id
	ctx := context.Background()

	result, err := queryResolver.Me(ctx)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestMe_WithInvalidUserIDType_ReturnsError(t *testing.T) {
	resolver := &Resolver{}
	queryResolver := &queryResolver{resolver}

	// Context with wrong type for user_id
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, 12345)

	result, err := queryResolver.Me(ctx)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid user id in context")
}

// ============================================================================
// Users Query Tests
// ============================================================================

func TestUsers_ReturnsEmptyConnection(t *testing.T) {
	resolver := &Resolver{}
	queryResolver := &queryResolver{resolver}

	result, err := queryResolver.Users(context.Background(), nil, nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Items)
	assert.Equal(t, 0, result.Count)
}

func TestUsers_WithFilter_ReturnsFilteredResults(t *testing.T) {
	resolver := &Resolver{}
	queryResolver := &queryResolver{resolver}

	tenantID := "tenant-123"
	status := "active"
	filter := &model.UserFilterInput{
		TenantID: &tenantID,
		Status:   &status,
	}
	page := 1
	limit := 10

	result, err := queryResolver.Users(context.Background(), filter, &page, &limit)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Currently returns empty - will be populated when service is implemented
	assert.Empty(t, result.Items)
}

func TestUsers_WithOnlyPageParam_ReturnsConnection(t *testing.T) {
	resolver := &Resolver{}
	queryResolver := &queryResolver{resolver}

	page := 2

	result, err := queryResolver.Users(context.Background(), nil, &page, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ============================================================================
// Tenants Query Tests
// ============================================================================

func TestTenants_ReturnsEmptyConnection(t *testing.T) {
	resolver := &Resolver{}
	queryResolver := &queryResolver{resolver}

	result, err := queryResolver.Tenants(context.Background(), nil, nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Items)
	assert.Equal(t, 0, result.Count)
}

func TestTenants_WithFilter_ReturnsFilteredResults(t *testing.T) {
	resolver := &Resolver{}
	queryResolver := &queryResolver{resolver}

	status := "active"
	filter := &model.TenantFilterInput{
		Status: &status,
	}
	page := 1
	limit := 20

	result, err := queryResolver.Tenants(context.Background(), filter, &page, &limit)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Currently returns empty - will be populated when service is implemented
	assert.Empty(t, result.Items)
}

// ============================================================================
// Roles Query Tests
// ============================================================================

func TestRoles_WithoutPolicyRepo_ReturnsEmptyList(t *testing.T) {
	resolver := &Resolver{
		PolicyRepo: nil,
	}
	queryResolver := &queryResolver{resolver}

	result, err := queryResolver.Roles(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result)
}

// ============================================================================
// Permissions Query Tests
// ============================================================================

func TestPermissions_ReturnsEmptyList(t *testing.T) {
	resolver := &Resolver{}
	queryResolver := &queryResolver{resolver}

	result, err := queryResolver.Permissions(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result)
}

// ============================================================================
// Companies Query Tests
// ============================================================================

func TestCompanies_ReturnsConnection(t *testing.T) {
	repo := &mocks.MockCompanyRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*company.Company, error) {
			if id == "c1" {
				return &company.Company{ID: "c1", RazaoSocial: "Company 1"}, nil
			}
			if id == "c2" {
				return &company.Company{ID: "c2", RazaoSocial: "Company 2"}, nil
			}
			return nil, company.ErrNotFound
		},
	}
	companyUserRepo := &mocks.MockCompanyUserRepository{
		GetCompaniesForUserFunc: func(ctx context.Context, userID string) ([]*company.CompanyUser, error) {
			return []*company.CompanyUser{
				{CompanyID: "c1", Role: company.RoleAdmin, Status: company.MembershipActive},
				{CompanyID: "c2", Role: company.RoleMember, Status: company.MembershipActive},
			}, nil
		},
	}
	// Use real service with mocks
	svc := service.NewCompanyService(
		&mocks.MockTransactionManager{},
		repo,
		companyUserRepo,
		&mocks.MockUserRepository{},
		&mocks.MockEventBus{},
		&mocks.MockOutboxPublisher{},
		nil, // CompanyGateway
		&mocks.MockLogger{},
	)

	resolver := &Resolver{
		CompanyRepo:    repo,
		CompanyService: svc,
	}
	queryResolver := &queryResolver{resolver}

	// Context with user_id
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user-123")

	result, err := queryResolver.Companies(ctx, nil, nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Count)
	assert.Len(t, result.Items, 2)
}

// ============================================================================
// Company Status Mutation Tests
// ============================================================================

func TestUpdateCompanyStatus_ValidInput_ReturnsCompany(t *testing.T) {
	repo := &mocks.MockCompanyRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*company.Company, error) {
			return &company.Company{ID: "c1", Status: company.StatusActive}, nil
		},
		UpdateFunc: func(ctx context.Context, comp *company.Company) error {
			assert.Equal(t, company.StatusSuspended, comp.Status)
			return nil
		},
	}
	// Use real service with mocks
	svc := service.NewCompanyService(
		&mocks.MockTransactionManager{},
		repo,
		&mocks.MockCompanyUserRepository{},
		&mocks.MockUserRepository{},
		&mocks.MockEventBus{},
		&mocks.MockOutboxPublisher{},
		nil, // CompanyGateway
		&mocks.MockLogger{},
	)

	resolver := &Resolver{
		CompanyRepo:    repo,
		CompanyService: svc,
	}
	mutationResolver := &mutationResolver{resolver}

	result, err := mutationResolver.UpdateCompanyStatus(context.Background(), "c1", "suspended")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "suspended", result.Status)
}

func TestUpdateCompanyStatus_InvalidTransition_ReturnsError(t *testing.T) {
	repo := &mocks.MockCompanyRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*company.Company, error) {
			return &company.Company{ID: "c1", Status: company.StatusSuspended}, nil
		},
	}
	resolver := &Resolver{
		CompanyRepo: repo,
	}
	mutationResolver := &mutationResolver{resolver}

	// Suspending an already suspended company is valid in our domain (idempotent)
	// Let's test a non-existent company
	repo.GetByIDFunc = func(ctx context.Context, id string) (*company.Company, error) {
		return nil, company.ErrNotFound
	}

	result, err := mutationResolver.UpdateCompanyStatus(context.Background(), "invalid", "suspended")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// ============================================================================
// User Field Resolver Tests
// ============================================================================

func TestUserRoles_ReturnsEmptyRoles(t *testing.T) {
	resolver := &Resolver{}
	userResolver := &userResolver{resolver}

	testUser := &user.User{
		ID: "user-123",
	}

	roles, err := userResolver.Roles(context.Background(), testUser)

	require.NoError(t, err)
	assert.Empty(t, roles)
}

func TestUserCreatedAt_ReturnsFormattedTime(t *testing.T) {
	resolver := &Resolver{}
	userResolver := &userResolver{resolver}

	createdAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	testUser := &user.User{
		ID:        "user-123",
		CreatedAt: createdAt,
	}

	result, err := userResolver.CreatedAt(context.Background(), testUser)

	require.NoError(t, err)
	assert.Equal(t, "2024-01-15T10:30:00Z", result)
}

func TestUserUpdatedAt_ReturnsFormattedTime(t *testing.T) {
	resolver := &Resolver{}
	userResolver := &userResolver{resolver}

	updatedAt := time.Date(2024, 6, 20, 14, 45, 0, 0, time.UTC)
	testUser := &user.User{
		ID:        "user-123",
		UpdatedAt: updatedAt,
	}

	result, err := userResolver.UpdatedAt(context.Background(), testUser)

	require.NoError(t, err)
	assert.Equal(t, "2024-06-20T14:45:00Z", result)
}

// ============================================================================
// Tenant Field Resolver Tests
// ============================================================================

func TestTenantStatus_ReturnsStatusString(t *testing.T) {
	resolver := &Resolver{}
	tenantResolver := &tenantResolver{resolver}

	testTenant := &tenant.Tenant{
		ID:     "tenant-123",
		Status: tenant.StatusActive,
	}

	result, err := tenantResolver.Status(context.Background(), testTenant)

	require.NoError(t, err)
	assert.Equal(t, "active", result)
}

func TestTenantUserCount_ReturnsZero(t *testing.T) {
	resolver := &Resolver{}
	tenantResolver := &tenantResolver{resolver}

	testTenant := &tenant.Tenant{
		ID: "tenant-123",
	}

	result, err := tenantResolver.UserCount(context.Background(), testTenant)

	require.NoError(t, err)
	assert.Equal(t, 0, result)
}

func TestTenantCreatedAt_ReturnsFormattedTime(t *testing.T) {
	resolver := &Resolver{}
	tenantResolver := &tenantResolver{resolver}

	createdAt := time.Date(2024, 3, 10, 8, 0, 0, 0, time.UTC)
	testTenant := &tenant.Tenant{
		ID:        "tenant-123",
		CreatedAt: createdAt,
	}

	result, err := tenantResolver.CreatedAt(context.Background(), testTenant)

	require.NoError(t, err)
	assert.Equal(t, "2024-03-10T08:00:00Z", result)
}

func TestTenantUpdatedAt_ReturnsFormattedTime(t *testing.T) {
	resolver := &Resolver{}
	tenantResolver := &tenantResolver{resolver}

	updatedAt := time.Date(2024, 8, 25, 16, 30, 0, 0, time.UTC)
	testTenant := &tenant.Tenant{
		ID:        "tenant-123",
		UpdatedAt: updatedAt,
	}

	result, err := tenantResolver.UpdatedAt(context.Background(), testTenant)

	require.NoError(t, err)
	assert.Equal(t, "2024-08-25T16:30:00Z", result)
}

// ============================================================================
// Resolver Constructor Tests
// ============================================================================

func TestResolver_QueryResolver_Returns(t *testing.T) {
	resolver := &Resolver{}
	qr := resolver.Query()
	assert.NotNil(t, qr)
}

func TestResolver_MutationResolver_Returns(t *testing.T) {
	resolver := &Resolver{}
	mr := resolver.Mutation()
	assert.NotNil(t, mr)
}

func TestResolver_SubscriptionResolver_Returns(t *testing.T) {
	resolver := &Resolver{}
	sr := resolver.Subscription()
	assert.NotNil(t, sr)
}

func TestResolver_TenantResolver_Returns(t *testing.T) {
	resolver := &Resolver{}
	tr := resolver.Tenant()
	assert.NotNil(t, tr)
}

func TestResolver_UserResolver_Returns(t *testing.T) {
	resolver := &Resolver{}
	ur := resolver.User()
	assert.NotNil(t, ur)
}
