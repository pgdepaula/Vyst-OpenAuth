// Package testutil provides shared test utilities and helpers.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDB returns the database URL for tests.
// It first checks TEST_DATABASE_URL, then DATABASE_URL, then defaults.
func TestDB() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://postgres:postgres@localhost:5432/vyst_test?sslmode=disable"
}

// TestAPIURL returns the API URL for E2E tests.
func TestAPIURL() string {
	if url := os.Getenv("API_URL"); url != "" {
		return url
	}
	return "http://localhost:8982"
}

// WaitForDB waits for the database to become available.
func WaitForDB(t *testing.T, connStr string, timeout time.Duration) *sql.DB {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var db *sql.DB
	var err error

	for {
		select {
		case <-ctx.Done():
			require.NoError(t, fmt.Errorf("database not available after %v: %w", timeout, err))
		default:
			db, err = sql.Open("postgres", connStr)
			if err == nil {
				if pingErr := db.PingContext(ctx); pingErr == nil {
					return db
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// CleanupTable truncates a table for test isolation.
func CleanupTable(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()
	_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", tableName))
	require.NoError(t, err, "failed to cleanup table %s", tableName)
}

// RandomEmail generates a random test email.
func RandomEmail() string {
	return fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())
}

// RandomTenantName generates a random tenant name.
func RandomTenantName() string {
	return fmt.Sprintf("tenant_%d", time.Now().UnixNano())
}
