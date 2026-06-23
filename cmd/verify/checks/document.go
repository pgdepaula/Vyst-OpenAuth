package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pgdepaula/vyst-openauth/cmd/verify/config"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/runner"
)

// DocumentChecks returns document verification checks.
func DocumentChecks(authCtx *AuthContext) []runner.Check {
	return []runner.Check{
		{
			Name:  "POST /api/v1/documents/validate-cpf (valid)",
			Group: "Document API",
			Fn:    makeValidateCPFCheck(authCtx, "52998224725", true), // A valid CPF example (generated)
		},
		{
			Name:  "POST /api/v1/documents/validate-cpf (invalid)",
			Group: "Document API",
			Fn:    makeValidateCPFCheck(authCtx, "00000000000", false),
		},
	}
}

func makeValidateCPFCheck(authCtx *AuthContext, cpf string, expectValid bool) func(context.Context, *config.Config) (*runner.CheckResult, error) {
	return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
		start := time.Now()

		if authCtx.Token == "" {
			return &runner.CheckResult{
				Name:     fmt.Sprintf("Validate CPF %s", cpf),
				Group:    "Document API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    "no auth token available",
			}, nil
		}

		payload := map[string]string{
			"cpf": cpf,
		}

		resp, body, err := doPost(ctx, cfg, "/api/v1/documents/validate-cpf", authCtx.Token, payload)
		if err != nil {
			return &runner.CheckResult{
				Name:     fmt.Sprintf("Validate CPF %s", cpf),
				Group:    "Document API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    err.Error(),
			}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return &runner.CheckResult{
				Name:       fmt.Sprintf("Validate CPF %s", cpf),
				Group:      "Document API",
				Passed:     false,
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
				Error:      fmt.Sprintf("expected 200, got %d: %s", resp.StatusCode, truncate(string(body), 100)),
			}, nil
		}

		var result struct {
			Valid     bool   `json:"valid"`
			Formatted string `json:"formatted"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return &runner.CheckResult{
				Name:     fmt.Sprintf("Validate CPF %s", cpf),
				Group:    "Document API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    fmt.Sprintf("failed to parse response: %v", err),
			}, nil
		}

		if result.Valid != expectValid {
			return &runner.CheckResult{
				Name:     fmt.Sprintf("Validate CPF %s", cpf),
				Group:    "Document API",
				Passed:   false,
				Duration: time.Since(start),
				Error:    fmt.Sprintf("expected valid=%v, got %v", expectValid, result.Valid),
			}, nil
		}

		return &runner.CheckResult{
			Name:       fmt.Sprintf("Validate CPF %s", cpf),
			Group:      "Document API",
			Passed:     true,
			Duration:   time.Since(start),
			StatusCode: resp.StatusCode,
		}, nil
	}
}
