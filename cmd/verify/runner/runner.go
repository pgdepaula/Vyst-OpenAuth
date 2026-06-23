// Package runner provides the verification execution engine.
package runner

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"github.com/pgdepaula/vyst-openauth/cmd/verify/config"
)

// CheckResult represents the result of a single verification check.
type CheckResult struct {
	Name       string        `json:"name"`
	Group      string        `json:"group"`
	Passed     bool          `json:"passed"`
	Duration   time.Duration `json:"duration"`
	Error      string        `json:"error,omitempty"`
	StatusCode int           `json:"status_code,omitempty"`
	Retries    int           `json:"retries,omitempty"`
}

// Check is a verification check to run.
type Check struct {
	Name  string
	Group string
	Fn    func(ctx context.Context, cfg *config.Config) (*CheckResult, error)
}

// Runner executes verification checks with parallelism and retry logic.
type Runner struct {
	cfg      *config.Config
	checks   []Check
	results  []CheckResult
	mu       sync.Mutex
	reporter *Reporter
}

// NewRunner creates a new verification runner.
func NewRunner(cfg *config.Config) *Runner {
	return &Runner{
		cfg:      cfg,
		checks:   make([]Check, 0),
		results:  make([]CheckResult, 0),
		reporter: NewReporter(cfg),
	}
}

// AddCheck registers a check to be executed.
func (r *Runner) AddCheck(check Check) {
	r.checks = append(r.checks, check)
}

// AddChecks registers multiple checks.
func (r *Runner) AddChecks(checks []Check) {
	r.checks = append(r.checks, checks...)
}

// Run executes all registered checks with parallelism and retry logic.
func (r *Runner) Run(ctx context.Context) error {
	r.reporter.Start(len(r.checks))

	// If parallelism is 1, run strictly sequentially to guarantee order
	if r.cfg.Parallelism == 1 {
		for _, check := range r.checks {
			result := r.runCheckWithRetry(ctx, check)
			r.addResult(result)
			r.reporter.ReportCheck(result)
		}
		return r.reporter.Finish(r.results)
	}

	// Create a semaphore for parallelism control
	sem := make(chan struct{}, r.cfg.Parallelism)
	var wg sync.WaitGroup

	for _, check := range r.checks {
		wg.Add(1)
		go func(c Check) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			result := r.runCheckWithRetry(ctx, c)
			r.addResult(result)
			r.reporter.ReportCheck(result)
		}(check)
	}

	wg.Wait()
	return r.reporter.Finish(r.results)
}

// runCheckWithRetry executes a check with exponential backoff retry.
func (r *Runner) runCheckWithRetry(ctx context.Context, check Check) CheckResult {
	var lastErr error
	var result *CheckResult

	for attempt := 0; attempt <= r.cfg.MaxRetries; attempt++ {
		start := time.Now()

		checkResult, err := check.Fn(ctx, r.cfg)
		if err == nil && checkResult != nil && checkResult.Passed {
			checkResult.Retries = attempt
			return *checkResult
		}

		if checkResult != nil {
			result = checkResult
			result.Retries = attempt
		}
		lastErr = err

		if attempt < r.cfg.MaxRetries {
			delay := r.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return CheckResult{
					Name:     check.Name,
					Group:    check.Group,
					Passed:   false,
					Duration: time.Since(start),
					Error:    "context cancelled",
					Retries:  attempt,
				}
			case <-time.After(delay):
				// Continue to next retry
			}
		}
	}

	// All retries exhausted
	if result != nil {
		return *result
	}

	errMsg := "unknown error"
	if lastErr != nil {
		errMsg = lastErr.Error()
	}

	return CheckResult{
		Name:    check.Name,
		Group:   check.Group,
		Passed:  false,
		Error:   errMsg,
		Retries: r.cfg.MaxRetries,
	}
}

// calculateBackoff returns the delay for a given retry attempt using exponential backoff with jitter.
func (r *Runner) calculateBackoff(attempt int) time.Duration {
	base := r.cfg.RetryDelay
	multiplier := 1
	for i := 0; i < attempt; i++ {
		multiplier *= 2
	}
	delay := base * time.Duration(multiplier)
	// Add jitter: ±25%
	jitter := time.Duration(secureInt63n(int64(delay / 4)))
	if secureInt63n(2) == 0 {
		delay += jitter
	} else {
		delay -= jitter
	}
	return delay
}

func secureInt63n(max int64) int64 {
	if max <= 0 {
		return 0
	}
	value, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0
	}
	return value.Int64()
}

// addResult safely adds a result to the results slice.
func (r *Runner) addResult(result CheckResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results = append(r.results, result)
}

// Results returns all check results.
func (r *Runner) Results() []CheckResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]CheckResult{}, r.results...)
}

// Summary returns a summary of the verification run.
func (r *Runner) Summary() (total, passed, failed int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	total = len(r.results)
	for _, result := range r.results {
		if result.Passed {
			passed++
		} else {
			failed++
		}
	}
	return
}

// HasFailures returns true if any check failed.
func (r *Runner) HasFailures() bool {
	_, _, failed := r.Summary()
	return failed > 0
}
