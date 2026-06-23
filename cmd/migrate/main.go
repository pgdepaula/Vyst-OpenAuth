package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for database/sql
	"github.com/joho/godotenv"
)

type migrationConfig struct {
	dir   string
	dsn   string
	reset bool
}

func main() {
	// Load .env file
	_ = godotenv.Load()

	cfg := parseFlags()

	db := openDatabase(cfg.dsn)
	defer func() { _ = db.Close() }()

	resetDatabase(db, cfg.reset)
	ensureMigrationsTable(db)

	migrations := readMigrations(cfg.dir)
	currentVersion := getCurrentVersion(db)
	applyPendingMigrations(db, cfg.dir, migrations, currentVersion)

	log.Println("All migrations applied.")
}

func parseFlags() migrationConfig {
	dir := flag.String("dir", "migrations", "Directory containing migration files")
	dsn := flag.String("dsn", "", "Database connection string")
	reset := flag.Bool("reset", false, "Reset database (DROP SCHEMA public CASCADE)")
	flag.Parse()

	if *dsn == "" {
		*dsn = os.Getenv("DATABASE_URL")
	}
	if *dsn == "" {
		*dsn = "postgres://postgres:postgres@localhost:5432/vyst_identity?sslmode=disable"
	}
	return migrationConfig{
		dir:   *dir,
		dsn:   *dsn,
		reset: *reset,
	}
}

func openDatabase(dsn string) *sql.DB {
	log.Printf("Connecting to database...")
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	return db
}

func resetDatabase(db *sql.DB, reset bool) {
	if !reset {
		return
	}
	log.Println("Resetting database...")
	if _, err := db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO postgres; GRANT ALL ON SCHEMA public TO public;"); err != nil {
		log.Fatalf("Failed to reset database: %v", err)
	}
	log.Println("Database reset successfully.")
}

func ensureMigrationsTable(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			dirty BOOLEAN NOT NULL
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create schema_migrations table: %v", err)
	}
}

func readMigrations(dir string) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	var migrations []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".up.sql") {
			migrations = append(migrations, f.Name())
		}
	}
	sort.Strings(migrations)
	return migrations
}

func getCurrentVersion(db *sql.DB) int64 {
	var currentVersion int64
	var dirty bool
	err := db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&currentVersion, &dirty)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatalf("Failed to get current version: %v", err)
	}
	if dirty {
		log.Fatalf("Database is in dirty state. Fix manually.")
	}
	log.Printf("Current version: %d", currentVersion)
	return currentVersion
}

func applyPendingMigrations(db *sql.DB, dir string, migrations []string, currentVersion int64) {
	for _, migration := range migrations {
		version, err := parseMigrationVersion(migration)
		if err != nil {
			log.Fatalf("Failed to parse migration version from %s: %v", migration, err)
		}

		if version <= currentVersion {
			continue
		}
		applyMigration(db, dir, migration, version)
	}
}

func parseMigrationVersion(name string) (int64, error) {
	var version int64
	n, err := fmt.Sscanf(name, "%d", &version)
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, fmt.Errorf("missing numeric migration prefix")
	}
	return version, nil
}

func applyMigration(db *sql.DB, dir, migration string, version int64) {
	log.Printf("Applying migration: %s", migration)

	// #nosec G304 - migration file path is controlled and validated by the runner parameters
	content, err := os.ReadFile(filepath.Join(dir, migration))
	if err != nil {
		log.Fatalf("Failed to read migration file %s: %v", migration, err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	if _, err := tx.Exec("DELETE FROM schema_migrations"); err != nil {
		rollbackAndFatal(tx, "Failed to clear version: %v", err)
	}
	if _, err := tx.Exec("INSERT INTO schema_migrations (version, dirty) VALUES ($1, $2)", version, true); err != nil {
		rollbackAndFatal(tx, "Failed to set dirty: %v", err)
	}

	if _, err := tx.Exec(string(content)); err != nil {
		rollbackAndFatal(tx, "Failed to execute migration %s: %v", migration, err)
	}
	if _, err := tx.Exec("UPDATE schema_migrations SET dirty = $1 WHERE version = $2", false, version); err != nil {
		rollbackAndFatal(tx, "Failed to set clean: %v", err)
	}
	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	log.Printf("Applied migration %s successfully", migration)
}

func rollbackAndFatal(tx *sql.Tx, format string, args ...interface{}) {
	if err := tx.Rollback(); err != nil {
		log.Printf("Failed to rollback transaction: %v", err)
	}
	log.Fatalf(format, args...)
}
