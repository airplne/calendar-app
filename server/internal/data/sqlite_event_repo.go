package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

// SQLiteEventRepo implements domain.EventRepo using SQLite
type SQLiteEventRepo struct {
	db *sql.DB
	tx *sql.Tx // optional transaction; when set, used instead of db
}

// NewSQLiteEventRepo creates a new SQLite event repository
func NewSQLiteEventRepo(db *sql.DB) *SQLiteEventRepo {
	return &SQLiteEventRepo{db: db}
}

// WithTx returns a new SQLiteEventRepo that operates within the given transaction.
// The returned instance shares the same db reference but uses tx for all operations.
func (r *SQLiteEventRepo) WithTx(tx *sql.Tx) *SQLiteEventRepo {
	return &SQLiteEventRepo{
		db: r.db,
		tx: tx,
	}
}

// execer returns either the transaction or the database for executing queries.
// This allows the same methods to work with or without an active transaction.
func (r *SQLiteEventRepo) execer() interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// Create inserts a new event into the database
func (r *SQLiteEventRepo) Create(ctx context.Context, event *domain.Event) error {
	// Validate event
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	query := `
		INSERT INTO events (
			calendar_id, uid, ics, summary, description, location,
			start_time, end_time, all_day, recurrence_rule, etag,
			sequence, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.execer().ExecContext(ctx, query,
		event.CalendarID,
		event.UID,
		event.ICS,
		event.Summary,
		nullString(event.Description),
		nullString(event.Location),
		event.StartTime,
		event.EndTime,
		event.AllDay,
		nullString(event.RecurrenceRule),
		event.ETag,
		event.Sequence,
		event.Status,
		now,
		now,
	)
	if err != nil {
		// Check for UNIQUE constraint violation
		if isUniqueConstraintError(err) {
			return domain.ErrConflict
		}
		return fmt.Errorf("failed to create event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	event.ID = id
	event.CreatedAt = now
	event.UpdatedAt = now

	return nil
}

// GetByUID retrieves an event by calendar ID and UID
func (r *SQLiteEventRepo) GetByUID(ctx context.Context, calendarID int64, uid string) (*domain.Event, error) {
	query := `
		SELECT id, calendar_id, uid, ics, summary, description, location,
			   start_time, end_time, all_day, recurrence_rule, etag,
			   sequence, status, created_at, updated_at
		FROM events
		WHERE calendar_id = ? AND uid = ?
	`

	row := r.execer().QueryRowContext(ctx, query, calendarID, uid)
	event, err := scanEvent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get event by UID: %w", err)
	}

	return event, nil
}

// GetByID retrieves an event by its ID
func (r *SQLiteEventRepo) GetByID(ctx context.Context, id int64) (*domain.Event, error) {
	query := `
		SELECT id, calendar_id, uid, ics, summary, description, location,
			   start_time, end_time, all_day, recurrence_rule, etag,
			   sequence, status, created_at, updated_at
		FROM events
		WHERE id = ?
	`

	row := r.execer().QueryRowContext(ctx, query, id)
	event, err := scanEvent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get event by ID: %w", err)
	}

	return event, nil
}

// List retrieves events within a time range for a calendar
// Returns events that overlap with the [start, end] range
func (r *SQLiteEventRepo) List(ctx context.Context, calendarID int64, start, end time.Time) ([]*domain.Event, error) {
	query := `
		SELECT id, calendar_id, uid, ics, summary, description, location,
			   start_time, end_time, all_day, recurrence_rule, etag,
			   sequence, status, created_at, updated_at
		FROM events
		WHERE calendar_id = ? AND start_time < ? AND end_time > ?
		ORDER BY start_time ASC
	`

	rows, err := r.execer().QueryContext(ctx, query, calendarID, end, start)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// ListAll retrieves all events for a calendar
func (r *SQLiteEventRepo) ListAll(ctx context.Context, calendarID int64) ([]*domain.Event, error) {
	query := `
		SELECT id, calendar_id, uid, ics, summary, description, location,
			   start_time, end_time, all_day, recurrence_rule, etag,
			   sequence, status, created_at, updated_at
		FROM events
		WHERE calendar_id = ?
		ORDER BY start_time ASC
	`

	rows, err := r.execer().QueryContext(ctx, query, calendarID)
	if err != nil {
		return nil, fmt.Errorf("failed to list all events: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// Update updates an event with ETag validation
// Returns domain.ErrPreconditionFailed if the ETag doesn't match
func (r *SQLiteEventRepo) Update(ctx context.Context, event *domain.Event, expectedETag string) error {
	// Validate event
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	// If we're already in a transaction (r.tx is set), use it directly.
	// Otherwise, create our own transaction for atomic ETag check + update.
	if r.tx != nil {
		return r.updateInTx(ctx, r.tx, event, expectedETag)
	}

	// Use transaction to ensure ETag check and update are atomic
	return WithTx(ctx, r.db, func(tx *sql.Tx) error {
		return r.updateInTx(ctx, tx, event, expectedETag)
	})
}

// updateInTx performs the actual update within a transaction
func (r *SQLiteEventRepo) updateInTx(ctx context.Context, tx *sql.Tx, event *domain.Event, expectedETag string) error {
	// First, check the current ETag
	var currentETag string
	err := tx.QueryRowContext(ctx,
		"SELECT etag FROM events WHERE calendar_id = ? AND uid = ?",
		event.CalendarID, event.UID,
	).Scan(&currentETag)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to check ETag: %w", err)
	}

	// Verify ETag matches
	if currentETag != expectedETag {
		slog.Debug("ETag mismatch",
			"expected", expectedETag,
			"current", currentETag,
			"calendar_id", event.CalendarID,
			"uid", event.UID,
		)
		return domain.ErrPreconditionFailed
	}

	// ETag matches, proceed with update
	query := `
		UPDATE events
		SET ics = ?, summary = ?, description = ?, location = ?,
			start_time = ?, end_time = ?, all_day = ?, recurrence_rule = ?,
			etag = ?, sequence = ?, status = ?, updated_at = ?
		WHERE calendar_id = ? AND uid = ?
	`

	now := time.Now()
	result, err := tx.ExecContext(ctx, query,
		event.ICS,
		event.Summary,
		nullString(event.Description),
		nullString(event.Location),
		event.StartTime,
		event.EndTime,
		event.AllDay,
		nullString(event.RecurrenceRule),
		event.ETag,
		event.Sequence,
		event.Status,
		now,
		event.CalendarID,
		event.UID,
	)
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	event.UpdatedAt = now
	return nil
}

// Delete removes an event from the database
func (r *SQLiteEventRepo) Delete(ctx context.Context, calendarID int64, uid string) error {
	query := `DELETE FROM events WHERE calendar_id = ? AND uid = ?`

	result, err := r.execer().ExecContext(ctx, query, calendarID, uid)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Helper functions are defined in sqlite_calendar_repo.go:
// - nullString
// - fromNullString
// - isUniqueConstraintError

// scanEvent scans a row into an Event struct
func scanEvent(row interface{ Scan(...interface{}) error }) (*domain.Event, error) {
	var e domain.Event
	var description, location, recurrenceRule sql.NullString

	err := row.Scan(
		&e.ID,
		&e.CalendarID,
		&e.UID,
		&e.ICS,
		&e.Summary,
		&description,
		&location,
		&e.StartTime,
		&e.EndTime,
		&e.AllDay,
		&recurrenceRule,
		&e.ETag,
		&e.Sequence,
		&e.Status,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	e.Description = fromNullString(description)
	e.Location = fromNullString(location)
	e.RecurrenceRule = fromNullString(recurrenceRule)

	return &e, nil
}
