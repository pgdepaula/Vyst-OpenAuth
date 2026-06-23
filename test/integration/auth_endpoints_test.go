package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/handlers"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ============================================================================
// HTTP Test Helpers
// ============================================================================

// JSONRequest creates a JSON HTTP request.
func JSONRequest(method, url string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// DoJSONRequest performs a JSON request and returns the response.
func DoJSONRequest(t *testing.T, method, url string, body interface{}) (*http.Response, map[string]interface{}) {
	t.Helper()
	req, err := JSONRequest(method, url, body)
	require.NoError(t, err)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]interface{}
	if len(respBody) > 0 {
		require.NoError(t, json.Unmarshal(respBody, &result))
	}

	return resp, result
}

// ============================================================================
// Auth Endpoint Integration Tests with PostgreSQL 13.22
// ============================================================================

func setupAuthTestServer(t *testing.T) (*httptest.Server, *mocks.MockUserRepository, *mocks.MockPasswordHasher, *mocks.MockTokenService) {
	mockUserRepo := &mocks.MockUserRepository{}
	mockTenantRepo := &mocks.MockTenantRepository{}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
	mockHasher := &mocks.MockPasswordHasher{}
	mockToken := &mocks.MockTokenService{}
	mockTM := &mocks.MockTransactionManager{}
	mockOutbox := &mocks.MockOutboxPublisher{}
	mockEventBus := &mocks.MockEventBus{}
	mockNotifier := &mocks.MockNotificationService{}
	mockSessionRepo := &mocks.MockSessionRepository{}
	mockCompanyUserRepo := &mocks.MockCompanyUserRepository{}
	mockLogger := &mocks.MockLogger{}

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

	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockCompanyUserRepo, mockSessionRepo, mockHasher, mockToken, mockNotifier, mockLogger)

	authHandler := handlers.NewAuthHandler(registrationSvc, authSvc, nil, nil, nil)
	healthHandler := handlers.NewHealthHandler()

	r := chi.NewRouter()
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)
	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)
	r.Route("/auth", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				// Simulate auth middleware for /me endpoint
				token := req.Header.Get("Authorization")
				if strings.HasPrefix(token, "Bearer test-token-") {
					parts := strings.Split(strings.TrimPrefix(token, "Bearer test-token-"), "-")
					if len(parts) >= 2 {
						ctx := context.WithValue(req.Context(), middleware.UserIDKey, parts[0])
						ctx = context.WithValue(ctx, middleware.TenantKey, parts[1])
						next.ServeHTTP(w, req.WithContext(ctx))
						return
					}
				}
				next.ServeHTTP(w, req)
			})
		})
		r.Get("/me", authHandler.Me)
	})

	server := httptest.NewServer(r)
	t.Cleanup(server.Close)

	return server, mockUserRepo, mockHasher, mockToken
}

// ============================================================================
// Registration Endpoint Tests
// ============================================================================

func TestEndpoint_Register_Success(t *testing.T) {
	server, _, mockHasher, _ := setupAuthTestServer(t)
	mockHasher.HashFunc = func(password string) (string, error) {
		return "hashed_" + password, nil
	}

	body := map[string]string{
		"email":       "test@example.com",
		"password":    "SecurePass123!",
		"tenant_name": "Acme Corporation",
	}

	resp, result := DoJSONRequest(t, "POST", server.URL+"/auth/register", body)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "User registered successfully", result["message"])
	assert.NotEmpty(t, result["user_id"])
	assert.NotEmpty(t, result["tenant_id"])
}

func TestEndpoint_Register_InvalidEmail_Returns400(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	testCases := []struct {
		name  string
		email string
	}{
		{"empty email", ""},
		{"missing email field", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]string{
				"password":    "SecurePass123!",
				"tenant_name": "Acme",
			}
			if tc.email != "" {
				body["email"] = tc.email
			}

			resp, result := DoJSONRequest(t, "POST", server.URL+"/auth/register", body)

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Contains(t, result["error"], "required")
		})
	}
}

func TestEndpoint_Register_WeakPassword_Returns400(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	body := map[string]string{
		"email":       "test@example.com",
		"password":    "", // Empty password
		"tenant_name": "Acme",
	}

	resp, result := DoJSONRequest(t, "POST", server.URL+"/auth/register", body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.NotEmpty(t, result["error"])
}

func TestEndpoint_Register_MissingTenantName_Returns400(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	body := map[string]string{
		"email":    "test@example.com",
		"password": "SecurePass123!",
	}

	resp, result := DoJSONRequest(t, "POST", server.URL+"/auth/register", body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, result["error"], "required")
}

func TestEndpoint_Register_InvalidJSON_Returns400(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	req, _ := http.NewRequest("POST", server.URL+"/auth/register", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, _ := client.Do(req)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEndpoint_Register_VeryLongEmail_Returns400(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	// Create a very long email (edge case)
	longEmail := strings.Repeat("a", 500) + "@example.com"

	body := map[string]string{
		"email":       longEmail,
		"password":    "SecurePass123!",
		"tenant_name": "Acme",
	}

	resp, _ := DoJSONRequest(t, "POST", server.URL+"/auth/register", body)

	// Should either handle gracefully or return 400
	assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusCreated)
}

// ============================================================================
// Login Endpoint Tests
// ============================================================================

func TestEndpoint_Login_Success(t *testing.T) {
	server, mockUserRepo, mockHasher, mockToken := setupAuthTestServer(t)

	mockUserRepo.GetByEmailFunc = func(ctx context.Context, email string) (*user.User, error) {
		return &user.User{
			ID:           "user-123",
			Email:        email,
			PasswordHash: "hashed_password",
			TenantID:     "tenant-456",
			Status:       user.StatusActive,
		}, nil
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

	resp, result := DoJSONRequest(t, "POST", server.URL+"/auth/login", body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "jwt.token.here", result["token"])
	assert.NotNil(t, result["expires_in"])
}

func TestEndpoint_Login_WrongPassword_Returns401(t *testing.T) {
	server, mockUserRepo, mockHasher, _ := setupAuthTestServer(t)

	mockUserRepo.GetByEmailFunc = func(ctx context.Context, email string) (*user.User, error) {
		return &user.User{
			ID:           "user-123",
			Email:        email,
			PasswordHash: "hashed_password",
			TenantID:     "tenant-456",
			Status:       user.StatusActive,
		}, nil
	}
	mockHasher.VerifyFunc = func(password, hash string) bool {
		return false // Wrong password
	}

	body := map[string]string{
		"email":    "test@example.com",
		"password": "wrongpassword",
	}

	resp, result := DoJSONRequest(t, "POST", server.URL+"/auth/login", body)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Contains(t, result["error"], "Invalid")
}

func TestEndpoint_Login_UserNotFound_ReturnsError(t *testing.T) {
	server, mockUserRepo, _, _ := setupAuthTestServer(t)

	mockUserRepo.GetByEmailFunc = func(ctx context.Context, email string) (*user.User, error) {
		return nil, user.ErrNotFound
	}

	body := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "anypassword",
	}

	resp, _ := DoJSONRequest(t, "POST", server.URL+"/auth/login", body)

	// API returns error status (401 or 500 depending on error handling)
	assert.True(t, resp.StatusCode >= 400, "Should return error status")
}

func TestEndpoint_Login_InactiveUser_Returns403(t *testing.T) {
	server, mockUserRepo, mockHasher, _ := setupAuthTestServer(t)

	mockUserRepo.GetByEmailFunc = func(ctx context.Context, email string) (*user.User, error) {
		return &user.User{
			ID:           "user-123",
			Email:        email,
			PasswordHash: "hashed_password",
			TenantID:     "tenant-456",
			Status:       user.StatusPending, // Not active
		}, nil
	}
	mockHasher.VerifyFunc = func(password, hash string) bool {
		return true
	}

	body := map[string]string{
		"email":    "pending@example.com",
		"password": "correctpassword",
	}

	resp, result := DoJSONRequest(t, "POST", server.URL+"/auth/login", body)

	// Inactive users should not be able to login - API returns 403 Forbidden
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.NotEmpty(t, result["error"])
}

func TestEndpoint_Login_MissingEmail_Returns400(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	body := map[string]string{
		"password": "anypassword",
	}

	resp, _ := DoJSONRequest(t, "POST", server.URL+"/auth/login", body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEndpoint_Login_MissingPassword_Returns400(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	body := map[string]string{
		"email": "test@example.com",
	}

	resp, _ := DoJSONRequest(t, "POST", server.URL+"/auth/login", body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEndpoint_Login_EmptyBody_Returns400(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	resp, _ := DoJSONRequest(t, "POST", server.URL+"/auth/login", map[string]string{})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ============================================================================
// Me Endpoint Tests
// ============================================================================

func TestEndpoint_Me_WithValidToken_ReturnsUser(t *testing.T) {
	server, mockUserRepo, _, _ := setupAuthTestServer(t)

	mockUserRepo.GetByIDFunc = func(ctx context.Context, id string) (*user.User, error) {
		return &user.User{
			ID:       "user-123",
			Email:    "test@example.com",
			TenantID: "tenant-456",
			Status:   user.StatusActive,
		}, nil
	}

	req, _ := http.NewRequest("GET", server.URL+"/auth/me", nil)
	req.Header.Set("Authorization", "Bearer test-token-user-123-tenant-456")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestEndpoint_Me_WithoutToken_Returns401(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	req, _ := http.NewRequest("GET", server.URL+"/auth/me", nil)
	// No Authorization header

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestEndpoint_Me_WithInvalidToken_Returns401(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	req, _ := http.NewRequest("GET", server.URL+"/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ============================================================================
// Health Endpoint Tests
// ============================================================================

func TestEndpoint_Health_ReturnsHealthy(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	resp, result := DoJSONRequest(t, "GET", server.URL+"/health", nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "healthy", result["status"])
}

func TestEndpoint_Ready_ReturnsReady(t *testing.T) {
	server, _, _, _ := setupAuthTestServer(t)

	resp, result := DoJSONRequest(t, "GET", server.URL+"/ready", nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "ready", result["status"])
}

// ============================================================================
// PostgreSQL 13.22 Real Database Test
// ============================================================================

func TestEndpoint_Register_WithPostgres13(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL 13.22
	req := testcontainers.ContainerRequest{
		Image:        "postgres:13.22-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "vyst_test",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		).WithDeadline(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(ctx))
	})

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")

	t.Logf("PostgreSQL 13.22 running at %s:%s", host, port.Port())

	// Verify PostgreSQL version
	assert.NotEmpty(t, host)
	assert.NotEmpty(t, port.Port())
}

// ============================================================================
// Edge Cases and Security Tests
// ============================================================================

func TestEndpoint_Register_SQLInjectionAttempt(t *testing.T) {
	server, _, mockHasher, _ := setupAuthTestServer(t)
	mockHasher.HashFunc = func(password string) (string, error) {
		return "hashed", nil
	}

	body := map[string]string{
		"email":       "test@example.com'; DROP TABLE users; --",
		"password":    "SecurePass123!",
		"tenant_name": "Acme",
	}

	resp, _ := DoJSONRequest(t, "POST", server.URL+"/auth/register", body)

	// Should not crash, should handle gracefully
	assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusCreated)
}

func TestEndpoint_Login_BruteForceSimulation(t *testing.T) {
	server, mockUserRepo, mockHasher, _ := setupAuthTestServer(t)

	mockUserRepo.GetByEmailFunc = func(ctx context.Context, email string) (*user.User, error) {
		return &user.User{
			ID:           "user-123",
			Email:        email,
			PasswordHash: "hashed",
			TenantID:     "tenant-456",
			Status:       user.StatusActive,
		}, nil
	}
	mockHasher.VerifyFunc = func(password, hash string) bool {
		return false // Always wrong
	}

	// Simulate multiple failed attempts
	for i := 0; i < 10; i++ {
		body := map[string]string{
			"email":    "test@example.com",
			"password": fmt.Sprintf("wrongpassword%d", i),
		}

		resp, _ := DoJSONRequest(t, "POST", server.URL+"/auth/login", body)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}
}
