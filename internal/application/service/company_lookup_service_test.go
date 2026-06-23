package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks

type mockCompanyInfoRepo struct {
	mock.Mock
}

func (m *mockCompanyInfoRepo) Save(ctx context.Context, info *company.CompanyInfo) error {
	args := m.Called(ctx, info)
	return args.Error(0)
}

func (m *mockCompanyInfoRepo) GetByCNPJ(ctx context.Context, cnpj string) (*company.CompanyInfo, error) {
	args := m.Called(ctx, cnpj)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*company.CompanyInfo), args.Error(1)
}

func (m *mockCompanyInfoRepo) SearchByName(ctx context.Context, query string, limit int) ([]*company.CompanyInfo, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*company.CompanyInfo), args.Error(1)
}

type mockCompanyDataPort struct {
	mock.Mock
}

func (m *mockCompanyDataPort) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockCompanyDataPort) GetByCNPJ(ctx context.Context, cnpj string) (*company.CompanyInfo, error) {
	args := m.Called(ctx, cnpj)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*company.CompanyInfo), args.Error(1)
}

func (m *mockCompanyDataPort) SearchByName(ctx context.Context, query string, limit int) ([]*company.CompanyInfo, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*company.CompanyInfo), args.Error(1)
}

type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Debug(msg string, args ...any)                {}
func (m *mockLogger) Info(msg string, args ...any)                 {}
func (m *mockLogger) Warn(msg string, args ...any)                 {}
func (m *mockLogger) Error(msg string, args ...any)                {}
func (m *mockLogger) With(args ...any) ports.Logger                { return m }
func (m *mockLogger) WithContext(ctx context.Context) ports.Logger { return m }

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(ctx context.Context, e event.Event) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

func TestCompanyLookupService_GetByCNPJ_CacheHit(t *testing.T) {
	repo := new(mockCompanyInfoRepo)
	provider := new(mockCompanyDataPort)
	logger := new(mockLogger)
	pub := new(mockPublisher)

	svc := service.NewCompanyLookupService(repo, []ports.CompanyDataPort{provider}, logger, pub, nil, nil)
	ctx := context.Background()

	validCNPJ := "00.000.000/0001-91"
	normalizedCNPJ := company.NormalizeCNPJ(validCNPJ)

	expectedInfo := &company.CompanyInfo{
		CNPJ:          normalizedCNPJ,
		RazaoSocial:   "Test Company",
		Situacao:      company.SituationActive,
		LastFetchedAt: time.Now(),
	}

	repo.On("GetByCNPJ", ctx, normalizedCNPJ).Return(expectedInfo, nil)
	pub.On("Publish", ctx, mock.AnythingOfType("event.Event")).Return(nil)

	info, err := svc.GetByCNPJ(ctx, "test-tenant", validCNPJ)

	assert.NoError(t, err)
	assert.Equal(t, expectedInfo, info)

	repo.AssertExpectations(t)
	provider.AssertNotCalled(t, "GetByCNPJ", mock.Anything, mock.Anything)
}

func TestCompanyLookupService_GetByCNPJ_FallbackToProvider(t *testing.T) {
	repo := new(mockCompanyInfoRepo)
	provider := new(mockCompanyDataPort)
	logger := new(mockLogger)
	pub := new(mockPublisher)

	svc := service.NewCompanyLookupService(repo, []ports.CompanyDataPort{provider}, logger, pub, nil, nil)
	ctx := context.Background()

	validCNPJ := "00.000.000/0001-91"
	normalizedCNPJ := company.NormalizeCNPJ(validCNPJ)

	expectedInfo := &company.CompanyInfo{
		CNPJ:          normalizedCNPJ,
		RazaoSocial:   "Provider Company",
		Situacao:      company.SituationActive,
		LastFetchedAt: time.Now(),
	}

	repo.On("GetByCNPJ", ctx, normalizedCNPJ).Return(nil, company.ErrCompanyInfoNotFound)
	provider.On("Name").Return("MockProvider")
	provider.On("GetByCNPJ", ctx, normalizedCNPJ).Return(expectedInfo, nil)
	repo.On("Save", ctx, expectedInfo).Return(nil)
	pub.On("Publish", ctx, mock.AnythingOfType("event.Event")).Return(nil)

	info, err := svc.GetByCNPJ(ctx, "test-tenant", validCNPJ)

	assert.NoError(t, err)
	assert.Equal(t, expectedInfo, info)

	repo.AssertExpectations(t)
	provider.AssertExpectations(t)
	pub.AssertExpectations(t)
}

func TestCompanyLookupService_GetByCNPJ_Invalid(t *testing.T) {
	svc := service.NewCompanyLookupService(nil, nil, new(mockLogger), nil, nil, nil)
	ctx := context.Background()

	invalidCNPJ := "11.111.111/1111-11"

	info, err := svc.GetByCNPJ(ctx, "test-tenant", invalidCNPJ)

	assert.ErrorIs(t, err, company.ErrCNPJInvalid)
	assert.Nil(t, info)
}

func TestCompanyLookupService_GetByCNPJ_AllProvidersFail(t *testing.T) {
	repo := new(mockCompanyInfoRepo)
	provider := new(mockCompanyDataPort)
	logger := new(mockLogger)
	pub := new(mockPublisher)

	svc := service.NewCompanyLookupService(repo, []ports.CompanyDataPort{provider}, logger, pub, nil, nil)
	ctx := context.Background()

	validCNPJ := "00.000.000/0001-91"
	normalizedCNPJ := company.NormalizeCNPJ(validCNPJ)

	repo.On("GetByCNPJ", ctx, normalizedCNPJ).Return(nil, company.ErrCompanyInfoNotFound)
	provider.On("Name").Return("MockProvider")
	provider.On("GetByCNPJ", ctx, normalizedCNPJ).Return(nil, errors.New("API Timeout"))

	info, err := svc.GetByCNPJ(ctx, "test-tenant", validCNPJ)

	assert.Error(t, err)
	assert.Equal(t, "API Timeout", err.Error())
	assert.Nil(t, info)

	repo.AssertExpectations(t)
	provider.AssertExpectations(t)
}
