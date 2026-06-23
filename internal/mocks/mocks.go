// Package mocks provides mock implementations of all interfaces for unit testing.
package mocks

import (
	"context"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/apikey"
	"github.com/pgdepaula/vyst-openauth/internal/domain/captcha"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/domain/invitation"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"github.com/pgdepaula/vyst-openauth/internal/domain/session"
	"github.com/pgdepaula/vyst-openauth/internal/domain/tenant"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

// --- User Repository Mock ---

type MockUserRepository struct {
	CreateFunc                 func(ctx context.Context, u *user.User) error
	GetByEmailFunc             func(ctx context.Context, email string) (*user.User, error)
	GetByIDFunc                func(ctx context.Context, id string) (*user.User, error)
	GetByResetTokenFunc        func(ctx context.Context, token string) (*user.User, error)
	GetByVerificationTokenFunc func(ctx context.Context, token string) (*user.User, error)
	UpdateFunc                 func(ctx context.Context, u *user.User) error

	CreateCalls                 []CreateUserCall
	GetByEmailCalls             []string
	GetByIDCalls                []string
	GetByResetTokenCalls        []string
	GetByVerificationTokenCalls []string
}

type CreateUserCall struct {
	User *user.User
}

func (m *MockUserRepository) Create(ctx context.Context, u *user.User) error {
	m.CreateCalls = append(m.CreateCalls, CreateUserCall{User: u})
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, u)
	}
	return nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	m.GetByEmailCalls = append(m.GetByEmailCalls, email)
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	m.GetByIDCalls = append(m.GetByIDCalls, id)
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByResetToken(ctx context.Context, token string) (*user.User, error) {
	m.GetByResetTokenCalls = append(m.GetByResetTokenCalls, token)
	if m.GetByResetTokenFunc != nil {
		return m.GetByResetTokenFunc(ctx, token)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	m.GetByVerificationTokenCalls = append(m.GetByVerificationTokenCalls, token)
	if m.GetByVerificationTokenFunc != nil {
		return m.GetByVerificationTokenFunc(ctx, token)
	}
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, u *user.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, u)
	}
	return nil
}

// --- Tenant Repository Mock ---

type MockTenantRepository struct {
	CreateFunc           func(ctx context.Context, t *tenant.Tenant) error
	GetByIDFunc          func(ctx context.Context, id string) (*tenant.Tenant, error)
	UpdateFunc           func(ctx context.Context, t *tenant.Tenant) error
	SetCurrentTenantFunc func(ctx context.Context, tenantID string) error
	ListFunc             func(ctx context.Context) ([]*tenant.Tenant, error)

	CreateCalls           []*tenant.Tenant
	SetCurrentTenantCalls []string
}

func (m *MockTenantRepository) Create(ctx context.Context, t *tenant.Tenant) error {
	m.CreateCalls = append(m.CreateCalls, t)
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, t)
	}
	return nil
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id string) (*tenant.Tenant, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockTenantRepository) Update(ctx context.Context, t *tenant.Tenant) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, t)
	}
	return nil
}

func (m *MockTenantRepository) SetCurrentTenant(ctx context.Context, tenantID string) error {
	m.SetCurrentTenantCalls = append(m.SetCurrentTenantCalls, tenantID)
	if m.SetCurrentTenantFunc != nil {
		return m.SetCurrentTenantFunc(ctx, tenantID)
	}
	return nil
}

func (m *MockTenantRepository) List(ctx context.Context) ([]*tenant.Tenant, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

// --- Policy Repository Mock ---

type MockPolicyRepository struct {
	CheckFunc           func(ctx context.Context, subject, relation, object string) (bool, error)
	WriteTupleFunc      func(ctx context.Context, tuple policy.Tuple) error
	DeleteTupleFunc     func(ctx context.Context, tuple policy.Tuple) error
	GetRolesForUserFunc func(ctx context.Context, userID string) ([]string, error)

	CheckCalls      []PolicyCheckCall
	WriteTupleCalls []policy.Tuple
}

type PolicyCheckCall struct {
	Subject  string
	Relation string
	Object   string
}

func (m *MockPolicyRepository) Check(ctx context.Context, subject, relation, object string) (bool, error) {
	m.CheckCalls = append(m.CheckCalls, PolicyCheckCall{Subject: subject, Relation: relation, Object: object})
	if m.CheckFunc != nil {
		return m.CheckFunc(ctx, subject, relation, object)
	}
	return false, nil
}

func (m *MockPolicyRepository) WriteTuple(ctx context.Context, tuple policy.Tuple) error {
	m.WriteTupleCalls = append(m.WriteTupleCalls, tuple)
	if m.WriteTupleFunc != nil {
		return m.WriteTupleFunc(ctx, tuple)
	}
	return nil
}

func (m *MockPolicyRepository) DeleteTuple(ctx context.Context, tuple policy.Tuple) error {
	if m.DeleteTupleFunc != nil {
		return m.DeleteTupleFunc(ctx, tuple)
	}
	return nil
}

func (m *MockPolicyRepository) GetRolesForUser(ctx context.Context, userID string) ([]string, error) {
	if m.GetRolesForUserFunc != nil {
		return m.GetRolesForUserFunc(ctx, userID)
	}
	return []string{}, nil
}

// --- Password Hasher Mock ---

type MockPasswordHasher struct {
	HashFunc   func(password string) (string, error)
	VerifyFunc func(password, hash string) bool

	HashCalls   []string
	VerifyCalls []VerifyCall
}

type VerifyCall struct {
	Password string
	Hash     string
}

func (m *MockPasswordHasher) Hash(password string) (string, error) {
	m.HashCalls = append(m.HashCalls, password)
	if m.HashFunc != nil {
		return m.HashFunc(password)
	}
	return "mocked_hash_" + password, nil
}

func (m *MockPasswordHasher) Verify(password, hash string) bool {
	m.VerifyCalls = append(m.VerifyCalls, VerifyCall{Password: password, Hash: hash})
	if m.VerifyFunc != nil {
		return m.VerifyFunc(password, hash)
	}
	return password == hash // Simple mock: password equals hash
}

// --- Token Service Mock ---

type MockTokenService struct {
	GenerateTokenFunc          func(userID, tenantID string, roles []string, activeCompanyID, companyRole, identityType string) (string, error)
	GenerateEncryptedTokenFunc func(payload map[string]interface{}) (string, error)
	ValidateTokenFunc          func(tokenString string) (*ports.Claims, error)

	GenerateTokenCalls []GenerateTokenCall
}

type GenerateTokenCall struct {
	UserID          string
	TenantID        string
	Roles           []string
	ActiveCompanyID string
	CompanyRole     string
	IdentityType    string
}

func (m *MockTokenService) GenerateToken(userID, tenantID string, roles []string, activeCompanyID, companyRole, identityType string) (string, error) {
	m.GenerateTokenCalls = append(m.GenerateTokenCalls, GenerateTokenCall{
		UserID: userID, TenantID: tenantID, Roles: roles, ActiveCompanyID: activeCompanyID, CompanyRole: companyRole, IdentityType: identityType,
	})
	if m.GenerateTokenFunc != nil {
		return m.GenerateTokenFunc(userID, tenantID, roles, activeCompanyID, companyRole, identityType)
	}
	return "mock_token_" + userID, nil
}

func (m *MockTokenService) GenerateEncryptedToken(payload map[string]interface{}) (string, error) {
	if m.GenerateEncryptedTokenFunc != nil {
		return m.GenerateEncryptedTokenFunc(payload)
	}
	return "mock_encrypted_token", nil
}

func (m *MockTokenService) ValidateToken(tokenString string) (*ports.Claims, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(tokenString)
	}
	return &ports.Claims{
		UserID:   "mock_user_id",
		TenantID: "mock_tenant_id",
		Roles:    []string{"user"},
	}, nil
}

func (m *MockTokenService) GenerateRefreshToken() string {
	return "mock_refresh_token"
}

func (m *MockTokenService) GenerateID() string {
	return "mock_id"
}

// --- Session Repository Mock ---

type MockSessionRepository struct {
	CreateFunc            func(ctx context.Context, s *session.Session) error
	GetByIDFunc           func(ctx context.Context, id string) (*session.Session, error)
	GetByRefreshTokenFunc func(ctx context.Context, token string) (*session.Session, error)
	UpdateFunc            func(ctx context.Context, s *session.Session) error
	DeleteFunc            func(ctx context.Context, id string) error
	DeleteByUserIDFunc    func(ctx context.Context, userID string) error

	CreateCalls []CreateSessionCall
}

type CreateSessionCall struct {
	Session *session.Session
}

func (m *MockSessionRepository) Create(ctx context.Context, s *session.Session) error {
	m.CreateCalls = append(m.CreateCalls, CreateSessionCall{Session: s})
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, s)
	}
	return nil
}

func (m *MockSessionRepository) GetByID(ctx context.Context, id string) (*session.Session, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockSessionRepository) GetByRefreshToken(ctx context.Context, token string) (*session.Session, error) {
	if m.GetByRefreshTokenFunc != nil {
		return m.GetByRefreshTokenFunc(ctx, token)
	}
	return nil, nil
}

func (m *MockSessionRepository) Update(ctx context.Context, s *session.Session) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, s)
	}
	return nil
}

func (m *MockSessionRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	if m.DeleteByUserIDFunc != nil {
		return m.DeleteByUserIDFunc(ctx, userID)
	}
	return nil
}

// --- Outbox Publisher Mock ---

type MockOutboxPublisher struct {
	PublishFunc func(ctx context.Context, evt event.Event) error

	PublishCalls []event.Event
}

func (m *MockOutboxPublisher) Publish(ctx context.Context, evt event.Event) error {
	m.PublishCalls = append(m.PublishCalls, evt)
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, evt)
	}
	return nil
}

// --- Transaction Manager Mock ---

type MockTransactionManager struct {
	RunInTransactionFunc func(ctx context.Context, fn func(ctx context.Context) error) error

	RunInTransactionCalls int
}

func (m *MockTransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	m.RunInTransactionCalls++
	if m.RunInTransactionFunc != nil {
		return m.RunInTransactionFunc(ctx, fn)
	}
	// By default, just execute the function directly
	return fn(ctx)
}

// --- Event Bus Mock ---

type MockEventBus struct {
	PublishFunc   func(ctx context.Context, evt event.Event) error
	SubscribeFunc func(eventType event.EventType, handler event.Handler)

	PublishCalls   []event.Event
	SubscribeCalls []event.EventType
}

func (m *MockEventBus) Publish(ctx context.Context, evt event.Event) error {
	m.PublishCalls = append(m.PublishCalls, evt)
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, evt)
	}
	return nil
}

func (m *MockEventBus) Subscribe(eventType event.EventType, handler event.Handler) func() {
	m.SubscribeCalls = append(m.SubscribeCalls, eventType)
	if m.SubscribeFunc != nil {
		m.SubscribeFunc(eventType, handler)
	}
	return func() {}
}

// --- Notification Service Mock ---

type MockNotificationService struct {
	SendEmailFunc func(to, subject, body string) error
	SendSMSFunc   func(to, content string) error

	SendEmailCalls []SendEmailCall
	SendSMSCalls   []SendSMSCall
}

type SendEmailCall struct {
	To      string
	Subject string
	Body    string
}

type SendSMSCall struct {
	To      string
	Content string
}

func (m *MockNotificationService) SendEmail(to, subject, body string) error {
	m.SendEmailCalls = append(m.SendEmailCalls, SendEmailCall{To: to, Subject: subject, Body: body})
	if m.SendEmailFunc != nil {
		return m.SendEmailFunc(to, subject, body)
	}
	return nil
}

func (m *MockNotificationService) SendSMS(to, content string) error {
	m.SendSMSCalls = append(m.SendSMSCalls, SendSMSCall{To: to, Content: content})
	if m.SendSMSFunc != nil {
		return m.SendSMSFunc(to, content)
	}
	return nil
}

// --- Helper Functions ---

// NewTestUser creates a user for testing purposes
func NewTestUser(id, email, tenantID string) *user.User {
	now := time.Now()
	return &user.User{
		ID:           id,
		Email:        email,
		PasswordHash: "test_hash",
		TenantID:     tenantID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// NewTestTenant creates a tenant for testing purposes
func NewTestTenant(id, name string) *tenant.Tenant {
	now := time.Now()
	return &tenant.Tenant{
		ID:        id,
		Name:      name,
		Status:    tenant.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// MockAPIKeyValidator is a mock implementation of middleware.APIKeyValidator
type MockAPIKeyValidator struct {
	ValidateAPIKeyFunc func(ctx context.Context, rawKey string) (*apikey.APIKey, error)
}

func (m *MockAPIKeyValidator) ValidateAPIKey(ctx context.Context, rawKey string) (*apikey.APIKey, error) {
	if m.ValidateAPIKeyFunc != nil {
		return m.ValidateAPIKeyFunc(ctx, rawKey)
	}
	return nil, nil
}

// --- Mock Logger ---

// MockLogger implements ports.Logger for testing.
type MockLogger struct{}

func (m *MockLogger) Debug(msg string, args ...any) {}
func (m *MockLogger) Info(msg string, args ...any)  {}
func (m *MockLogger) Warn(msg string, args ...any)  {}
func (m *MockLogger) Error(msg string, args ...any) {}
func (m *MockLogger) With(args ...any) ports.Logger {
	return m
}
func (m *MockLogger) WithContext(ctx context.Context) ports.Logger {
	return m
}

// --- Captcha Service Mock ---

// MockCaptchaService implements ports.CaptchaService for testing.
type MockCaptchaService struct {
	ValidateTokenFunc           func(ctx context.Context, token, remoteIP string) error
	ValidateTokenWithResultFunc func(ctx context.Context, token, remoteIP string) (*captcha.Result, error)
	GetConfigFunc               func() ports.CaptchaConfig
	IsEnabledFunc               func() bool

	ValidateTokenCalls []ValidateTokenCall
	Enabled            bool
	SiteKey            string
}

// ValidateTokenCall records a call to ValidateToken.
type ValidateTokenCall struct {
	Token    string
	RemoteIP string
}

func (m *MockCaptchaService) ValidateToken(ctx context.Context, token, remoteIP string) error {
	m.ValidateTokenCalls = append(m.ValidateTokenCalls, ValidateTokenCall{Token: token, RemoteIP: remoteIP})
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(ctx, token, remoteIP)
	}
	return nil
}

func (m *MockCaptchaService) ValidateTokenWithResult(ctx context.Context, token, remoteIP string) (*captcha.Result, error) {
	m.ValidateTokenCalls = append(m.ValidateTokenCalls, ValidateTokenCall{Token: token, RemoteIP: remoteIP})
	if m.ValidateTokenWithResultFunc != nil {
		return m.ValidateTokenWithResultFunc(ctx, token, remoteIP)
	}
	return captcha.NewResult(true, 0.9), nil
}

func (m *MockCaptchaService) GetConfig() ports.CaptchaConfig {
	if m.GetConfigFunc != nil {
		return m.GetConfigFunc()
	}
	return ports.CaptchaConfig{
		SiteKey: m.SiteKey,
		Enabled: m.Enabled,
	}
}

func (m *MockCaptchaService) IsEnabled() bool {
	if m.IsEnabledFunc != nil {
		return m.IsEnabledFunc()
	}
	return m.Enabled
}

// NewMockCaptchaService creates a mock captcha service with default settings.
func NewMockCaptchaService(enabled bool, siteKey string) *MockCaptchaService {
	return &MockCaptchaService{
		Enabled: enabled,
		SiteKey: siteKey,
	}
}

// --- Invitation Repository Mock ---

type MockInvitationRepository struct {
	CreateFunc               func(ctx context.Context, inv *invitation.Invitation) error
	GetByTokenFunc           func(ctx context.Context, token string) (*invitation.Invitation, error)
	GetByEmailAndCompanyFunc func(ctx context.Context, email, companyID string) (*invitation.Invitation, error)
	UpdateFunc               func(ctx context.Context, inv *invitation.Invitation) error
	ListByCompanyFunc        func(ctx context.Context, companyID string) ([]*invitation.Invitation, error)

	CreateCalls []CreateInvitationCall
}

type CreateInvitationCall struct {
	Invitation *invitation.Invitation
}

func (m *MockInvitationRepository) Create(ctx context.Context, inv *invitation.Invitation) error {
	m.CreateCalls = append(m.CreateCalls, CreateInvitationCall{Invitation: inv})
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, inv)
	}
	return nil
}

func (m *MockInvitationRepository) GetByToken(ctx context.Context, token string) (*invitation.Invitation, error) {
	if m.GetByTokenFunc != nil {
		return m.GetByTokenFunc(ctx, token)
	}
	return nil, nil
}

func (m *MockInvitationRepository) GetByEmailAndCompany(ctx context.Context, email, companyID string) (*invitation.Invitation, error) {
	if m.GetByEmailAndCompanyFunc != nil {
		return m.GetByEmailAndCompanyFunc(ctx, email, companyID)
	}
	return nil, nil
}

func (m *MockInvitationRepository) Update(ctx context.Context, inv *invitation.Invitation) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, inv)
	}
	return nil
}

func (m *MockInvitationRepository) ListByCompany(ctx context.Context, companyID string) ([]*invitation.Invitation, error) {
	if m.ListByCompanyFunc != nil {
		return m.ListByCompanyFunc(ctx, companyID)
	}
	return nil, nil
}
