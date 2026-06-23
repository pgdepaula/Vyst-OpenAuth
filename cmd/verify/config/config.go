// Package config provides configuration for the verification CLI.
package config

import (
	"time"
)

// Config holds all configuration for the verification runner.
type Config struct {
	// BaseURL is the base URL of the API to verify.
	BaseURL string

	// GRPCURL is the gRPC server address.
	GRPCURL string

	// DatabaseURL is the database connection string for direct DB checks.
	DatabaseURL string

	// Timeout is the maximum time for individual requests.
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts for failed requests.
	MaxRetries int

	// RetryDelay is the base delay between retries (used with exponential backoff).
	RetryDelay time.Duration

	// Parallelism is the number of concurrent checks to run.
	Parallelism int

	// Verbose enables detailed logging.
	Verbose bool

	// Format controls output format: "terminal", "json", or "ci".
	Format string

	// SkipRLS skips Row Level Security verification.
	SkipRLS bool

	// SkipGRPC skips gRPC endpoint verification.
	SkipGRPC bool

	// SkipGraphQL skips GraphQL endpoint verification.
	SkipGraphQL bool
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		BaseURL:     "http://localhost:8982",
		GRPCURL:     "localhost:52151",
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/vyst_identity?sslmode=disable",
		Timeout:     10 * time.Second,
		MaxRetries:  3,
		RetryDelay:  500 * time.Millisecond,
		Parallelism: 4,
		Verbose:     false,
		Format:      "terminal",
		SkipRLS:     false,
		SkipGRPC:    false,
		SkipGraphQL: false,
	}
}
