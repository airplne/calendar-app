package caldav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/airplne/calendar-app/server/internal/data"
	"github.com/airplne/calendar-app/server/internal/domain"
)

func TestNormalizeClientFingerprint(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      string
	}{
		{name: "apple calendar", userAgent: "Mac+OS+X/14.0 (23A344) CalendarAgent/1000 DataAccess/1", want: domain.CalDAVClientAppleCalendar},
		{name: "fantastical", userAgent: "Fantastical/4.0 CFNetwork/1490", want: domain.CalDAVClientFantastical},
		{name: "davx5 ascii", userAgent: "DAVx5/4.4.1-ose (2024/01)", want: domain.CalDAVClientDAVx5},
		{name: "davx5 superscript", userAgent: "DAVx⁵/4.4.1", want: domain.CalDAVClientDAVx5},
		{name: "thunderbird", userAgent: "Mozilla/5.0 Thunderbird/115.0", want: domain.CalDAVClientThunderbird},
		{name: "unknown", userAgent: "CustomSyncClient/1.0", want: domain.CalDAVClientUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeClientFingerprint(tt.userAgent); got != tt.want {
				t.Fatalf("NormalizeClientFingerprint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRedactCalDAVPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "well known", path: "/.well-known/caldav", want: "/.well-known/caldav"},
		{name: "dav root", path: "/dav/", want: "/dav/"},
		{name: "calendar collection", path: "/dav/calendars/alice/private-work/", want: "/dav/calendars/{principal}/{calendar}/"},
		{name: "calendar object", path: "/dav/calendars/alice/private-work/doctor-visit-123.ics", want: "/dav/calendars/{principal}/{calendar}/{object}.ics"},
		{name: "prefix stripped object", path: "/calendars/alice/private-work/doctor-visit-123.ics", want: "/calendars/{principal}/{calendar}/{object}.ics"},
		{name: "principal", path: "/dav/principals/alice/", want: "/dav/principals/{principal}/"},
		{name: "unknown dav resource", path: "/dav/some/private/value", want: "/dav/{resource}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactCalDAVPath(tt.path)
			if got != tt.want {
				t.Fatalf("RedactCalDAVPath() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "alice") || strings.Contains(got, "doctor") || strings.Contains(got, "private-work") {
				t.Fatalf("redacted path leaked private path data: %q", got)
			}
		})
	}
}

func TestOperationClassification(t *testing.T) {
	readOp := BuildCalDAVOperation("GET", "/dav/calendars/user/default/event.ics", http.StatusOK, 10*time.Millisecond, http.Header{}, http.Header{}, "Custom/1", 0, 100)
	if readOp.OperationKind != domain.CalDAVOperationRead {
		t.Fatalf("GET kind = %q, want read", readOp.OperationKind)
	}
	if readOp.Outcome != domain.CalDAVOperationSuccess {
		t.Fatalf("GET outcome = %q, want success", readOp.Outcome)
	}

	writeOp := BuildCalDAVOperation("PUT", "/dav/calendars/user/default/event.ics", http.StatusCreated, 10*time.Millisecond, http.Header{}, http.Header{"ETag": []string{"abc"}}, "Custom/1", 200, 0)
	if writeOp.OperationKind != domain.CalDAVOperationWrite {
		t.Fatalf("PUT kind = %q, want write", writeOp.OperationKind)
	}
	if writeOp.ETagOutcome != domain.CalDAVETagGenerated {
		t.Fatalf("PUT ETag outcome = %q, want generated", writeOp.ETagOutcome)
	}

	conflictOp := BuildCalDAVOperation("PUT", "/dav/calendars/user/default/event.ics", http.StatusPreconditionFailed, 10*time.Millisecond, http.Header{"If-Match": []string{"wrong"}}, http.Header{}, "Custom/1", 200, 0)
	if conflictOp.Outcome != domain.CalDAVOperationRecoverableFailure {
		t.Fatalf("412 outcome = %q, want recoverable_failure", conflictOp.Outcome)
	}
	if conflictOp.ErrorCode != domain.CalDAVErrorETagConflict {
		t.Fatalf("412 error = %q, want etag_conflict", conflictOp.ErrorCode)
	}
	if conflictOp.ETagOutcome != domain.CalDAVETagMismatched {
		t.Fatalf("412 ETag outcome = %q, want mismatched", conflictOp.ETagOutcome)
	}

	parseOp := domain.CalDAVOperation{Outcome: ClassifyOperationOutcome(domain.CalDAVOperationRead, http.StatusBadRequest, domain.CalDAVErrorParse)}
	if parseOp.Outcome != domain.CalDAVOperationIntegrityFailure {
		t.Fatalf("parse outcome = %q, want integrity_failure", parseOp.Outcome)
	}
}

func TestOperationMetadataMiddlewareRecordsRedactedRequest(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	db, err := data.OpenDB(tmpDir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := data.RunMigrations(db, getMigrationsDir(t)); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	userRepo := data.NewSQLiteUserRepo(db)
	calendarRepo := data.NewSQLiteCalendarRepo(db)
	eventRepo := data.NewSQLiteEventRepo(db)
	operationRepo := data.NewSQLiteCalDAVOperationRepo(db)
	user, err := userRepo.Create(ctx, "testuser")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := calendarRepo.Create(ctx, &domain.Calendar{UserID: user.ID, Name: "default", DisplayName: "Calendar"}); err != nil {
		t.Fatalf("create calendar: %v", err)
	}

	handler := NewHandlerWithReposAndOperationRecorder(db, userRepo, calendarRepo, eventRepo, operationRepo)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	icsData := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
UID:private-event-1
DTSTAMP:20260116T080000Z
SUMMARY:Sensitive Doctor Appointment
DTSTART:20260116T090000Z
DTEND:20260116T100000Z
END:VEVENT
END:VCALENDAR`

	req, err := http.NewRequest("PUT", srv.URL+"/dav/calendars/testuser/default/private-event-1.ics", strings.NewReader(icsData))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.SetBasicAuth("testuser", "testpass")
	req.Header.Set("Content-Type", "text/calendar")
	req.Header.Set("User-Agent", "Fantastical/4.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT status = %d, want 201/204", resp.StatusCode)
	}

	operations, err := operationRepo.ListRecent(ctx, 10)
	if err != nil {
		t.Fatalf("list operations: %v", err)
	}
	if len(operations) == 0 {
		t.Fatal("expected at least one operation record")
	}
	op := operations[0]
	if op.Method != "PUT" {
		t.Fatalf("method = %q, want PUT", op.Method)
	}
	if op.ClientFingerprint != domain.CalDAVClientFantastical {
		t.Fatalf("client = %q, want fantastical", op.ClientFingerprint)
	}
	if op.OperationKind != domain.CalDAVOperationWrite {
		t.Fatalf("kind = %q, want write", op.OperationKind)
	}
	if op.PathPattern != "/dav/calendars/{principal}/{calendar}/{object}.ics" {
		t.Fatalf("path pattern = %q", op.PathPattern)
	}

	serialized := strings.Join([]string{op.PathPattern, op.RedactedError, op.ClientFingerprint, op.Method}, " ")
	for _, forbidden := range []string{"BEGIN:VCALENDAR", "Sensitive Doctor Appointment", "private-event-1", "testuser", "default"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("operation metadata leaked private content %q in %q", forbidden, serialized)
		}
	}
}
