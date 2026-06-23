package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/captcha"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/handlers"
	"github.com/pgdepaula/vyst-openauth/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// CAPTCHA Integration Tests
// ============================================================================
// These tests verify the complete CAPTCHA flow including:
// - Config endpoint returns correct data
// - Register with/without CAPTCHA
// - Login with/without CAPTCHA
// - CAPTCHA validation errors are handled correctly

// TestCaptchaConfigEndpoint_ReturnsCorrectConfig tests the captcha-config endpoint.
func TestCaptchaConfigEndpoint_ReturnsCorrectConfig(t *testing.T) {
	// Setup mock captcha service
	captchaSvc := mocks.NewMockCaptchaService(true, "test-site-key-12345")

	handler := createTestAuthHandlerWithCaptcha(captchaSvc)

	req := httptest.NewRequest("GET", "/auth/captcha-config", nil)
	rec := httptest.NewRecorder()

	handler.GetCaptchaSiteKey(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-site-key-12345", response["site_key"])
	assert.Equal(t, true, response["enabled"])
}

// TestCaptchaConfigEndpoint_DisabledCaptcha tests the config when CAPTCHA is disabled.
func TestCaptchaConfigEndpoint_DisabledCaptcha(t *testing.T) {
	captchaSvc := mocks.NewMockCaptchaService(false, "")

	handler := createTestAuthHandlerWithCaptcha(captchaSvc)

	req := httptest.NewRequest("GET", "/auth/captcha-config", nil)
	rec := httptest.NewRecorder()

	handler.GetCaptchaSiteKey(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	assert.Equal(t, false, response["enabled"])
}

// TestRegister_WithCaptchaEnabled_RequiresToken tests CAPTCHA validation on register.
func TestRegister_WithCaptchaEnabled_RequiresToken(t *testing.T) {
	captchaSvc := mocks.NewMockCaptchaService(true, "test-site-key")
	captchaSvc.ValidateTokenFunc = func(ctx context.Context, token, remoteIP string) error {
		if token == "" {
			return captcha.ErrCaptchaTokenMissing
		}
		return nil
	}

	handler := createTestAuthHandlerWithCaptcha(captchaSvc)

	body := map[string]string{
		"email":       "test@example.com",
		"password":    "securepassword123",
		"tenant_name": "Test Corp",
		// Missing captcha_token
	}
	jsonBody := mustMarshalJSON(t, body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	// Should return 400 due to missing CAPTCHA token
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Contains(t, response["error"], "CAPTCHA")
}

// TestRegister_WithValidCaptchaToken_Succeeds tests successful registration with CAPTCHA.
func TestRegister_WithValidCaptchaToken_Succeeds(t *testing.T) {
	captchaSvc := mocks.NewMockCaptchaService(true, "test-site-key")
	captchaSvc.ValidateTokenFunc = func(ctx context.Context, token, remoteIP string) error {
		// #nosec G101 -- test fixture token, not a credential.
		if token == "valid-captcha-token" {
			return nil
		}
		return captcha.ErrCaptchaInvalid
	}

	handler := createTestAuthHandlerWithCaptcha(captchaSvc)

	body := map[string]string{
		"email":         "test@example.com",
		"password":      "securepassword123",
		"tenant_name":   "Test Corp",
		"captcha_token": "valid-captcha-token",
	}
	jsonBody := mustMarshalJSON(t, body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	// With valid CAPTCHA, should proceed (may be 201 or 400 from other validation)
	// The key is that we don't get a CAPTCHA error
	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	// Should not be a CAPTCHA error
	if rec.Code == http.StatusBadRequest {
		assert.NotContains(t, response["error"], "CAPTCHA")
	}

	// Verify CAPTCHA was validated
	assert.Len(t, captchaSvc.ValidateTokenCalls, 1)
	assert.Equal(t, "valid-captcha-token", captchaSvc.ValidateTokenCalls[0].Token)
}

// TestLogin_WithCaptchaEnabled_RequiresToken tests CAPTCHA validation on login.
func TestLogin_WithCaptchaEnabled_RequiresToken(t *testing.T) {
	captchaSvc := mocks.NewMockCaptchaService(true, "test-site-key")
	captchaSvc.ValidateTokenFunc = func(ctx context.Context, token, remoteIP string) error {
		if token == "" {
			return captcha.ErrCaptchaTokenMissing
		}
		return nil
	}

	handler := createTestAuthHandlerWithCaptcha(captchaSvc)

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		// Missing captcha_token
	}
	jsonBody := mustMarshalJSON(t, body)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	// Should return 400 due to missing CAPTCHA token
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Contains(t, response["error"], "CAPTCHA")
}

// TestRegister_WithInvalidCaptchaToken_Fails tests CAPTCHA rejection with invalid token.
func TestRegister_WithInvalidCaptchaToken_Fails(t *testing.T) {
	captchaSvc := mocks.NewMockCaptchaService(true, "test-site-key")
	captchaSvc.ValidateTokenFunc = func(ctx context.Context, token, remoteIP string) error {
		return captcha.ErrCaptchaInvalid
	}

	handler := createTestAuthHandlerWithCaptcha(captchaSvc)

	body := map[string]string{
		"email":         "test@example.com",
		"password":      "securepassword123",
		"tenant_name":   "Test Corp",
		"captcha_token": "invalid-token",
	}
	jsonBody := mustMarshalJSON(t, body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Contains(t, response["error"], "CAPTCHA")
}

// TestRegister_WithExpiredCaptchaToken_ReturnsAppropriateError tests expired CAPTCHA handling.
func TestRegister_WithExpiredCaptchaToken_ReturnsAppropriateError(t *testing.T) {
	captchaSvc := mocks.NewMockCaptchaService(true, "test-site-key")
	captchaSvc.ValidateTokenFunc = func(ctx context.Context, token, remoteIP string) error {
		return captcha.ErrCaptchaExpired
	}

	handler := createTestAuthHandlerWithCaptcha(captchaSvc)

	body := map[string]string{
		"email":         "test@example.com",
		"password":      "securepassword123",
		"tenant_name":   "Test Corp",
		"captcha_token": "expired-token",
	}
	jsonBody := mustMarshalJSON(t, body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Contains(t, response["error"], "expired")
}

// TestCaptchaDisabled_AllowsRequestsWithoutToken tests that disabled CAPTCHA allows requests.
func TestCaptchaDisabled_AllowsRequestsWithoutToken(t *testing.T) {
	captchaSvc := mocks.NewMockCaptchaService(false, "")

	handler := createTestAuthHandlerWithCaptcha(captchaSvc)

	body := map[string]string{
		"email":       "test@example.com",
		"password":    "securepassword123",
		"tenant_name": "Test Corp",
		// No captcha_token - should be fine when CAPTCHA is disabled
	}
	jsonBody := mustMarshalJSON(t, body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	// Should not fail with CAPTCHA error
	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	if rec.Code == http.StatusBadRequest {
		assert.NotContains(t, response["error"], "CAPTCHA")
	}

	// Verify CAPTCHA was NOT validated (disabled)
	assert.Len(t, captchaSvc.ValidateTokenCalls, 0)
}

// TestCaptchaValidation_PassesRemoteIP tests that remote IP is passed to CAPTCHA service.
func TestCaptchaValidation_PassesRemoteIP(t *testing.T) {
	captchaSvc := mocks.NewMockCaptchaService(true, "test-site-key")
	var capturedRemoteIP string
	captchaSvc.ValidateTokenFunc = func(ctx context.Context, token, remoteIP string) error {
		capturedRemoteIP = remoteIP
		return nil
	}

	handler := createTestAuthHandlerWithCaptcha(captchaSvc)

	body := map[string]string{
		"email":         "test@example.com",
		"password":      "securepassword123",
		"tenant_name":   "Test Corp",
		"captcha_token": "valid-token",
	}
	jsonBody := mustMarshalJSON(t, body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	// Verify remote IP was captured
	assert.Equal(t, "192.168.1.100", capturedRemoteIP)
}

// ============================================================================
// Helper Functions
// ============================================================================

func mustMarshalJSON(t *testing.T, body interface{}) []byte {
	t.Helper()
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	return jsonBody
}

func createTestAuthHandlerWithCaptcha(captchaSvc ports.CaptchaService) *handlers.AuthHandler {
	mockUserRepo := &mocks.MockUserRepository{}
	mockHasher := &mocks.MockPasswordHasher{
		HashFunc: func(password string) (string, error) {
			return "hashed_password", nil
		},
	}
	mockTokenSvc := &mocks.MockTokenService{}
	mockTM := &mocks.MockTransactionManager{}
	mockTenantRepo := &mocks.MockTenantRepository{}
	mockPolicyRepo := &mocks.MockPolicyRepository{}
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
	authSvc := service.NewAuthService(mockUserRepo, mockPolicyRepo, mockCompanyUserRepo, mockSessionRepo, mockHasher, mockTokenSvc, mockNotifier, mockLogger)

	return handlers.NewAuthHandler(registrationSvc, authSvc, nil, captchaSvc, nil)
}
