package checks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pgdepaula/vyst-openauth/cmd/verify/config"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/runner"
)

// GraphQLChecks returns GraphQL API verification checks.
func GraphQLChecks(authCtx *AuthContext) []runner.Check {
	return []runner.Check{
		{
			Name:  "GraphQL Introspection",
			Group: "GraphQL",
			Fn:    makeGraphQLIntrospectionCheck(),
		},
		{
			Name:  "GraphQL me Query",
			Group: "GraphQL",
			Fn:    makeGraphQLMeCheck(authCtx),
		},
		{
			Name:  "GraphQL Companies Query",
			Group: "GraphQL",
			Fn:    makeGraphQLCompaniesCheck(authCtx),
		},
		{
			Name:  "GraphQL UpdateCompanyStatus Mutation",
			Group: "GraphQL",
			Fn:    makeGraphQLUpdateCompanyStatusCheck(authCtx),
		},
	}
}

func makeGraphQLIntrospectionCheck() func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		query := `{"query": "{ __schema { types { name } } }"}`

		resp, body, err := doGraphQL(ctx, cfg, query, "")
		if err != nil {
			return &runner.CheckResult{
				Name:     "GraphQL Introspection",
				Group:    "GraphQL",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "GraphQL Introspection",
				Group:      "GraphQL",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "GraphQL Introspection",
			Group:      "GraphQL",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeGraphQLMeCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "GraphQL me Query",
				Group:    "GraphQL",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		query := `{"query": "{ me { id email } }"}`

		resp, body, err := doGraphQL(ctx, cfg, query, authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     "GraphQL me Query",
				Group:    "GraphQL",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "GraphQL me Query",
				Group:      "GraphQL",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "GraphQL me Query",
			Group:      "GraphQL",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeGraphQLCompaniesCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "GraphQL Companies Query",
				Group:    "GraphQL",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		query := `{"query": "{ companies(page: 1, limit: 10) { items { id cnpj razao_social } count } }"}`

		resp, body, err := doGraphQL(ctx, cfg, query, authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     "GraphQL Companies Query",
				Group:    "GraphQL",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "GraphQL Companies Query",
				Group:      "GraphQL",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		// Parse response to get CompanyID for subsequent checks
		var companiesResp struct {
			Data struct {
				Companies struct {
					Items []struct {
						ID string `json:"id"`
					} `json:"items"`
				} `json:"companies"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &companiesResp); err != nil {
			return &runner.CheckResult{
				Name:     "GraphQL Companies Query",
				Group:    "GraphQL",
				Passed:   false,
				Duration: time.Since(start),
				Error:    fmt.Sprintf("failed to parse response: %v", err),
			}, nil
		}

		if len(companiesResp.Data.Companies.Items) > 0 {
			authCtx.CompanyID = companiesResp.Data.Companies.Items[0].ID
		}

		return &runner.CheckResult{
			Name:       "GraphQL Companies Query",
			Group:      "GraphQL",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func makeGraphQLUpdateCompanyStatusCheck(authCtx *AuthContext) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     "GraphQL UpdateCompanyStatus Mutation",
				Group:    "GraphQL",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		if authCtx.CompanyID == "" {
			return &runner.CheckResult{
				Name:     "GraphQL UpdateCompanyStatus Mutation",
				Group:    "GraphQL",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no company_id available from auth context",
			}, nil
		}

		// Mutation to suspend the company created during registration
		query := fmt.Sprintf(`{"query": "mutation { updateCompanyStatus(id: \"%s\", status: \"SUSPENDED\") { id status } }"}`, authCtx.CompanyID)

		resp, body, err := doGraphQL(ctx, cfg, query, authCtx.Token)
		if err != nil {
			return &runner.CheckResult{
				Name:     "GraphQL UpdateCompanyStatus Mutation",
				Group:    "GraphQL",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       "GraphQL UpdateCompanyStatus Mutation",
				Group:      "GraphQL",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		// Simple verify that no GraphQL errors returned (body shouldn't contain "errors")
		if bytes.Contains(body, []byte("errors")) {
			return &runner.CheckResult{
				Name:     "GraphQL UpdateCompanyStatus Mutation",
				Group:    "GraphQL",
				Passed:   false,
				Duration: time.Since(start),
				Error:    fmt.Sprintf("GraphQL error: %s", truncate(string(body), 200)),
			}, nil
		}

		return &runner.CheckResult{
			Name:       "GraphQL UpdateCompanyStatus Mutation",
			Group:      "GraphQL",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}

func doGraphQL(ctx context.Context, cfg *config.Config, query, token string) (*http.Response, []byte, error) {
	client := &http.Client{Timeout: cfg.Timeout}

	// Correct path is /query based on server.go
	req, err := http.NewRequestWithContext(ctx, "POST", cfg.BaseURL+"/query", bytes.NewBuffer([]byte(query)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp, body, nil
}
