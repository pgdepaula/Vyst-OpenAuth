package resolvers

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import (
	"context"

	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

// CompanyLookupUseCase defines the operations for company lookups.
type CompanyLookupUseCase interface {
	GetByCNPJ(ctx context.Context, tenantID, cnpj string) (*company.CompanyInfo, error)
	SearchByName(ctx context.Context, tenantID, query string, limit int) ([]*company.CompanyInfo, error)
	Lookup(ctx context.Context, tenantID, query string, limit int) ([]*company.CompanyInfo, error)
}

// Resolver holds all dependencies for GraphQL resolvers.
// Designed for Admin Dashboard operations (no auth mutations).
type Resolver struct {
	AuthService          *service.AuthService
	PolicyService        *service.PolicyService
	CompanyService       *service.CompanyService
	CompanyLookupService CompanyLookupUseCase
	UserRepo             user.Repository
	PolicyRepo           policy.RoleRepository
	CompanyRepo          company.Repository
	EventBus             event.Bus
}
