package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/airplne/calendar-app/server/internal/caldav"
	"github.com/airplne/calendar-app/server/internal/domain"
	"github.com/airplne/calendar-app/server/internal/services"
)

type fakeDebugAPIOperationLister struct {
	operations []*domain.CalDAVOperation
	err        error
}

func (f fakeDebugAPIOperationLister) ListRecent(ctx context.Context, limit int) ([]*domain.CalDAVOperation, error) {
	if f.err != nil {
		return nil, f.err
	}
	if limit > 0 && len(f.operations) > limit {
		return f.operations[:limit], nil
	}
	return f.operations, nil
}

func TestDebugBundleAPIRequiresBasicAuth(t *testing.T) {
	handler := authenticatedDebugBundleHandler(fakeDebugAPIOperationLister{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug-bundle", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestDebugBundleAPIReturnsRedactedBundle(t *testing.T) {
	handler := authenticatedDebugBundleHandler(fakeDebugAPIOperationLister{operations: []*domain.CalDAVOperation{{
		OccurredAt:        time.Now().UTC(),
		Method:            "PUT",
		PathPattern:       "/dav/calendars/{principal}/{calendar}/{object}.ics",
		StatusCode:        412,
		DurationMillis:    10,
		ClientFingerprint: domain.CalDAVClientDAVx5,
		ETagOutcome:       domain.CalDAVETagMismatched,
		OperationKind:     domain.CalDAVOperationWrite,
		Outcome:           domain.CalDAVOperationRecoverableFailure,
		ErrorCode:         domain.CalDAVErrorETagConflict,
		RedactedError:     "ETag precondition failed.",
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug-bundle", nil)
	req.SetBasicAuth("admin", "secret")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.Contains(cd, "calendar-app-debug-bundle.json") {
		t.Fatalf("content-disposition = %q", cd)
	}

	body := rr.Body.String()
	for _, fragment := range []string{
		"BEGIN:VCALENDAR",
		"Sensitive Doctor Appointment",
		"Detailed private description",
		"attendee@example.com",
		"private-event-1",
		"PrivateDeviceName",
		"SecretBuildToken",
		"Authorization",
		"secret",
	} {
		if strings.Contains(body, fragment) {
			t.Fatalf("debug bundle leaked private fragment %q in %s", fragment, body)
		}
	}

	var bundle services.DebugBundle
	if err := json.Unmarshal(rr.Body.Bytes(), &bundle); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	if bundle.SchemaVersion != services.DebugBundleSchemaVersion {
		t.Fatalf("schema_version = %q", bundle.SchemaVersion)
	}
	if bundle.Redaction.RawICSIncluded || bundle.Redaction.RawUserAgentIncluded || bundle.Redaction.SecretsIncluded {
		t.Fatalf("unexpected redaction flags: %+v", bundle.Redaction)
	}
	if len(bundle.CalDAV.RecentOperations) != 1 {
		t.Fatalf("recent operations = %d, want 1", len(bundle.CalDAV.RecentOperations))
	}
	if bundle.CalDAV.RecentOperations[0].ClientFingerprint != domain.CalDAVClientDAVx5 {
		t.Fatalf("client fingerprint = %q", bundle.CalDAV.RecentOperations[0].ClientFingerprint)
	}
}

func TestDebugBundleAPIReturnsSafeGenericError(t *testing.T) {
	handler := authenticatedDebugBundleHandler(fakeDebugAPIOperationLister{err: errors.New("database failed for private-event-1 BEGIN:VCALENDAR")})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug-bundle", nil)
	req.SetBasicAuth("admin", "secret")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "debug_bundle_unavailable") {
		t.Fatalf("body missing safe error code: %s", body)
	}
	for _, fragment := range []string{"private-event-1", "BEGIN:VCALENDAR", "database failed"} {
		if strings.Contains(body, fragment) {
			t.Fatalf("error response leaked private fragment %q in %s", fragment, body)
		}
	}
}

func authenticatedDebugBundleHandler(lister fakeDebugAPIOperationLister) http.Handler {
	syncService := services.NewSyncHealthService(lister, services.UnknownGreenSyncProvider())
	debugService := services.NewDebugBundleService(syncService, services.DebugBundleOptions{Version: "test", Environment: "test", GeneratedBy: "admin"})
	return caldav.BasicAuthMiddleware(caldav.AuthConfig{Username: "admin", Password: "secret"}, fakeDebugUserRepo{})(NewDebugBundleHandler(debugService))
}

type fakeDebugUserRepo struct{}

func (fakeDebugUserRepo) Create(ctx context.Context, username string) (*domain.User, error) {
	return &domain.User{ID: 1, Username: username}, nil
}

func (fakeDebugUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return &domain.User{ID: id, Username: "admin"}, nil
}

func (fakeDebugUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if username != "admin" {
		return nil, domain.ErrNotFound
	}
	return &domain.User{ID: 1, Username: username}, nil
}
