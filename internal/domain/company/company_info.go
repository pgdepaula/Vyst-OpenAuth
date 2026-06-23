// Package company contains the Company domain entity and repository interface.
// This is the core domain layer - no external dependencies allowed.
package company

import (
	"context"
	"errors"
	"time"
)

// Domain errors for company info fetching operations.
var (
	// ErrCompanyInfoNotFound is returned when company info is not found in cache or external API.
	ErrCompanyInfoNotFound = errors.New("company info not found")

	// ErrCompanyInactive is returned when an operation requires an active company but it's not.
	ErrCompanyInactive = errors.New("company is inactive")

	// ErrSearchNotSupported is returned when a provider does not support searching by name.
	ErrSearchNotSupported = errors.New("text search is not supported by this provider")
)

// CadastralSituation represents the registration status of a company according to Receita Federal.
type CadastralSituation string

const (
	// SituationActive indicates the company is active and fully operational.
	SituationActive CadastralSituation = "ATIVA"

	// SituationSuspended indicates the company has been suspended.
	SituationSuspended CadastralSituation = "SUSPENSA"

	// SituationInapt indicates the company is unfit/irregular.
	SituationInapt CadastralSituation = "INAPTA"

	// SituationLowered indicates the company has been closed down (baixada).
	SituationLowered CadastralSituation = "BAIXADA"

	// SituationNull indicates the registration is null.
	SituationNull CadastralSituation = "NULA"
)

// IsValid checks if the cadastral situation is a known value.
func (s CadastralSituation) IsValid() bool {
	return s == SituationActive || s == SituationSuspended || s == SituationInapt || s == SituationLowered || s == SituationNull
}

// CompanyInfo represents the enriched company data fetched from an external source (like INPI/Receita Federal).
// It acts as an aggregate for read-only caching and search purposes.
type CompanyInfo struct {
	CNPJ             string             // Unique, validated (14 digits)
	RazaoSocial      string             // Legal name (razão social)
	NomeFantasia     string             // Trade name (nome fantasia)
	Situacao         CadastralSituation // Registration status (ATIVA, BAIXADA, etc)
	NaturezaJuridica string             // Legal nature code/description
	DataAbertura     time.Time          // Foundation date
	Endereco         Address            // Company address
	Telefones        []string           // Contact phone numbers
	Emails           []string           // Contact email addresses
	CNAEPrincipal    string             // Principal economic activity code
	LastFetchedAt    time.Time          // Timestamp when the data was last fetched from the external provider
}

// IsActive returns true if the company is in an active (ATIVA) cadastral situation.
func (ci *CompanyInfo) IsActive() bool {
	return ci.Situacao == SituationActive
}

// CheckEligibility checks if the company info allows it to be registered/used.
func (ci *CompanyInfo) CheckEligibility() error {
	if !ci.IsActive() {
		return ErrCompanyInactive
	}
	return nil
}

// CompanyInfoRepository defines the contract for persisting enriched company data (cache).
// Implementations live in the infrastructure layer.
type CompanyInfoRepository interface {
	// Save persists company info to the cache storage.
	Save(ctx context.Context, info *CompanyInfo) error

	// GetByCNPJ retrieves company info by CNPJ from the cache.
	// Returns ErrCompanyInfoNotFound if not present or expired (depending on cache eviction logic).
	GetByCNPJ(ctx context.Context, cnpj string) (*CompanyInfo, error)

	// SearchByName performs a text search on RazaoSocial or NomeFantasia.
	SearchByName(ctx context.Context, query string, limit int) ([]*CompanyInfo, error)
}
