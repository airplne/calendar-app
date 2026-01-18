package data

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
)

// RunMigrations executes pending database migrations
// migrationsDir is the path to the directory containing .sql migration files
func RunMigrations(db *sql.DB, migrationsDir string) error {
	// Set goose dialect to sqlite3
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	slog.Info("Running database migrations", "dir", migrationsDir)

	// Run all pending migrations
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get current version
	version, err := goose.GetDBVersion(db)
	if err != nil {
		slog.Warn("Could not get migration version", "error", err)
	} else {
		slog.Info("Migrations complete", "version", version)
	}

	return nil
}

// MigrateOnly runs migrations and returns (useful for --migrate-only flag)
func MigrateOnly(dataDir, migrationsDir string) error {
	db, err := OpenDB(dataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	return RunMigrations(db, migrationsDir)
}
