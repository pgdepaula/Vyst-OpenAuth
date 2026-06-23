package risk

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// LoginHistory represents a record of a user's login.
type LoginHistory struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	IPAddress string
	UserAgent string
	LoginAt   time.Time
	Latitude  float64
	Longitude float64
}

// LoginHistoryRepository defines the interface for persisting login history.
type LoginHistoryRepository interface {
	Save(ctx context.Context, history *LoginHistory) error
	GetLastLogin(ctx context.Context, userID uuid.UUID) (*LoginHistory, error)
}

// RiskRule defines the interface for a risk analysis rule.
type RiskRule interface {
	// Evaluate analyzes the context and returns a risk score (0.0 - 1.0) and a reason.
	Evaluate(ctx context.Context, userID uuid.UUID, ip string, userAgent string) (float64, string, error)
	Name() string
}

// RiskEngine orchestrates the risk analysis process.
type RiskEngine interface {
	Analyze(ctx context.Context, userID uuid.UUID, ip string, userAgent string) (float64, []string, error)
}
