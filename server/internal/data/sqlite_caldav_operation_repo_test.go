package data

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

func TestSQLiteCalDAVOperationRepo_RecordAndListRecent(t *testing.T) {
	db := openOperationTestDB(t)
	repo := NewSQLiteCalDAVOperationRepoWithRetention(db, 10, 14*24*time.Hour)

	op := &domain.CalDAVOperation{
		ID:                "op-1",
		OccurredAt:        time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
		Method:            "PUT",
		PathPattern:       "/dav/calendars/{principal}/{calendar}/{object}.ics",
		StatusCode:        412,
		DurationMillis:    25,
		ClientFingerprint: domain.CalDAVClientFantastical,
		ETagOutcome:       domain.CalDAVETagMismatched,
		OperationKind:     domain.CalDAVOperationWrite,
		Outcome:           domain.CalDAVOperationRecoverableFailure,
		ErrorCode:         domain.CalDAVErrorETagConflict,
		RedactedError:     "ETag precondition failed.",
		RequestSizeBytes:  123,
		ResponseSizeBytes: 45,
	}
	if err := repo.Record(op); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	operations, err := repo.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}
	if len(operations) != 1 {
		t.Fatalf("len(operations) = %d, want 1", len(operations))
	}
	got := operations[0]
	if got.ID != op.ID || got.Method != op.Method || got.PathPattern != op.PathPattern || got.ErrorCode != op.ErrorCode || got.ClientFingerprint != op.ClientFingerprint {
		t.Fatalf("operation mismatch: got %+v want %+v", got, op)
	}
}

func TestSQLiteCalDAVOperationRepo_DoesNotPersistRawUserAgent(t *testing.T) {
	db := openOperationTestDB(t)
	repo := NewSQLiteCalDAVOperationRepoWithRetention(db, 10, 14*24*time.Hour)
	privateUA := "PrivateDeviceName/SecretBuildToken CalendarClient"

	op := operationFixture("op-private-ua", time.Now().UTC())
	op.ClientFingerprint = domain.CalDAVClientUnknown
	if err := repo.Record(op); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	operations, err := repo.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}
	if len(operations) != 1 {
		t.Fatalf("len(operations) = %d, want 1", len(operations))
	}
	if operations[0].ClientFingerprint != domain.CalDAVClientUnknown {
		t.Fatalf("ClientFingerprint = %q, want %q", operations[0].ClientFingerprint, domain.CalDAVClientUnknown)
	}

	var tableSQL string
	if err := db.QueryRowContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'caldav_operations'`).Scan(&tableSQL); err != nil {
		t.Fatalf("failed to read table schema: %v", err)
	}
	if strings.Contains(tableSQL, "client_user_agent") {
		t.Fatalf("caldav_operations schema should not include raw user-agent column: %s", tableSQL)
	}

	rows, err := db.QueryContext(context.Background(), `SELECT operation_id, method, path_pattern, client_fingerprint, etag_outcome, operation_kind, outcome, error_code, redacted_error FROM caldav_operations`)
	if err != nil {
		t.Fatalf("failed to query persisted metadata: %v", err)
	}
	defer rows.Close()
	var persisted strings.Builder
	for rows.Next() {
		var cols [9]string
		if err := rows.Scan(&cols[0], &cols[1], &cols[2], &cols[3], &cols[4], &cols[5], &cols[6], &cols[7], &cols[8]); err != nil {
			t.Fatalf("failed to scan persisted metadata: %v", err)
		}
		persisted.WriteString(strings.Join(cols[:], " "))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate persisted metadata: %v", err)
	}
	if strings.Contains(persisted.String(), privateUA) || strings.Contains(persisted.String(), "SecretBuildToken") || strings.Contains(persisted.String(), "PrivateDeviceName") {
		t.Fatalf("persisted metadata leaked raw user-agent: %q", persisted.String())
	}
}

func TestSQLiteCalDAVOperationRepo_PrunesByCount(t *testing.T) {
	db := openOperationTestDB(t)
	repo := NewSQLiteCalDAVOperationRepoWithRetention(db, 2, 14*24*time.Hour)
	base := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		op := operationFixture("op-count-"+string(rune('a'+i)), base.Add(time.Duration(i)*time.Minute))
		if err := repo.Record(op); err != nil {
			t.Fatalf("Record(%d) error = %v", i, err)
		}
	}

	operations, err := repo.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}
	if len(operations) != 2 {
		t.Fatalf("len(operations) = %d, want 2", len(operations))
	}
	if operations[0].ID != "op-count-c" || operations[1].ID != "op-count-b" {
		t.Fatalf("unexpected retained operations: got %s, %s", operations[0].ID, operations[1].ID)
	}
}

func TestSQLiteCalDAVOperationRepo_PrunesByAge(t *testing.T) {
	db := openOperationTestDB(t)
	repo := NewSQLiteCalDAVOperationRepoWithRetention(db, 10, time.Hour)

	oldOp := operationFixture("op-old", time.Now().UTC().Add(-2*time.Hour))
	newOp := operationFixture("op-new", time.Now().UTC())
	if err := repo.Record(oldOp); err != nil {
		t.Fatalf("Record(old) error = %v", err)
	}
	if err := repo.Record(newOp); err != nil {
		t.Fatalf("Record(new) error = %v", err)
	}

	operations, err := repo.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}
	if len(operations) != 1 {
		t.Fatalf("len(operations) = %d, want 1", len(operations))
	}
	if operations[0].ID != "op-new" {
		t.Fatalf("retained operation = %q, want op-new", operations[0].ID)
	}
}

func operationFixture(id string, occurredAt time.Time) *domain.CalDAVOperation {
	return &domain.CalDAVOperation{
		ID:                id,
		OccurredAt:        occurredAt,
		Method:            "GET",
		PathPattern:       "/dav/calendars/{principal}/{calendar}/{object}.ics",
		StatusCode:        200,
		DurationMillis:    5,
		ClientFingerprint: domain.CalDAVClientUnknown,
		ETagOutcome:       domain.CalDAVETagNotApplicable,
		OperationKind:     domain.CalDAVOperationRead,
		Outcome:           domain.CalDAVOperationSuccess,
	}
}

func openOperationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := OpenDB(t.TempDir())
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	migrationsDir := filepath.Join("..", "..", "migrations")
	if err := RunMigrations(db, migrationsDir); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}
	return db
}
