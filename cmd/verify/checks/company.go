package checks

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pgdepaula/vyst-openauth/cmd/verify/config"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/runner"
)

// CompanyChecks returns company-related endpoint verification checks.
// These checks verify the company CRUD and user management functionality.
func CompanyChecks(authCtx *AuthContext) []runner.Check {
	return []runner.Check{
		// Create Company
		{
			Name:  "POST /api/v1/companies (create)",
			Group: "Company API",
			Fn:    makeCreateCompanyCheck(authCtx),
		},
		// List Companies
		{
			Name:  "GET /api/v1/companies (list)",
			Group: "Company API",
			Fn:    makeListCompaniesCheck(authCtx),
		},
		// Get Company by ID
		{
			Name:  "GET /api/v1/companies/{id}",
			Group: "Company API",
			Fn:    makeGetCompanyCheck(authCtx),
		},
		// Switch Company Context
		{
			Name:  "POST /api/v1/auth/switch-company",
			Group: "Company API",
			Fn:    makeSwitchCompanyCheck(authCtx),
		},
		// Clear Company Context
		{
			Name:  "DELETE /api/v1/auth/company-context",
			Group: "Company API",
			Fn:    makeClearCompanyContextCheck(authCtx),
		},
		// Create Company with Invalid CNPJ
		{
			Name:  "POST /api/v1/companies (invalid CNPJ)",
			Group: "Company API",
			Fn:    makeCreateCompanyInvalidCNPJCheck(authCtx),
		},
		// Lookup Company by CNPJ
		{
			Name:  "GET /api/v1/companies/lookup (valid CNPJ)",
			Group: "Company API",
			Fn:    makeLookupCompanyCheck(authCtx),
		},
	}
}

func makeCreateCompanyCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()
		checkName := "POST /api/v1/companies (create)"

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		// Use a valid, dynamically generated CNPJ
		payload := map[string]interface{}{
			"cnpj":                generateCNPJ(),
			"razao_social":        fmt.Sprintf("Test Company %d", time.Now().UnixNano()),
			"nome_fantasia":       "Verify Test Co",
			"representante_legal": "John Doe",
			"endereco": map[string]string{
				"logradouro": "Rua Test",
				"numero":     "123",
				"cidade":     "São Paulo",
				"uf":         "SP",
				"cep":        "01310100",
			},
		}

		resp, body, err := doPost(ctx, cfg, "/api/v1/companies", authCtx.Token, payload)
		if err != nil {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		// Accept 201 (Created) or 409 (Conflict if CNPJ already exists from previous run)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
			return &runner.CheckResult{
				Name:       checkName,
				Group:      "Company API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 201 or 409, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		// If created, store the company ID for subsequent checks
		if resp.StatusCode == http.StatusCreated {
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err == nil {
				if id, ok := result["id"].(string); ok {
					authCtx.CompanyID = id
				}
			}
		}

		return &runner.CheckResult{
			Name:       checkName,
			Group:      "Company API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeListCompaniesCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()
		checkName := "GET /api/v1/companies (list)"

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		resp, body, err := doGet(ctx, cfg, "/api/v1/companies", authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       checkName,
				Group:      "Company API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		// Parse response to get company ID if not already set
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			if companies, ok := result["companies"].([]interface{}); ok && len(companies) > 0 {
				if first, ok := companies[0].(map[string]interface{}); ok {
					if company, ok := first["company"].(map[string]interface{}); ok {
						if id, ok := company["id"].(string); ok && authCtx.CompanyID == "" {
							authCtx.CompanyID = id
						}
					}
				}
			}
		}

		return &runner.CheckResult{
			Name:       checkName,
			Group:      "Company API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeGetCompanyCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()
		checkName := "GET /api/v1/companies/{id}"

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		if authCtx.CompanyID == "" {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no company ID available (create company first)",
			}, nil
		}

		resp, body, err := doGet(ctx, cfg, "/api/v1/companies/"+authCtx.CompanyID, authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       checkName,
				Group:      "Company API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       checkName,
			Group:      "Company API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeSwitchCompanyCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()
		checkName := "POST /api/v1/auth/switch-company"

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		if authCtx.CompanyID == "" {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no company ID available (create company first)",
			}, nil
		}

		payload := map[string]string{
			"company_id": authCtx.CompanyID,
		}

		resp, body, err := doPost(ctx, cfg, "/api/v1/auth/switch-company", authCtx.Token, payload)
		if err != nil {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       checkName,
				Group:      "Company API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       checkName,
			Group:      "Company API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeClearCompanyContextCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()
		checkName := "DELETE /api/v1/auth/company-context"

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		resp, body, err := doDelete(ctx, cfg, "/api/v1/auth/company-context", authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       checkName,
				Group:      "Company API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       checkName,
			Group:      "Company API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeCreateCompanyInvalidCNPJCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()
		checkName := "POST /api/v1/companies (invalid CNPJ)"

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		// Use an invalid CNPJ (wrong check digit)
		payload := map[string]interface{}{
			"cnpj":         "12345678901234",
			"razao_social": "Invalid CNPJ Test",
		}

		resp, body, err := doPost(ctx, cfg, "/api/v1/companies", authCtx.Token, payload)
		if err != nil {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		// Should return 400 Bad Request for invalid CNPJ
		if resp.StatusCode != http.StatusBadRequest {
			return &runner.CheckResult{
				Name:       checkName,
				Group:      "Company API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 400 for invalid CNPJ, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       checkName,
			Group:      "Company API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeLookupCompanyCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()
		checkName := "GET /api/v1/companies/lookup"

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		// Use a known valid testing CNPJ or random one. We'll use a valid dummy CNPJ or one that would typically 404 cleanly
		// BrasilAPI might not find random valid CNPJs, but the endpoint should at least return 200 with empty items or 404. Let's lookup a common one or check structure.
		// Using generic format
		targetCNPJ := "00.000.000/0001-91" // Banco do Brasil as example

		resp, body, err := doGet(ctx, cfg, "/api/v1/companies/lookup?q="+targetCNPJ, authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     checkName,
				Group:    "Company API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       checkName,
				Group:      "Company API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return &runner.CheckResult{
				Name:       checkName,
				Group:      "Company API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      "invalid json response string",
			}, nil
		}

		if _, ok := result["items"].([]interface{}); !ok {
			return &runner.CheckResult{
				Name:       checkName,
				Group:      "Company API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      "missing 'items' array in response",
			}, nil
		}

		return &runner.CheckResult{
			Name:       checkName,
			Group:      "Company API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func generateCNPJ() string {
	// Generate first 8 digits
	numbers := make([]int, 0, 14)
	for i := 0; i < 8; i++ {
		numbers = append(numbers, randomDigit())
	}
	// Append 0001
	numbers = append(numbers, 0, 0, 0, 1)

	// Calculate first digit
	d1 := calculateCheckDigit(numbers, 12)
	numbers = append(numbers, d1)

	// Calculate second digit
	d2 := calculateCheckDigit(numbers, 13)
	numbers = append(numbers, d2)

	var sb strings.Builder
	for _, n := range numbers {
		sb.WriteString(fmt.Sprintf("%d", n))
	}
	return sb.String()
}

func randomDigit() int {
	var b [1]byte
	if _, err := crand.Read(b[:]); err != nil {
		return int(time.Now().UnixNano() % 10)
	}
	return int(b[0] % 10)
}

func calculateCheckDigit(numbers []int, position int) int {
	var weights []int
	if position == 12 {
		weights = []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	} else {
		weights = []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	}

	sum := 0
	for i := 0; i < position; i++ {
		sum += numbers[i] * weights[i]
	}

	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}
