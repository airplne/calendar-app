package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

type fakeDebugOperationLister struct {
	operations []*domain.CalDAVOperation
	err        error
}

func (f fakeDebugOperationLister) ListRecent(ctx context.Context, limit int) ([]*domain.CalDAVOperation, error) {
	if f.err != nil {
		return nil, f.err
	}
	if limit > 0 && len(f.operations) > limit {
		return f.operations[:limit], nil
	}
	return f.operations, nil
}

func TestDebugBundleBuildsRedactedDiagnostics(t *testing.T) {
	t.Setenv("CALENDARAPP_PASS", "super-secret-password")
	t.Setenv("TODOIST_TOKEN", "todoist-secret-token")
	t.Setenv("OPENAI_API_KEY", "llm-secret-key")

	privateFragments := []string{
		"BEGIN:VCALENDAR",
		"Sensitive Doctor Appointment",
		"Detailed private description",
		"attendee@example.com",
		"private-event-1",
		"testuser",
		"default",
		"PrivateDeviceName",
		"SecretBuildToken",
		"super-secret-password",
		"todoist-secret-token",
		"llm-secret-key",
	}

	now := time.Now().UTC()
	ops := []*domain.CalDAVOperation{{
		OccurredAt:        now,
		Method:            "PUT",
		PathPattern:       "/dav/calendars/{principal}/{calendar}/{object}.ics",
		StatusCode:        412,
		DurationMillis:    17,
		ClientFingerprint: domain.CalDAVClientFantastical,
		ETagOutcome:       domain.CalDAVETagMismatched,
		OperationKind:     domain.CalDAVOperationWrite,
		Outcome:           domain.CalDAVOperationRecoverableFailure,
		ErrorCode:         domain.CalDAVErrorETagConflict,
		RedactedError:     "ETag precondition failed.",
	}}
	service := NewDebugBundleService(
		NewSyncHealthService(fakeDebugOperationLister{operations: ops}, UnknownGreenSyncProvider()),
		DebugBundleOptions{Version: "test", Environment: "development", DataDir: "/private/data", MigrationsDir: "/private/migrations", GeneratedBy: "testuser"},
	)

	bundle, err := service.Build(context.Background())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if bundle.SchemaVersion != DebugBundleSchemaVersion {
		t.Fatalf("SchemaVersion = %q, want %q", bundle.SchemaVersion, DebugBundleSchemaVersion)
	}
	if bundle.Redaction.RawICSIncluded || bundle.Redaction.RawUserAgentIncluded || bundle.Redaction.SecretsIncluded {
		t.Fatalf("unexpected raw-data flags: %+v", bundle.Redaction)
	}
	if bundle.SyncHealth.Status != string(domain.SyncHealthUnknown) {
		t.Fatalf("sync status = %q, want unknown", bundle.SyncHealth.Status)
	}
	if len(bundle.CalDAV.RecentOperations) != 1 {
		t.Fatalf("recent operations = %d, want 1", len(bundle.CalDAV.RecentOperations))
	}
	if bundle.CalDAV.RecentOperations[0].ClientFingerprint != domain.CalDAVClientFantastical {
		t.Fatalf("client fingerprint = %q", bundle.CalDAV.RecentOperations[0].ClientFingerprint)
	}
	if len(bundle.ErrorTraces) != 1 || bundle.ErrorTraces[0].ErrorCode != string(domain.CalDAVErrorETagConflict) {
		t.Fatalf("error traces = %+v", bundle.ErrorTraces)
	}

	payload, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	body := string(payload)
	for _, fragment := range privateFragments {
		if strings.Contains(body, fragment) {
			t.Fatalf("debug bundle leaked private fragment %q in %s", fragment, body)
		}
	}
	for _, secretName := range []string{"CALENDARAPP_PASS", "TODOIST_TOKEN", "OPENAI_API_KEY"} {
		if !strings.Contains(body, secretName) {
			t.Fatalf("debug bundle should include redacted secret name %q for diagnostics: %s", secretName, body)
		}
	}
}

func TestDebugBundleBuildReturnsServiceError(t *testing.T) {
	service := NewDebugBundleService(
		NewSyncHealthService(fakeDebugOperationLister{err: errors.New("database contains private-event-1")}, UnknownGreenSyncProvider()),
		DebugBundleOptions{},
	)

	_, err := service.Build(context.Background())
	if err == nil {
		t.Fatal("Build() error = nil, want error")
	}
}

func TestDebugBundleSchemaVersion(t *testing.T) {
	if DebugBundleSchemaVersion != "debug_bundle.v1" {
		t.Fatalf("DebugBundleSchemaVersion = %q", DebugBundleSchemaVersion)
	}
}
