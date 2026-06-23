package risk

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type StandardRiskEngine struct {
	rules []RiskRule
}

func NewStandardRiskEngine(rules ...RiskRule) *StandardRiskEngine {
	return &StandardRiskEngine{rules: rules}
}

func (e *StandardRiskEngine) Analyze(ctx context.Context, userID uuid.UUID, ip string, userAgent string) (float64, []string, error) {
	var totalScore float64
	var reasons []string

	for _, rule := range e.rules {
		score, reason, err := rule.Evaluate(ctx, userID, ip, userAgent)
		if err != nil {
			// Log error but continue? Or fail open/closed?
			// For now, we log and continue, assuming 0 risk for that rule.
			// In a real system, we might want to fail closed for critical rules.
			slog.Error("Error evaluating rule", "rule", rule.Name(), "error", err)
			continue
		}

		if score > 0 {
			totalScore += score
			reasons = append(reasons, fmt.Sprintf("%s: %s (Score: %.2f)", rule.Name(), reason, score))
		}
	}

	// Cap score at 1.0
	if totalScore > 1.0 {
		totalScore = 1.0
	}

	return totalScore, reasons, nil
}
