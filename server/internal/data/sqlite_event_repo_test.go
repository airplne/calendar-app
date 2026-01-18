package data

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Create temp directory for test DB
	tmpDir := t.TempDir()

	db, err := OpenDB(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Run migrations - get the migrations directory path
	// The migrations are at server/migrations/
	migrationsDir := filepath.Join(getProjectRoot(t), "migrations")

	if err := RunMigrations(db, migrationsDir); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func getProjectRoot(t *testing.T) string {
	// Walk up from current directory to find project root
	// The test runs from server/internal/data/
	wd, _ := os.Getwd()
	// Go up 2 levels: data -> internal -> server
	return filepath.Join(wd, "..", "..")
}

// createTestUser creates a test user and returns the ID
func createTestUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	userRepo := NewSQLiteUserRepo(db)
	user, err := userRepo.Create(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user.ID
}

// createTestCalendar creates a test calendar and returns it
func createTestCalendar(t *testing.T, db *sql.DB, userID int64) *domain.Calendar {
	t.Helper()
	calRepo := NewSQLiteCalendarRepo(db)
	cal := &domain.Calendar{
		UserID:      userID,
		Name:        "test-calendar",
		DisplayName: "Test Calendar",
	}
	if err := calRepo.Create(context.Background(), cal); err != nil {
		t.Fatalf("Failed to create test calendar: %v", err)
	}
	return cal
}

func TestSQLiteEventRepo_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)

	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	ics := "BEGIN:VCALENDAR\nVERSION:2.0\nBEGIN:VEVENT\nUID:test-1\nSUMMARY:Test\nEND:VEVENT\nEND:VCALENDAR"

	event := &domain.Event{
		CalendarID: cal.ID,
		UID:        "test-1",
		ICS:        ics,
		Summary:    "Test Event",
		StartTime:  time.Now().Truncate(time.Second),
		EndTime:    time.Now().Add(time.Hour).Truncate(time.Second),
		ETag:       domain.GenerateETag([]byte(ics)),
		Status:     "CONFIRMED",
	}

	// Create
	if err := repo.Create(ctx, event); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if event.ID == 0 {
		t.Error("Event ID should be set after create")
	}

	// Get by UID
	got, err := repo.GetByUID(ctx, cal.ID, "test-1")
	if err != nil {
		t.Fatalf("GetByUID failed: %v", err)
	}
	if got.UID != event.UID {
		t.Errorf("UID mismatch: got %s, want %s", got.UID, event.UID)
	}
	if got.ETag != event.ETag {
		t.Errorf("ETag mismatch: got %s, want %s", got.ETag, event.ETag)
	}

	// Get by ID
	got2, err := repo.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got2.UID != event.UID {
		t.Errorf("UID mismatch via GetByID: got %s, want %s", got2.UID, event.UID)
	}
}

func TestSQLiteEventRepo_Create_Conflict(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	ics := "BEGIN:VEVENT\nUID:duplicate\nEND:VEVENT"
	event := &domain.Event{
		CalendarID: cal.ID,
		UID:        "duplicate",
		ICS:        ics,
		StartTime:  time.Now(),
		ETag:       domain.GenerateETag([]byte(ics)),
		Status:     "CONFIRMED",
	}

	// Create first event
	if err := repo.Create(ctx, event); err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Try to create duplicate
	event2 := &domain.Event{
		CalendarID: cal.ID,
		UID:        "duplicate",
		ICS:        ics,
		StartTime:  time.Now(),
		ETag:       domain.GenerateETag([]byte(ics)),
		Status:     "CONFIRMED",
	}

	err := repo.Create(ctx, event2)
	if err != domain.ErrConflict {
		t.Errorf("Expected ErrConflict, got: %v", err)
	}
}

func TestSQLiteEventRepo_Update_ETagMismatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	ics := "BEGIN:VEVENT\nUID:etag-test\nEND:VEVENT"
	event := &domain.Event{
		CalendarID: cal.ID,
		UID:        "etag-test",
		ICS:        ics,
		Summary:    "Original",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Hour),
		ETag:       domain.GenerateETag([]byte(ics)),
		Status:     "CONFIRMED",
	}

	if err := repo.Create(ctx, event); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Try to update with wrong ETag
	event.Summary = "Updated"
	err := repo.Update(ctx, event, `"wrong-etag"`)

	if err != domain.ErrPreconditionFailed {
		t.Errorf("Expected ErrPreconditionFailed, got: %v", err)
	}
}

func TestSQLiteEventRepo_Update_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	ics1 := "BEGIN:VEVENT\nUID:update-test\nSUMMARY:V1\nEND:VEVENT"
	event := &domain.Event{
		CalendarID: cal.ID,
		UID:        "update-test",
		ICS:        ics1,
		Summary:    "V1",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Hour),
		ETag:       domain.GenerateETag([]byte(ics1)),
		Status:     "CONFIRMED",
	}

	if err := repo.Create(ctx, event); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	originalETag := event.ETag

	// Update with correct ETag
	ics2 := "BEGIN:VEVENT\nUID:update-test\nSUMMARY:V2\nEND:VEVENT"
	event.ICS = ics2
	event.Summary = "V2"
	event.ETag = domain.GenerateETag([]byte(ics2))

	err := repo.Update(ctx, event, originalETag)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	got, _ := repo.GetByUID(ctx, cal.ID, "update-test")
	if got.Summary != "V2" {
		t.Errorf("Summary not updated: got %s", got.Summary)
	}
	if got.ETag != event.ETag {
		t.Errorf("ETag not updated: got %s, want %s", got.ETag, event.ETag)
	}
}

func TestSQLiteEventRepo_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	event := &domain.Event{
		CalendarID: cal.ID,
		UID:        "non-existent",
		ICS:        "BEGIN:VEVENT\nUID:non-existent\nEND:VEVENT",
		StartTime:  time.Now(),
		ETag:       `"test"`,
		Status:     "CONFIRMED",
	}

	err := repo.Update(ctx, event, `"any-etag"`)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteEventRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	event := &domain.Event{
		CalendarID: cal.ID,
		UID:        "delete-test",
		ICS:        "BEGIN:VEVENT\nUID:delete-test\nEND:VEVENT",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Hour),
		ETag:       `"test"`,
		Status:     "CONFIRMED",
	}
	repo.Create(ctx, event)

	// Delete
	if err := repo.Delete(ctx, cal.ID, "delete-test"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err := repo.GetByUID(ctx, cal.ID, "delete-test")
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestSQLiteEventRepo_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	err := repo.Delete(ctx, cal.ID, "non-existent")
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteEventRepo_GetByUID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	_, err := repo.GetByUID(ctx, cal.ID, "non-existent")
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteEventRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestSQLiteEventRepo_List_TimeRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	now := time.Now().Truncate(time.Hour)

	// Create events at different times
	events := []struct {
		uid   string
		start time.Time
		end   time.Time
	}{
		{"event-1", now.Add(-2 * time.Hour), now.Add(-1 * time.Hour)},     // Before range
		{"event-2", now, now.Add(time.Hour)},                              // In range
		{"event-3", now.Add(30 * time.Minute), now.Add(90 * time.Minute)}, // Overlaps
		{"event-4", now.Add(3 * time.Hour), now.Add(4 * time.Hour)},       // After range
	}

	for _, e := range events {
		event := &domain.Event{
			CalendarID: cal.ID,
			UID:        e.uid,
			ICS:        "BEGIN:VEVENT\nUID:" + e.uid + "\nEND:VEVENT",
			StartTime:  e.start,
			EndTime:    e.end,
			ETag:       `"test"`,
			Status:     "CONFIRMED",
		}
		repo.Create(ctx, event)
	}

	// Query range [now, now+2h]
	list, err := repo.List(ctx, cal.ID, now, now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should include event-2 and event-3 (overlapping with range)
	if len(list) != 2 {
		t.Errorf("Expected 2 events in range, got %d", len(list))
	}

	// Verify UIDs
	uids := make(map[string]bool)
	for _, ev := range list {
		uids[ev.UID] = true
	}
	if !uids["event-2"] || !uids["event-3"] {
		t.Error("Expected event-2 and event-3 in results")
	}
}

func TestSQLiteEventRepo_ListAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	now := time.Now()

	// Create multiple events
	for i := 1; i <= 3; i++ {
		event := &domain.Event{
			CalendarID: cal.ID,
			UID:        "event-" + string(rune('0'+i)),
			ICS:        "BEGIN:VEVENT\nEND:VEVENT",
			StartTime:  now.Add(time.Duration(i) * time.Hour),
			EndTime:    now.Add(time.Duration(i+1) * time.Hour),
			ETag:       `"test"`,
			Status:     "CONFIRMED",
		}
		repo.Create(ctx, event)
	}

	// List all
	list, err := repo.ListAll(ctx, cal.ID)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 events, got %d", len(list))
	}
}

func TestSQLiteEventRepo_Create_ValidationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := createTestUser(t, db)
	cal := createTestCalendar(t, db, userID)
	repo := NewSQLiteEventRepo(db)
	ctx := context.Background()

	// Event with missing UID
	event := &domain.Event{
		CalendarID: cal.ID,
		UID:        "", // Invalid
		ICS:        "BEGIN:VEVENT\nEND:VEVENT",
		StartTime:  time.Now(),
		ETag:       `"test"`,
		Status:     "CONFIRMED",
	}

	err := repo.Create(ctx, event)
	if err == nil {
		t.Error("Expected validation error, got nil")
	}
}
