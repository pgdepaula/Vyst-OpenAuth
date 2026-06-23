// Package main provides the entry point for the Vyst Identity verification CLI.
//
// This CLI runs comprehensive system verification checks including:
// - Health/Ready endpoints
// - Authentication flow (register, login, me, 2FA)
// - REST API endpoints (roles, tenants, API keys)
// - GraphQL API operations
//
// Usage:
//
//	go run ./cmd/verify/...                    # Run with defaults
//	go run ./cmd/verify/... --url=http://...   # Custom API URL
//	go run ./cmd/verify/... --format=json      # JSON output
//	go run ./cmd/verify/... --verbose          # Verbose output
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pgdepaula/vyst-openauth/cmd/verify/checks"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/config"
	"github.com/pgdepaula/vyst-openauth/cmd/verify/runner"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	cfg := config.DefaultConfig()

	// Parse flags
	flag.StringVar(&cfg.BaseURL, "url", cfg.BaseURL, "Base URL of the API to verify")
	flag.StringVar(&cfg.GRPCURL, "grpc-url", cfg.GRPCURL, "gRPC server address")
	flag.StringVar(&cfg.DatabaseURL, "db-url", cfg.DatabaseURL, "Database connection string")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Request timeout")
	flag.IntVar(&cfg.MaxRetries, "retries", cfg.MaxRetries, "Maximum retry attempts")
	flag.IntVar(&cfg.Parallelism, "parallelism", cfg.Parallelism, "Number of concurrent checks")
	flag.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "Enable verbose output")
	flag.StringVar(&cfg.Format, "format", cfg.Format, "Output format: terminal, json, ci")
	flag.BoolVar(&cfg.SkipGRPC, "skip-grpc", cfg.SkipGRPC, "Skip gRPC verification")
	flag.BoolVar(&cfg.SkipGraphQL, "skip-graphql", cfg.SkipGraphQL, "Skip GraphQL verification")
	flag.BoolVar(&cfg.SkipRLS, "skip-rls", cfg.SkipRLS, "Skip RLS verification")

	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help")

	flag.Parse()

	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Println("vyst-verify", version, "(commit:", commit, ")")
		os.Exit(0)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n⚠️  Received interrupt, cancelling...")
		cancel()
	}()

	// Run verification
	if err := run(ctx, cfg); err != nil {
		if cfg.Format != "json" {
			fmt.Fprintf(os.Stderr, "\n❌ Verification failed: %v\n", err)
		}
		os.Exit(1)
	}

	os.Exit(0)
}

func run(ctx context.Context, cfg *config.Config) error {
	// Create runner
	r := runner.NewRunner(cfg)

	// Create shared auth context for dependent checks
	authCtx := checks.NewAuthContext()

	// Register checks in order of dependency
	// Health checks (no dependencies)
	r.AddChecks(checks.HealthChecks())

	// Auth checks (depend on each other, run sequentially)
	// Note: These are registered but the runner handles them
	// For auth flow, we need sequential execution
	authChecks := checks.AuthChecks(authCtx)

	// Run auth checks sequentially first
	authRunner := runner.NewRunner(&config.Config{
		BaseURL:     cfg.BaseURL,
		DatabaseURL: cfg.DatabaseURL,
		Timeout:     cfg.Timeout,
		MaxRetries:  cfg.MaxRetries,
		RetryDelay:  cfg.RetryDelay,
		Parallelism: 1, // Sequential for auth
		Verbose:     cfg.Verbose,
		Format:      "terminal", // Suppress output for now
	})

	for _, check := range authChecks {
		authRunner.AddCheck(check)
	}

	// Execute auth checks first
	if cfg.Format == "terminal" && cfg.Verbose {
		fmt.Println("\n📋 Running Authentication Flow...")
	}

	authCtxWithTimeout, authCancel := context.WithTimeout(ctx, 60*time.Second)
	defer authCancel()

	if err := authRunner.Run(authCtxWithTimeout); err != nil {
		// Auth failed, but we can still run independent checks
		if cfg.Verbose {
			fmt.Printf("⚠️  Auth flow had failures, some checks may be skipped\n")
		}
	}

	// Copy results from auth runner
	for _, result := range authRunner.Results() {
		r.AddCheck(runner.Check{
			Name:  result.Name,
			Group: result.Group,
			Fn: func(result runner.CheckResult) func(context.Context, *config.Config) (*runner.CheckResult, error) {
				return func(ctx context.Context, cfg *config.Config) (*runner.CheckResult, error) {
					return &result, nil
				}
			}(result),
		})
	}

	// REST API checks (depend on auth token)
	if authCtx.Token != "" {
		r.AddChecks(checks.RESTChecks(authCtx))
		r.AddChecks(checks.CompanyChecks(authCtx))
		r.AddChecks(checks.DocumentChecks(authCtx))
	}

	// GraphQL checks
	if !cfg.SkipGraphQL {
		r.AddChecks(checks.GraphQLChecks(authCtx))
	}

	// gRPC checks
	if !cfg.SkipGRPC {
		r.AddChecks(checks.GRPCChecks(authCtx))
	}

	// Run all remaining checks
	return r.Run(ctx)
}

func printUsage() {
	fmt.Print(`
Vyst Identity - System Verification CLI

USAGE:
    vyst-verify [OPTIONS]

OPTIONS:
    --url <URL>          Base URL of the API (default: http://localhost:8982)
    --grpc-url <URL>     gRPC server address (default: localhost:52151)
    --db-url <URL>       Database connection string
    --timeout <DURATION> Request timeout (default: 10s)
    --retries <N>        Maximum retry attempts (default: 3)
    --parallelism <N>    Number of concurrent checks (default: 4)
    --verbose            Enable verbose output
    --format <FORMAT>    Output format: terminal, json, ci (default: terminal)
    --skip-grpc          Skip gRPC verification
    --skip-graphql       Skip GraphQL verification
    --skip-rls           Skip RLS verification
    --version            Show version information
    --help               Show this help message

EXAMPLES:
    # Verify local development server
    go run ./cmd/verify/...

    # Verify staging environment
    go run ./cmd/verify/... --url=https://staging.example.com

    # CI mode with JSON output
    go run ./cmd/verify/... --format=json --url=$API_URL

    # Verbose mode for debugging
    go run ./cmd/verify/... --verbose --retries=5
`)
}
