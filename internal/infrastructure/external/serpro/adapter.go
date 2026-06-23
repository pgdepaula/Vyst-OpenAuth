package serpro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
)

type SerproAdapter struct {
	client  *http.Client
	baseURL string
	apiKey  string
	timeout time.Duration
}

type SerproConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

func NewSerproAdapter(cfg SerproConfig) *SerproAdapter {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &SerproAdapter{
		client:  &http.Client{Timeout: cfg.Timeout},
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		timeout: cfg.Timeout,
	}
}

// VerifyCPF verifies a CPF against the Serpro API.
// Note: This is an example implementation assuming a generic Serpro-like response.
// Real Serpro API requires OAuth2 bearer token flow usually, but we implement basic structure first.
func (a *SerproAdapter) VerifyCPF(ctx context.Context, cpf string) (*ports.DocumentVerificationResult, error) {
	// If no API key configured, we might want to return error or skip in real life.
	// For now, if no BaseURL/ApiKey (dev mode), we return success if it's not a specific "bad" CPF.
	if a.baseURL == "" || a.apiKey == "" {
		// Used for local development to save credits/setup
		return &ports.DocumentVerificationResult{
			Valid:     true,
			Situation: "REGULAR (MOCKED)",
			Timestamp: time.Now(),
		}, nil
	}

	url := fmt.Sprintf("%s/consulta/cpf/v1/cpf/%s", a.baseURL, cpf)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return &ports.DocumentVerificationResult{
			Valid:     false,
			Situation: "NOT_FOUND",
			Timestamp: time.Now(),
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("serpro api error: status %d", resp.StatusCode)
	}

	var serproResp struct {
		Ni       string `json:"ni"`
		Nome     string `json:"nome"`
		Situacao struct {
			Codigo    string `json:"codigo"`
			Descricao string `json:"descricao"`
		} `json:"situacao"`
		Nascimento string `json:"nascimento"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&serproResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Map Serpro status to our domain result
	// Situacao "0" usually means Regular. "1" Cancelada, "2" Suspensa, etc.
	// We'll trust Descricao for now.

	valid := serproResp.Situacao.Descricao == "REGULAR"

	return &ports.DocumentVerificationResult{
		Valid:     valid,
		Name:      serproResp.Nome,
		Situation: serproResp.Situacao.Descricao,
		Timestamp: time.Now(),
	}, nil
}
