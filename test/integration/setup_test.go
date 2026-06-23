package integration

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestEnv struct {
	DB        *sql.DB // App user (vyst_app)
	AdminDB   *sql.DB // Superuser (postgres)
	Pool      *pgxpool.Pool
	Container *postgres.PostgresContainer
	ConnStr   string
}

func SetupTestEnv(t *testing.T) *TestEnv {
	ctx := context.Background()

	dbName := "vyst_test"
	dbUser := "postgres"
	dbPassword := "postgres"

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	// Apply migrations
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	migrationsPath := filepath.Join(basepath, "../../migrations")

	m, err := migrate.New(
		"file://"+migrationsPath,
		connStr,
	)
	require.NoError(t, err)

	err = m.Up()
	require.NoError(t, err)

	// Create pgx pool for application code (as vyst_app to enforce RLS)
	// We need to construct the connection string for vyst_app
	// The container returns a string like "postgres://postgres:postgres@localhost:port/vyst_test?sslmode=disable"
	// We need "postgres://vyst_app:vyst_app_secure_password@localhost:port/vyst_test?sslmode=disable"

	// Simple string replacement (hacky but works for test)
	// Or better, use url.Parse

	// Let's just use the fact that we know the structure or use the container to get the host/port
	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")

	appConnStr := "postgres://vyst_app:vyst_app_secure_password@" + host + ":" + port.Port() + "/" + dbName + "?sslmode=disable"

	// Re-open DB as vyst_app for the test to use
	appDB, err := sql.Open("postgres", appConnStr)
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, appConnStr)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
		_ = appDB.Close() // Close app connection
		_ = db.Close()    // Close admin connection
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	return &TestEnv{
		DB:        appDB, // Use app user for tests
		AdminDB:   db,    // Use superuser for verification
		Pool:      pool,
		Container: pgContainer,
		ConnStr:   appConnStr,
	}
}
