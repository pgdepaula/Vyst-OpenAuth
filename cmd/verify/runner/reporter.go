package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pgdepaula/vyst-openauth/cmd/verify/config"
)

// Reporter handles output formatting for verification results.
type Reporter struct {
	cfg       *config.Config
	startTime time.Time
	total     int
}

// NewReporter creates a new reporter with the given configuration.
func NewReporter(cfg *config.Config) *Reporter {
	return &Reporter{cfg: cfg}
}

// Start initializes the reporter for a new verification run.
func (r *Reporter) Start(total int) {
	r.startTime = time.Now()
	r.total = total

	switch r.cfg.Format {
	case "json":
		// No header for JSON
	case "ci":
		fmt.Println("::group::Vyst Identity System Verification")
	default: // terminal
		r.printHeader()
	}
}

// ReportCheck reports a single check result.
func (r *Reporter) ReportCheck(result CheckResult) {
	if r.cfg.Format == "json" {
		// JSON format collects all results at the end
		return
	}

	icon := "✅"
	if !result.Passed {
		icon = "❌"
	}

	switch r.cfg.Format {
	case "ci":
		if result.Passed {
			fmt.Printf("  %s %s (%s)\n", icon, result.Name, result.Duration.Round(time.Millisecond))
		} else {
			fmt.Printf("  %s %s: %s\n", icon, result.Name, result.Error)
		}
	default: // terminal
		if r.cfg.Verbose || !result.Passed {
			retryInfo := ""
			if result.Retries > 0 {
				retryInfo = fmt.Sprintf(" (retries: %d)", result.Retries)
			}
			if result.Passed {
				fmt.Printf("  %s %s%s - %s\n", icon, result.Name, retryInfo, result.Duration.Round(time.Millisecond))
			} else {
				fmt.Printf("  %s %s%s - %s\n", icon, result.Name, retryInfo, result.Error)
			}
		}
	}
}

// Finish completes the verification run and outputs the final summary.
func (r *Reporter) Finish(results []CheckResult) error {
	elapsed := time.Since(r.startTime)

	switch r.cfg.Format {
	case "json":
		return r.finishJSON(results, elapsed)
	case "ci":
		return r.finishCI(results, elapsed)
	default:
		return r.finishTerminal(results, elapsed)
	}
}

func (r *Reporter) printHeader() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         VYST IDENTITY - SYSTEM VERIFICATION                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n🎯 Target: %s\n", r.cfg.BaseURL)
	fmt.Printf("⚙️  Parallelism: %d | Retries: %d\n\n", r.cfg.Parallelism, r.cfg.MaxRetries)
}

func (r *Reporter) finishTerminal(results []CheckResult, elapsed time.Duration) error {
	passed := 0
	failed := 0
	var failures []CheckResult

	for _, result := range results {
		if result.Passed {
			passed++
		} else {
			failed++
			failures = append(failures, result)
		}
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                         SUMMARY")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("\n📊 Total: %d | ✅ Passed: %d | ❌ Failed: %d\n", len(results), passed, failed)
	fmt.Printf("⏱️  Duration: %s\n", elapsed.Round(time.Millisecond))

	if len(failures) > 0 {
		fmt.Println("\n❌ Failed Checks:")
		// Group by group
		grouped := make(map[string][]CheckResult)
		for _, f := range failures {
			grouped[f.Group] = append(grouped[f.Group], f)
		}

		groups := make([]string, 0, len(grouped))
		for g := range grouped {
			groups = append(groups, g)
		}
		sort.Strings(groups)

		for _, g := range groups {
			fmt.Printf("\n  [%s]\n", g)
			for _, f := range grouped[g] {
				fmt.Printf("    • %s: %s\n", f.Name, f.Error)
			}
		}
	}

	if failed > 0 {
		fmt.Println("\n❌ Verification FAILED")
		return fmt.Errorf("%d checks failed", failed)
	}

	fmt.Println("\n✅ All verifications passed!")
	return nil
}

func (r *Reporter) finishCI(results []CheckResult, elapsed time.Duration) error {
	passed := 0
	failed := 0

	for _, result := range results {
		if result.Passed {
			passed++
		} else {
			failed++
		}
	}

	fmt.Println("::endgroup::")
	fmt.Printf("\n📊 Results: %d passed, %d failed (%s)\n", passed, failed, elapsed.Round(time.Millisecond))

	if failed > 0 {
		return fmt.Errorf("%d checks failed", failed)
	}
	return nil
}

func (r *Reporter) finishJSON(results []CheckResult, elapsed time.Duration) error {
	passed := 0
	failed := 0

	for _, result := range results {
		if result.Passed {
			passed++
		} else {
			failed++
		}
	}

	output := struct {
		Total    int           `json:"total"`
		Passed   int           `json:"passed"`
		Failed   int           `json:"failed"`
		Duration string        `json:"duration"`
		Success  bool          `json:"success"`
		Results  []CheckResult `json:"results"`
	}{
		Total:    len(results),
		Passed:   passed,
		Failed:   failed,
		Duration: elapsed.Round(time.Millisecond).String(),
		Success:  failed == 0,
		Results:  results,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	if failed > 0 {
		return fmt.Errorf("%d checks failed", failed)
	}
	return nil
}

// FormatDuration formats a duration in a human-readable way.
func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// TruncateString truncates a string to the given length.
func TruncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
