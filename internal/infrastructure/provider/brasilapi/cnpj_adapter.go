package brasilapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
)

// BrasilAPIAdapter implements CompanyDataPort using the BrasilAPI public endpoints
type BrasilAPIAdapter struct {
	client  *http.Client
	baseURL string
}

// NewBrasilAPIAdapter creates a new adapter.
func NewBrasilAPIAdapter(client *http.Client, baseURL string) *BrasilAPIAdapter {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	if baseURL == "" {
		baseURL = "https://brasilapi.com.br/api/cnpj/v1"
	}
	return &BrasilAPIAdapter{
		client:  client,
		baseURL: baseURL,
	}
}

// Name returns the provider's name.
func (a *BrasilAPIAdapter) Name() string {
	return "BrasilAPI"
}

// GetByCNPJ fetches company data utilizing BrasilAPI.
func (a *BrasilAPIAdapter) GetByCNPJ(ctx context.Context, cnpj string) (*company.CompanyInfo, error) {
	normalized := company.NormalizeCNPJ(cnpj)
	url := fmt.Sprintf("%s/%s", a.baseURL, normalized)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("brasilapi: failed to create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("brasilapi: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, company.ErrCompanyInfoNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brasilapi: unexpected status code %d", resp.StatusCode)
	}

	var data struct {
		CNPJ              string `json:"cnpj"`
		RazaoSocial       string `json:"razao_social"`
		NomeFantasia      string `json:"nome_fantasia"`
		DescricaoSituacao string `json:"descricao_situacao_cadastral"`
		DataInicioAtv     string `json:"data_inicio_atividade"`
		CNAEPrincipal     int    `json:"cnae_fiscal"`
		DescricaoCNAE     string `json:"cnae_fiscal_descricao"`
		NaturezaJuridica  string `json:"natureza_juridica"`

		// Address
		Logradouro  string `json:"logradouro"`
		Numero      string `json:"numero"`
		Complemento string `json:"complemento"`
		Bairro      string `json:"bairro"`
		Municipio   string `json:"municipio"`
		UF          string `json:"uf"`
		CEP         string `json:"cep"`

		DDDTelefone1 string `json:"ddd_telefone_1"`
		DDDTelefone2 string `json:"ddd_telefone_2"`
		Email        string `json:"email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("brasilapi: failed to decode response: %w", err)
	}

	// Parse date
	var parsedDate time.Time
	if data.DataInicioAtv != "" {
		parsedDate, err = time.Parse("2006-01-02", data.DataInicioAtv)
		if err != nil {
			parsedDate = time.Time{}
		}
	}

	// Map situation
	situacao := mapCadastralSituation(data.DescricaoSituacao)
	telefones := collectNonEmpty(data.DDDTelefone1, data.DDDTelefone2)
	emails := collectNonEmpty(data.Email)

	info := &company.CompanyInfo{
		CNPJ:             normalized,
		RazaoSocial:      data.RazaoSocial,
		NomeFantasia:     data.NomeFantasia,
		Situacao:         situacao,
		NaturezaJuridica: data.NaturezaJuridica,
		DataAbertura:     parsedDate,
		Endereco: company.Address{
			Logradouro:  data.Logradouro,
			Numero:      data.Numero,
			Complemento: data.Complemento,
			Bairro:      data.Bairro,
			Cidade:      data.Municipio,
			UF:          data.UF,
			CEP:         data.CEP,
		},
		Telefones:     telefones,
		Emails:        emails,
		CNAEPrincipal: fmt.Sprintf("%d", data.CNAEPrincipal),
		LastFetchedAt: time.Now(),
	}

	return info, nil
}

func mapCadastralSituation(value string) company.CadastralSituation {
	situacao := company.CadastralSituation(value)
	if situacao.IsValid() {
		return situacao
	}
	switch value {
	case "ATIVA":
		return company.SituationActive
	case "BAIXADA":
		return company.SituationLowered
	case "INAPTA":
		return company.SituationInapt
	case "SUSPENSA":
		return company.SituationSuspended
	case "NULA":
		return company.SituationNull
	default:
		return company.SituationSuspended
	}
}

func collectNonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

// SearchByName is not supported natively by BrasilAPI free tier.
func (a *BrasilAPIAdapter) SearchByName(ctx context.Context, query string, limit int) ([]*company.CompanyInfo, error) {
	// Not supported. Returning ErrSearchNotSupported allows fallback to work seamlessly.
	return nil, company.ErrSearchNotSupported
}
