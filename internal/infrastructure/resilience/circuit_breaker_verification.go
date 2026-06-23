package resilience

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/sony/gobreaker"
)

// CircuitBreakerDocumentVerificationPort is a decorator that adds circuit breaking.
type CircuitBreakerDocumentVerificationPort struct {
	delegate ports.DocumentVerificationPort
	cb       *gobreaker.CircuitBreaker
	logger   ports.Logger
}

// NewCircuitBreakerDocumentVerificationPort creates a new circuit breaker decorator.
func NewCircuitBreakerDocumentVerificationPort(delegate ports.DocumentVerificationPort, name string, logger ports.Logger) *CircuitBreakerDocumentVerificationPort {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warn("Circuit Breaker State Changed", "name", name, "from", from.String(), "to", to.String())
		},
	}

	return &CircuitBreakerDocumentVerificationPort{
		delegate: delegate,
		cb:       gobreaker.NewCircuitBreaker(settings),
		logger:   logger,
	}
}

func (p *CircuitBreakerDocumentVerificationPort) VerifyCPF(ctx context.Context, cpf string) (*ports.DocumentVerificationResult, error) {
	result, err := p.cb.Execute(func() (interface{}, error) {
		return p.delegate.VerifyCPF(ctx, cpf)
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			p.logger.Warn("Circuit Breaker Open: Failing Fast", "cpf", "masked")
			return nil, fmt.Errorf("service unavailable (circuit open): %w", err)
		}
		return nil, err
	}

	return result.(*ports.DocumentVerificationResult), nil
}
