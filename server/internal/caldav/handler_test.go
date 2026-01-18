package caldav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/airplne/calendar-app/server/internal/data"
	"github.com/airplne/calendar-app/server/internal/domain"
)

// caldavBase is the mount point for CalDAV endpoints (matches main.go)
const caldavBase = "/dav"

func setupTestServer(t *testing.T) (*httptest.Server, *data.SQLiteUserRepo, *data.SQLiteCalendarRepo, *data.SQLiteEventRepo) {
	t.Helper()

	tmpDir := t.TempDir()
	db, err := data.OpenDB(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Run migrations
	migrationsDir := getMigrationsDir(t)
	if err := data.RunMigrations(db, migrationsDir); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	userRepo := data.NewSQLiteUserRepo(db)
	calendarRepo := data.NewSQLiteCalendarRepo(db)
	eventRepo := data.NewSQLiteEventRepo(db)

	// Create test user
	user, err := userRepo.Create(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create default calendar
	defaultCal := &domain.Calendar{
		UserID:      user.ID,
		Name:        "default",
		DisplayName: "Calendar",
	}
	if err := calendarRepo.Create(context.Background(), defaultCal); err != nil {
		t.Fatalf("Failed to create calendar: %v", err)
	}

	// Create handler
	handler := NewHandlerWithRepos(userRepo, calendarRepo, eventRepo)

	srv := httptest.NewServer(handler)
	t.Cleanup(func() { srv.Close() })

	return srv, userRepo, calendarRepo, eventRepo
}

func getMigrationsDir(t *testing.T) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "..", "migrations")
}

func TestCalDAV_Unauthorized_Without_Auth(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	resp, err := http.Get(srv.URL + caldavBase + "/calendars/testuser/default/")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	if resp.Header.Get("WWW-Authenticate") == "" {
		t.Error("Expected WWW-Authenticate header")
	}
}

func TestCalDAV_PROPFIND_ListCalendars(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	req, _ := http.NewRequest("PROPFIND", srv.URL+caldavBase+"/calendars/testuser/", nil)
	req.SetBasicAuth("testuser", "testpass")
	req.Header.Set("Depth", "1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 207 Multi-Status
	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("Expected 207 Multi-Status, got %d", resp.StatusCode)
	}
}

func TestCalDAV_PUT_CreateEvent(t *testing.T) {
	srv, _, _, eventRepo := setupTestServer(t)

	icsData := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
UID:test-event-1
DTSTAMP:20260116T080000Z
SUMMARY:Test Event
DTSTART:20260116T090000Z
DTEND:20260116T100000Z
END:VEVENT
END:VCALENDAR`

	req, _ := http.NewRequest("PUT", srv.URL+caldavBase+"/calendars/testuser/default/test-event-1.ics",
		strings.NewReader(icsData))
	req.SetBasicAuth("testuser", "testpass")
	req.Header.Set("Content-Type", "text/calendar")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 201 Created or 204 No Content
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 201 or 204, got %d. Body: %s", resp.StatusCode, string(body))
	}

	// ETag should be returned
	if resp.Header.Get("ETag") == "" {
		t.Error("Expected ETag header in response")
	}

	// Verify event was created in database
	ctx := context.Background()
	// Get the calendar to verify event creation
	events, err := eventRepo.ListAll(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to verify event creation: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("Expected 1 event in database, got %d", len(events))
	}
}

func TestCalDAV_GET_RetrieveEvent(t *testing.T) {
	srv, _, calRepo, eventRepo := setupTestServer(t)
	ctx := context.Background()

	// Get user's calendar
	cal, err := calRepo.GetByName(ctx, 1, "default")
	if err != nil {
		t.Fatalf("Failed to get calendar: %v", err)
	}

	// Create event directly in DB
	event := &domain.Event{
		CalendarID: cal.ID,
		UID:        "get-test-event",
		ICS:        "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//Test//EN\nBEGIN:VEVENT\nUID:get-test-event\nDTSTAMP:20260116T080000Z\nSUMMARY:Get Test\nDTSTART:20260116T090000Z\nDTEND:20260116T100000Z\nEND:VEVENT\nEND:VCALENDAR",
		Summary:    "Get Test",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Hour),
		ETag:       domain.GenerateETag([]byte("test")),
		Status:     "CONFIRMED",
	}
	if err := eventRepo.Create(ctx, event); err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	req, _ := http.NewRequest("GET", srv.URL+caldavBase+"/calendars/testuser/default/get-test-event.ics", nil)
	req.SetBasicAuth("testuser", "testpass")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	// Should return ICS content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/calendar") {
		t.Errorf("Expected text/calendar content type, got %s", contentType)
	}
}

func TestCalDAV_PUT_ETagConflict(t *testing.T) {
	srv, _, calRepo, eventRepo := setupTestServer(t)
	ctx := context.Background()

	cal, err := calRepo.GetByName(ctx, 1, "default")
	if err != nil {
		t.Fatalf("Failed to get calendar: %v", err)
	}

	// Create initial event
	ics1 := "BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//Test//Test//EN\nBEGIN:VEVENT\nUID:conflict-test\nDTSTAMP:20260116T080000Z\nSUMMARY:V1\nDTSTART:20260116T090000Z\nDTEND:20260116T100000Z\nEND:VEVENT\nEND:VCALENDAR"
	event := &domain.Event{
		CalendarID: cal.ID,
		UID:        "conflict-test",
		ICS:        ics1,
		Summary:    "V1",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Hour),
		ETag:       domain.GenerateETag([]byte(ics1)),
		Status:     "CONFIRMED",
	}
	if err := eventRepo.Create(ctx, event); err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Try to update with wrong ETag
	ics2 := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
UID:conflict-test
DTSTAMP:20260116T080000Z
SUMMARY:V2
DTSTART:20260116T090000Z
DTEND:20260116T100000Z
END:VEVENT
END:VCALENDAR`

	req, _ := http.NewRequest("PUT", srv.URL+caldavBase+"/calendars/testuser/default/conflict-test.ics",
		strings.NewReader(ics2))
	req.SetBasicAuth("testuser", "testpass")
	req.Header.Set("Content-Type", "text/calendar")
	req.Header.Set("If-Match", `"wrong-etag"`)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 412 Precondition Failed
	if resp.StatusCode != http.StatusPreconditionFailed {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 412 Precondition Failed, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestCalDAV_DELETE_Event(t *testing.T) {
	srv, _, calRepo, eventRepo := setupTestServer(t)
	ctx := context.Background()

	cal, err := calRepo.GetByName(ctx, 1, "default")
	if err != nil {
		t.Fatalf("Failed to get calendar: %v", err)
	}

	// Create event
	event := &domain.Event{
		CalendarID: cal.ID,
		UID:        "delete-test",
		ICS:        "BEGIN:VEVENT\nUID:delete-test\nEND:VEVENT",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Hour),
		ETag:       `"test"`,
		Status:     "CONFIRMED",
	}
	if err := eventRepo.Create(ctx, event); err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	req, _ := http.NewRequest("DELETE", srv.URL+caldavBase+"/calendars/testuser/default/delete-test.ics", nil)
	req.SetBasicAuth("testuser", "testpass")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 204 or 200, got %d", resp.StatusCode)
	}

	// Verify deleted
	_, err = eventRepo.GetByUID(ctx, cal.ID, "delete-test")
	if err != domain.ErrNotFound {
		t.Error("Event should be deleted")
	}
}

func TestCalDAV_PROPPATCH_AppleCompatibility(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	req, _ := http.NewRequest("PROPPATCH", srv.URL+caldavBase+"/calendars/testuser/", nil)
	req.SetBasicAuth("testuser", "testpass")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 207 Multi-Status
	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("Expected 207 Multi-Status, got %d", resp.StatusCode)
	}
}
