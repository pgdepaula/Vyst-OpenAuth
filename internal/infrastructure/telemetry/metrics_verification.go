package telemetry

import (
	"context"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricDocumentVerificationPort is a decorator that records metrics for document verifications.
type MetricDocumentVerificationPort struct {
	delegate   ports.DocumentVerificationPort
	attemptCtr metric.Int64Counter
	duration   metric.Float64Histogram
	errorCtr   metric.Int64Counter
}

// NewMetricDocumentVerificationPort creates a new metric decorator.
func NewMetricDocumentVerificationPort(delegate ports.DocumentVerificationPort) (*MetricDocumentVerificationPort, error) {
	meter := otel.GetMeterProvider().Meter("vyst-identity")

	attemptCtr, err := meter.Int64Counter("document_verification_attempts_total", metric.WithDescription("Total number of document verification attempts"))
	if err != nil {
		return nil, err
	}

	duration, err := meter.Float64Histogram("document_verification_duration_seconds", metric.WithDescription("Duration of document verification"))
	if err != nil {
		return nil, err
	}

	errorCtr, err := meter.Int64Counter("document_verification_errors_total", metric.WithDescription("Total number of document verification errors"))
	if err != nil {
		return nil, err
	}

	return &MetricDocumentVerificationPort{
		delegate:   delegate,
		attemptCtr: attemptCtr,
		duration:   duration,
		errorCtr:   errorCtr,
	}, nil
}

func (m *MetricDocumentVerificationPort) VerifyCPF(ctx context.Context, cpf string) (*ports.DocumentVerificationResult, error) {
	start := time.Now()

	// Record attempt
	m.attemptCtr.Add(ctx, 1, metric.WithAttributes(attribute.String("type", "CPF")))

	result, err := m.delegate.VerifyCPF(ctx, cpf)

	duration := time.Since(start).Seconds()

	// Record duration
	m.duration.Record(ctx, duration, metric.WithAttributes(attribute.String("type", "CPF")))

	if err != nil {
		// Record error
		m.errorCtr.Add(ctx, 1, metric.WithAttributes(attribute.String("type", "CPF")))
		return nil, err
	}

	return result, nil
}
