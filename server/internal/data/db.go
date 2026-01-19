package data

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/glebarez/sqlite" // Register SQLite driver
)

// OpenDB opens a SQLite database connection with WAL mode enabled
// dataDir is the directory where calendar.db will be stored
func OpenDB(dataDir string) (*sql.DB, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "calendar.db")

	// Open database with WAL mode and foreign keys enabled
	// Use _pragma=journal_mode(wal)&_pragma=foreign_keys(on) in DSN
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Database connection established",
		"path", dbPath,
		"journal_mode", "wal",
	)

	return db, nil
}

// CloseDB closes the database connection gracefully
func CloseDB(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}
