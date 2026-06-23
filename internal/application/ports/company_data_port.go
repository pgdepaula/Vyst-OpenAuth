package ports

import (
	"context"

	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
)

// CompanyDataPort abstracts external providers of company information (Receita Federal, INPI, etc.).
// By defining it as a port, the application layer doesn't depend on HTTP clients or provider-specific details.
type CompanyDataPort interface {
	// Name returns the provider's name (e.g., "BrasilAPI", "ReceitaWS").
	Name() string

	// GetByCNPJ fetches company data from an external source using the CNPJ.
	// Returns company.ErrCompanyInfoNotFound if the CNPJ doesn't exist in the provider.
	GetByCNPJ(ctx context.Context, cnpj string) (*company.CompanyInfo, error)

	// SearchByName searches for companies matching the given query (RazaoSocial or NomeFantasia).
	// Limit controls the maximum number of results from the external provider.
	// Note: Not all providers support text search. In that case, they may return ErrMethodNotAllowed.
	SearchByName(ctx context.Context, query string, limit int) ([]*company.CompanyInfo, error)
}
