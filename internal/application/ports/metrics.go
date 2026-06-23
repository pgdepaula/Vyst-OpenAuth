package ports

// MetricsService defines the contract for application-level metrics.
type MetricsService interface {
	// IncrementCounter increments a specific metric by 1.
	IncrementCounter(metricName string, tags map[string]string)

	// RecordDuration records the time an operation took.
	RecordDuration(metricName string, durationSeconds float64, tags map[string]string)
}
