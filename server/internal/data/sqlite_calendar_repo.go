package data

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

// SQLiteCalendarRepo implements domain.CalendarRepo using SQLite
type SQLiteCalendarRepo struct {
	db *sql.DB
}

// NewSQLiteCalendarRepo creates a new SQLite calendar repository
func NewSQLiteCalendarRepo(db *sql.DB) *SQLiteCalendarRepo {
	return &SQLiteCalendarRepo{db: db}
}

func (r *SQLiteCalendarRepo) Create(ctx context.Context, calendar *domain.Calendar) error {
	if err := calendar.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `INSERT INTO calendars (user_id, name, display_name, color, description, sync_token, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		calendar.UserID,
		calendar.Name,
		calendar.DisplayName,
		nullString(calendar.Color),
		nullString(calendar.Description),
		nullString(calendar.SyncToken),
		now,
		now,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrConflict
		}
		return fmt.Errorf("failed to create calendar: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	calendar.ID = id
	calendar.CreatedAt = now
	calendar.UpdatedAt = now

	return nil
}

func (r *SQLiteCalendarRepo) GetByID(ctx context.Context, id int64) (*domain.Calendar, error) {
	query := `SELECT id, user_id, name, display_name, color, description, sync_token, created_at, updated_at
	          FROM calendars WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	calendar, err := scanCalendar(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get calendar: %w", err)
	}

	return calendar, nil
}

func (r *SQLiteCalendarRepo) GetByName(ctx context.Context, userID int64, name string) (*domain.Calendar, error) {
	query := `SELECT id, user_id, name, display_name, color, description, sync_token, created_at, updated_at
	          FROM calendars WHERE user_id = ? AND name = ?`

	row := r.db.QueryRowContext(ctx, query, userID, name)
	calendar, err := scanCalendar(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get calendar: %w", err)
	}

	return calendar, nil
}

func (r *SQLiteCalendarRepo) ListByUser(ctx context.Context, userID int64) ([]*domain.Calendar, error) {
	query := `SELECT id, user_id, name, display_name, color, description, sync_token, created_at, updated_at
	          FROM calendars WHERE user_id = ? ORDER BY created_at`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}
	defer rows.Close()

	var calendars []*domain.Calendar
	for rows.Next() {
		calendar, err := scanCalendar(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan calendar: %w", err)
		}
		calendars = append(calendars, calendar)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return calendars, nil
}

func (r *SQLiteCalendarRepo) Update(ctx context.Context, calendar *domain.Calendar) error {
	if err := calendar.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `UPDATE calendars
	          SET name = ?, display_name = ?, color = ?, description = ?, updated_at = ?
	          WHERE id = ?`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		calendar.Name,
		calendar.DisplayName,
		nullString(calendar.Color),
		nullString(calendar.Description),
		now,
		calendar.ID,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrConflict
		}
		return fmt.Errorf("failed to update calendar: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	calendar.UpdatedAt = now

	return nil
}

func (r *SQLiteCalendarRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM calendars WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete calendar: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *SQLiteCalendarRepo) IncrementSyncToken(ctx context.Context, calendarID int64) (string, error) {
	newToken := fmt.Sprintf("v%d", time.Now().UnixNano())

	result, err := r.db.ExecContext(ctx,
		"UPDATE calendars SET sync_token = ?, updated_at = ? WHERE id = ?",
		newToken, time.Now(), calendarID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to update sync token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return "", domain.ErrNotFound
	}

	return newToken, nil
}

// scanCalendar scans a row into a Calendar struct
func scanCalendar(row interface{ Scan(...interface{}) error }) (*domain.Calendar, error) {
	var c domain.Calendar
	var color, description, syncToken sql.NullString

	err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.Name,
		&c.DisplayName,
		&color,
		&description,
		&syncToken,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	c.Color = fromNullString(color)
	c.Description = fromNullString(description)
	c.SyncToken = fromNullString(syncToken)

	return &c, nil
}

// nullString converts a string to sql.NullString
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// fromNullString converts sql.NullString to string
func fromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// isUniqueConstraintError checks if error is a unique constraint violation
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// SQLite unique constraint error message contains "UNIQUE constraint failed"
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
