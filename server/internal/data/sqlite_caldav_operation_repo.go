package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

const (
	DefaultCalDAVOperationRetentionCount = 1000
	DefaultCalDAVOperationRetentionAge   = 14 * 24 * time.Hour
)

// SQLiteCalDAVOperationRepo persists redacted CalDAV operation metadata.
type SQLiteCalDAVOperationRepo struct {
	db             *sql.DB
	retentionCount int
	retentionAge   time.Duration
}

func NewSQLiteCalDAVOperationRepo(db *sql.DB) *SQLiteCalDAVOperationRepo {
	return &SQLiteCalDAVOperationRepo{
		db:             db,
		retentionCount: DefaultCalDAVOperationRetentionCount,
		retentionAge:   DefaultCalDAVOperationRetentionAge,
	}
}

func NewSQLiteCalDAVOperationRepoWithRetention(db *sql.DB, count int, age time.Duration) *SQLiteCalDAVOperationRepo {
	if count <= 0 {
		count = DefaultCalDAVOperationRetentionCount
	}
	if age <= 0 {
		age = DefaultCalDAVOperationRetentionAge
	}
	return &SQLiteCalDAVOperationRepo{db: db, retentionCount: count, retentionAge: age}
}

func (r *SQLiteCalDAVOperationRepo) Record(operation *domain.CalDAVOperation) error {
	if operation == nil {
		return fmt.Errorf("caldav operation is nil")
	}
	if operation.ID == "" {
		operation.ID = fmt.Sprintf("caldav-%d", time.Now().UnixNano())
	}
	if operation.OccurredAt.IsZero() {
		operation.OccurredAt = time.Now().UTC()
	}

	query := `
		INSERT INTO caldav_operations (
			operation_id, occurred_at, method, path_pattern, status_code, duration_ms,
			client_user_agent, client_fingerprint, etag_outcome, operation_kind, outcome,
			error_code, redacted_error, request_size_bytes, response_size_bytes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(context.Background(), query,
		operation.ID,
		operation.OccurredAt,
		operation.Method,
		operation.PathPattern,
		operation.StatusCode,
		operation.DurationMillis,
		operation.ClientUserAgent,
		operation.ClientFingerprint,
		string(operation.ETagOutcome),
		string(operation.OperationKind),
		string(operation.Outcome),
		string(operation.ErrorCode),
		operation.RedactedError,
		operation.RequestSizeBytes,
		operation.ResponseSizeBytes,
	)
	if err != nil {
		return fmt.Errorf("failed to record CalDAV operation: %w", err)
	}
	return r.Prune()
}

func (r *SQLiteCalDAVOperationRepo) Prune() error {
	cutoff := time.Now().UTC().Add(-r.retentionAge)
	if _, err := r.db.ExecContext(context.Background(), `DELETE FROM caldav_operations WHERE occurred_at < ?`, cutoff); err != nil {
		return fmt.Errorf("failed to prune old CalDAV operations: %w", err)
	}

	query := `
		DELETE FROM caldav_operations
		WHERE id NOT IN (
			SELECT id FROM caldav_operations
			ORDER BY occurred_at DESC, id DESC
			LIMIT ?
		)
	`
	if _, err := r.db.ExecContext(context.Background(), query, r.retentionCount); err != nil {
		return fmt.Errorf("failed to prune CalDAV operations by count: %w", err)
	}
	return nil
}

func (r *SQLiteCalDAVOperationRepo) ListRecent(ctx context.Context, limit int) ([]*domain.CalDAVOperation, error) {
	if limit <= 0 || limit > r.retentionCount {
		limit = r.retentionCount
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT operation_id, occurred_at, method, path_pattern, status_code, duration_ms,
		       client_user_agent, client_fingerprint, etag_outcome, operation_kind, outcome,
		       error_code, redacted_error, request_size_bytes, response_size_bytes
		FROM caldav_operations
		ORDER BY occurred_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list CalDAV operations: %w", err)
	}
	defer rows.Close()

	var operations []*domain.CalDAVOperation
	for rows.Next() {
		op := &domain.CalDAVOperation{}
		var etagOutcome, kind, outcome, errorCode string
		if err := rows.Scan(
			&op.ID,
			&op.OccurredAt,
			&op.Method,
			&op.PathPattern,
			&op.StatusCode,
			&op.DurationMillis,
			&op.ClientUserAgent,
			&op.ClientFingerprint,
			&etagOutcome,
			&kind,
			&outcome,
			&errorCode,
			&op.RedactedError,
			&op.RequestSizeBytes,
			&op.ResponseSizeBytes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan CalDAV operation: %w", err)
		}
		op.ETagOutcome = domain.CalDAVETagOutcome(etagOutcome)
		op.OperationKind = domain.CalDAVOperationKind(kind)
		op.Outcome = domain.CalDAVOperationOutcome(outcome)
		op.ErrorCode = domain.CalDAVErrorCode(errorCode)
		operations = append(operations, op)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate CalDAV operations: %w", err)
	}
	return operations, nil
}
