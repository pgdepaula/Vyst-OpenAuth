package risk_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/domain/risk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Implementations for Testing
// ============================================================================

type mockRiskRule struct {
	name   string
	score  float64
	reason string
	err    error
}

func (m *mockRiskRule) Name() string {
	return m.name
}

func (m *mockRiskRule) Evaluate(ctx context.Context, userID uuid.UUID, ip string, userAgent string) (float64, string, error) {
	return m.score, m.reason, m.err
}

type mockLoginHistoryRepo struct {
	lastLogin *risk.LoginHistory
	err       error
}

func (m *mockLoginHistoryRepo) GetLastLogin(ctx context.Context, userID uuid.UUID) (*risk.LoginHistory, error) {
	return m.lastLogin, m.err
}

func (m *mockLoginHistoryRepo) Save(ctx context.Context, history *risk.LoginHistory) error {
	return nil
}

// ============================================================================
// StandardRiskEngine Tests
// ============================================================================

func TestStandardRiskEngine_Analyze_NoRules_ReturnsZero(t *testing.T) {
	engine := risk.NewStandardRiskEngine()
	ctx := context.Background()
	userID := uuid.New()

	score, reasons, err := engine.Analyze(ctx, userID, "192.168.1.1", "Mozilla/5.0")

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
	assert.Empty(t, reasons)
}

func TestStandardRiskEngine_Analyze_SingleRule_ReturnsRuleScore(t *testing.T) {
	rule := &mockRiskRule{
		name:   "Test Rule",
		score:  0.5,
		reason: "Test reason",
	}
	engine := risk.NewStandardRiskEngine(rule)
	ctx := context.Background()
	userID := uuid.New()

	score, reasons, err := engine.Analyze(ctx, userID, "192.168.1.1", "Mozilla/5.0")

	require.NoError(t, err)
	assert.Equal(t, 0.5, score)
	assert.Len(t, reasons, 1)
	assert.Contains(t, reasons[0], "Test Rule")
	assert.Contains(t, reasons[0], "Test reason")
}

func TestStandardRiskEngine_Analyze_MultipleRules_AggregatesScores(t *testing.T) {
	rule1 := &mockRiskRule{name: "Rule1", score: 0.3, reason: "Reason 1"}
	rule2 := &mockRiskRule{name: "Rule2", score: 0.4, reason: "Reason 2"}
	engine := risk.NewStandardRiskEngine(rule1, rule2)
	ctx := context.Background()
	userID := uuid.New()

	score, reasons, err := engine.Analyze(ctx, userID, "192.168.1.1", "Mozilla/5.0")

	require.NoError(t, err)
	assert.Equal(t, 0.7, score)
	assert.Len(t, reasons, 2)
}

func TestStandardRiskEngine_Analyze_CapsScoreAtOne(t *testing.T) {
	rule1 := &mockRiskRule{name: "Rule1", score: 0.6, reason: "Reason 1"}
	rule2 := &mockRiskRule{name: "Rule2", score: 0.7, reason: "Reason 2"}
	engine := risk.NewStandardRiskEngine(rule1, rule2)
	ctx := context.Background()
	userID := uuid.New()

	score, reasons, err := engine.Analyze(ctx, userID, "192.168.1.1", "Mozilla/5.0")

	require.NoError(t, err)
	assert.Equal(t, 1.0, score, "Score should be capped at 1.0")
	assert.Len(t, reasons, 2)
}

func TestStandardRiskEngine_Analyze_ZeroScoreRule_NotIncludedInReasons(t *testing.T) {
	rule1 := &mockRiskRule{name: "Clean Rule", score: 0.0, reason: ""}
	rule2 := &mockRiskRule{name: "Risk Rule", score: 0.5, reason: "Risk detected"}
	engine := risk.NewStandardRiskEngine(rule1, rule2)
	ctx := context.Background()
	userID := uuid.New()

	score, reasons, err := engine.Analyze(ctx, userID, "192.168.1.1", "Mozilla/5.0")

	require.NoError(t, err)
	assert.Equal(t, 0.5, score)
	assert.Len(t, reasons, 1, "Only rules with score > 0 should be in reasons")
	assert.Contains(t, reasons[0], "Risk Rule")
}

func TestStandardRiskEngine_Analyze_RuleError_ContinuesWithOtherRules(t *testing.T) {
	errorRule := &mockRiskRule{name: "Error Rule", err: assert.AnError}
	goodRule := &mockRiskRule{name: "Good Rule", score: 0.5, reason: "Good reason"}
	engine := risk.NewStandardRiskEngine(errorRule, goodRule)
	ctx := context.Background()
	userID := uuid.New()

	score, reasons, err := engine.Analyze(ctx, userID, "192.168.1.1", "Mozilla/5.0")

	require.NoError(t, err, "Engine should not fail if individual rule fails")
	assert.Equal(t, 0.5, score)
	assert.Len(t, reasons, 1)
}

// ============================================================================
// ImpossibleTravelRule Tests
// ============================================================================

func TestImpossibleTravelRule_Name(t *testing.T) {
	repo := &mockLoginHistoryRepo{}
	rule := risk.NewImpossibleTravelRule(repo)

	assert.Equal(t, "Impossible Travel", rule.Name())
}

func TestImpossibleTravelRule_Evaluate_NoHistory_ReturnsZero(t *testing.T) {
	repo := &mockLoginHistoryRepo{lastLogin: nil}
	rule := risk.NewImpossibleTravelRule(repo)
	ctx := context.Background()
	userID := uuid.New()

	score, reason, err := rule.Evaluate(ctx, userID, "192.168.1.1", "Mozilla/5.0")

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
	assert.Empty(t, reason)
}

func TestImpossibleTravelRule_Evaluate_SameIP_ReturnsZero(t *testing.T) {
	repo := &mockLoginHistoryRepo{
		lastLogin: &risk.LoginHistory{
			UserID:    uuid.New(),
			IPAddress: "192.168.1.1",
			LoginAt:   time.Now().Add(-1 * time.Second),
		},
	}
	rule := risk.NewImpossibleTravelRule(repo)
	ctx := context.Background()
	userID := uuid.New()

	score, reason, err := rule.Evaluate(ctx, userID, "192.168.1.1", "Mozilla/5.0")

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
	assert.Empty(t, reason)
}

func TestImpossibleTravelRule_Evaluate_DifferentIPWithinThreshold_ReturnsMaxScore(t *testing.T) {
	repo := &mockLoginHistoryRepo{
		lastLogin: &risk.LoginHistory{
			UserID:    uuid.New(),
			IPAddress: "10.0.0.1",
			LoginAt:   time.Now().Add(-1 * time.Second),
		},
	}
	rule := risk.NewImpossibleTravelRule(repo)
	ctx := context.Background()
	userID := uuid.New()

	score, reason, err := rule.Evaluate(ctx, userID, "192.168.1.1", "Mozilla/5.0")

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
	assert.Contains(t, reason, "Impossible travel")
}

func TestImpossibleTravelRule_Evaluate_DifferentIPAfterThreshold_ReturnsZero(t *testing.T) {
	repo := &mockLoginHistoryRepo{
		lastLogin: &risk.LoginHistory{
			UserID:    uuid.New(),
			IPAddress: "10.0.0.1",
			LoginAt:   time.Now().Add(-10 * time.Second),
		},
	}
	rule := risk.NewImpossibleTravelRule(repo)
	ctx := context.Background()
	userID := uuid.New()

	score, reason, err := rule.Evaluate(ctx, userID, "192.168.1.1", "Mozilla/5.0")

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
	assert.Empty(t, reason)
}

// ============================================================================
// RiskRule Interface Documentation Tests
// ============================================================================

func TestRiskRule_Interface_RequiresNameAndEvaluate(t *testing.T) {
	var rule risk.RiskRule = &mockRiskRule{
		name:   "Test",
		score:  0.5,
		reason: "Test",
	}

	name := rule.Name()
	assert.NotEmpty(t, name)

	score, reason, err := rule.Evaluate(context.Background(), uuid.New(), "", "")
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
	assert.NotNil(t, reason)
	_ = err
}

// ============================================================================
// Table-Driven Risk Score Tests
// ============================================================================

func TestStandardRiskEngine_Analyze_ScoreAggregation_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		ruleScores    []float64
		expectedScore float64
	}{
		{"no rules", []float64{}, 0.0},
		{"single rule 0.0", []float64{0.0}, 0.0},
		{"single rule 0.5", []float64{0.5}, 0.5},
		{"single rule 1.0", []float64{1.0}, 1.0},
		{"two rules sum < 1", []float64{0.3, 0.4}, 0.7},
		{"two rules sum = 1", []float64{0.5, 0.5}, 1.0},
		{"two rules sum > 1", []float64{0.6, 0.6}, 1.0},
		{"three rules sum > 1", []float64{0.5, 0.3, 0.4}, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := make([]risk.RiskRule, len(tt.ruleScores))
			for i, score := range tt.ruleScores {
				rules[i] = &mockRiskRule{
					name:   "Rule",
					score:  score,
					reason: "Reason",
				}
			}
			engine := risk.NewStandardRiskEngine(rules...)
			ctx := context.Background()

			actualScore, _, err := engine.Analyze(ctx, uuid.New(), "", "")

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedScore, actualScore, 0.001)
		})
	}
}
