package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/tenant"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/handlers"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Test Helpers
// ============================================================================

// MockLogger implements ports.Logger for testing
type MockLogger struct{}

func (l *MockLogger) Debug(msg string, args ...any)                {}
func (l *MockLogger) Info(msg string, args ...any)                 {}
func (l *MockLogger) Warn(msg string, args ...any)                 {}
func (l *MockLogger) Error(msg string, args ...any)                {}
func (l *MockLogger) With(args ...any) ports.Logger                { return l }
func (l *MockLogger) WithContext(ctx context.Context) ports.Logger { return l }

func newTestAuthHandler() (*handlers.AuthHandler, *mocks.MockUserRepository, *mocks.MockTokenService, *mocks.MockPasswordHasher) {
	mockUserRepo := &mocks.MockUserRepository{}
	mockHasher := &mocks.MockPasswordHasher{}
	mockTokenSvc := &mocks.MockTokenService{}
	mockTM := &mocks.MockTransactionManager{}
	mockTenantRepo := &mocks.MockTenantRepository{}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockOutbox := &mocks.MockOutboxPublisher{}
	mockEventBus := &mocks.MockEventBus{}
	mockNotifier := &mocks.MockNotificationService{}
	mockParam := &mocks.MockCompanyUserRepository{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockLogger := &MockLogger{}

	registrationSvc := service.NewRegistrationService(
		mockTM,
		mockUserRepo,
		mockTenantRepo,
		mockPolicyRepo,
		mockHasher,
		mockOutbox,
		mockEventBus,
		mockNotifier,
		service.NewDocumentService(mockLogger, nil, nil),
	)
	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockParam, mockSessionRepo, mockHasher, mockTokenSvc, mockNotifier, mockLogger)

	handler := handlers.NewAuthHandler(registrationSvc, authSvc, nil, nil, nil)

	return handler, mockUserRepo, mockTokenSvc, mockHasher
}

func jsonPostRequest(path string, body interface{}) *http.Request {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// ============================================================================
// Register Handler Tests
// ============================================================================

func TestAuthHandler_Register_ValidRequest_Returns201(t *testing.T) {
	handler, _, _, mockHasher := newTestAuthHandler()
	mockHasher.HashFunc = func(password string) (string, error) {
		return "hashed_password", nil
	}

	body := map[string]string{
		"email":       "test@example.com",
		"password":    "securepassword",
		"tenant_name": "Acme Corp",
	}
	req := jsonPostRequest("/auth/register", body)
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "User registered successfully", response["message"])
	assert.NotEmpty(t, response["user_id"])
	assert.NotEmpty(t, response["tenant_id"])
}

func TestAuthHandler_Register_MissingEmail_Returns400(t *testing.T) {
	handler, _, _, _ := newTestAuthHandler()

	body := map[string]string{
		"password":    "securepassword",
		"tenant_name": "Acme Corp",
	}
	req := jsonPostRequest("/auth/register", body)
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Contains(t, response["error"], "required")
}

func TestAuthHandler_Register_MissingPassword_Returns400(t *testing.T) {
	handler, _, _, _ := newTestAuthHandler()

	body := map[string]string{
		"email":       "test@example.com",
		"tenant_name": "Acme Corp",
	}
	req := jsonPostRequest("/auth/register", body)
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Register_MissingTenantName_Returns400(t *testing.T) {
	handler, _, _, _ := newTestAuthHandler()

	body := map[string]string{
		"email":    "test@example.com",
		"password": "securepassword",
	}
	req := jsonPostRequest("/auth/register", body)
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Register_InvalidJSON_Returns400(t *testing.T) {
	handler, _, _, _ := newTestAuthHandler()

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// Login Handler Tests
// ============================================================================

func TestAuthHandler_Login_ValidCredentials_Returns200(t *testing.T) {
	handler, mockUserRepo, mockToken, mockHasher := newTestAuthHandler()

	mockUserRepo.GetByEmailFunc = func(ctx context.Context, email string) (*user.User, error) {
		u := mocks.NewTestUser("user-123", email, "tenant-456")
		u.Status = user.StatusActive
		return u, nil
	}
	mockHasher.VerifyFunc = func(password, hash string) bool {
		return true
	}
	mockToken.GenerateTokenFunc = func(userID, tenantID string, roles []string, activeCompanyID, companyRole, identityType string) (string, error) {
		return "jwt.token.here", nil
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": "correctpassword",
	}
	req := jsonPostRequest("/auth/login", body)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "jwt.token.here", response["token"])
	assert.Equal(t, float64(86400), response["expires_in"])
}

func TestAuthHandler_Login_InvalidCredentials_Returns401(t *testing.T) {
	handler, mockUserRepo, _, mockHasher := newTestAuthHandler()

	mockUserRepo.GetByEmailFunc = func(ctx context.Context, email string) (*user.User, error) {
		return mocks.NewTestUser("user-123", email, "tenant-456"), nil
	}
	mockHasher.VerifyFunc = func(password, hash string) bool {
		return false
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": "wrongpassword",
	}
	req := jsonPostRequest("/auth/login", body)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Contains(t, response["error"], "Invalid credentials")
}

func TestAuthHandler_Login_MissingEmail_Returns400(t *testing.T) {
	handler, _, _, _ := newTestAuthHandler()

	body := map[string]string{
		"password": "password",
	}
	req := jsonPostRequest("/auth/login", body)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Login_MissingPassword_Returns400(t *testing.T) {
	handler, _, _, _ := newTestAuthHandler()

	body := map[string]string{
		"email": "test@example.com",
	}
	req := jsonPostRequest("/auth/login", body)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Login_InvalidJSON_Returns400(t *testing.T) {
	handler, _, _, _ := newTestAuthHandler()

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString("{invalid}"))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// Me Handler Tests
// ============================================================================

func TestAuthHandler_Me_WithUserContext_ReturnsUser(t *testing.T) {
	handler, mockUserRepo, _, _ := newTestAuthHandler()

	testUser := mocks.NewTestUser("user-123", "test@example.com", "tenant-456")
	mockUserRepo.GetByIDFunc = func(ctx context.Context, id string) (*user.User, error) {
		return testUser, nil
	}

	req := httptest.NewRequest("GET", "/auth/me", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-123")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "test@example.com", response["email"])
}

func TestAuthHandler_Me_NoUserContext_Returns401(t *testing.T) {
	handler, _, _, _ := newTestAuthHandler()

	req := httptest.NewRequest("GET", "/auth/me", nil)
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestAuthHandler_Register_SetsContentTypeJSON(t *testing.T) {
	handler, _, _, mockHasher := newTestAuthHandler()
	mockHasher.HashFunc = func(password string) (string, error) {
		return "hash", nil
	}

	body := map[string]string{
		"email":       "test@example.com",
		"password":    "password",
		"tenant_name": "Test",
	}
	req := jsonPostRequest("/auth/register", body)
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestAuthHandler_Login_SetsContentTypeJSON(t *testing.T) {
	handler, mockUserRepo, mockToken, mockHasher := newTestAuthHandler()
	mockUserRepo.GetByEmailFunc = func(ctx context.Context, email string) (*user.User, error) {
		return mocks.NewTestUser("user-123", email, "tenant-456"), nil
	}
	mockHasher.VerifyFunc = func(password, hash string) bool { return true }
	mockToken.GenerateTokenFunc = func(userID, tenantID string, roles []string, activeCompanyID, companyRole, identityType string) (string, error) {
		return "token", nil
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password",
	}
	req := jsonPostRequest("/auth/login", body)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestAuthHandler_Register_EmptyBody_Returns400(t *testing.T) {
	handler, _, _, _ := newTestAuthHandler()

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Register_ReturnsCorrectUserResponseFields(t *testing.T) {
	handler, _, _, mockHasher := newTestAuthHandler()
	mockHasher.HashFunc = func(password string) (string, error) {
		return "hashed", nil
	}

	body := map[string]string{
		"email":       "dto@test.com",
		"password":    "password",
		"tenant_name": "DTOTest Corp",
	}
	req := jsonPostRequest("/auth/register", body)
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var response handlers.RegisterResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	assert.NotEmpty(t, response.UserID)
	assert.NotEmpty(t, response.TenantID)
	assert.Equal(t, "User registered successfully", response.Message)
}

func TestAuthHandler_Login_ReturnsCorrectLoginResponseFields(t *testing.T) {
	handler, mockUserRepo, mockToken, mockHasher := newTestAuthHandler()
	mockUserRepo.GetByEmailFunc = func(ctx context.Context, email string) (*user.User, error) {
		u := mocks.NewTestUser("user-123", email, "tenant-456")
		u.Status = user.StatusActive
		return u, nil
	}
	mockHasher.VerifyFunc = func(password, hash string) bool { return true }
	mockToken.GenerateTokenFunc = func(userID, tenantID string, roles []string, activeCompanyID, companyRole, identityType string) (string, error) {
		return "test.jwt.token", nil
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password",
	}
	req := jsonPostRequest("/auth/login", body)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response handlers.LoginResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	assert.Equal(t, "test.jwt.token", response.Token)
	assert.Equal(t, 86400, response.ExpiresIn)
}

// Placeholder test for unused import
var _ = tenant.StatusActive
var _ = ports.Claims{}
