package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
	"github.com/airplne/calendar-app/server/internal/services"
)

type fakeAPIOperationLister struct {
	operations []*domain.CalDAVOperation
}

func (f fakeAPIOperationLister) ListRecent(ctx context.Context, limit int) ([]*domain.CalDAVOperation, error) {
	if limit > 0 && len(f.operations) > limit {
		return f.operations[:limit], nil
	}
	return f.operations, nil
}

func TestSyncHealthAPIUnknownWhenNoData(t *testing.T) {
	handler := NewSyncHealthHandler(services.NewSyncHealthService(fakeAPIOperationLister{}, services.UnknownGreenSyncProvider()))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != string(domain.SyncHealthUnknown) {
		t.Fatalf("status = %v, want unknown", body["status"])
	}
	if body["green_sync_completed"] != false {
		t.Fatalf("green_sync_completed = %v, want false", body["green_sync_completed"])
	}
}

func TestSyncHealthAPIOperationsAreRedacted(t *testing.T) {
	privateFragments := []string{
		"BEGIN:VCALENDAR",
		"Sensitive Doctor Appointment",
		"private-event-1",
		"testuser",
		"default",
		"PrivateDeviceName",
		"SecretBuildToken",
	}
	ops := []*domain.CalDAVOperation{{
		OccurredAt:        time.Now().UTC(),
		Method:            "PUT",
		PathPattern:       "/dav/calendars/{principal}/{calendar}/{object}.ics",
		StatusCode:        412,
		DurationMillis:    20,
		ClientFingerprint: domain.CalDAVClientFantastical,
		ETagOutcome:       domain.CalDAVETagMismatched,
		OperationKind:     domain.CalDAVOperationWrite,
		Outcome:           domain.CalDAVOperationRecoverableFailure,
		ErrorCode:         domain.CalDAVErrorETagConflict,
		RedactedError:     "ETag precondition failed.",
	}}
	handler := NewSyncHealthHandler(services.NewSyncHealthService(fakeAPIOperationLister{operations: ops}, services.UnknownGreenSyncProvider()))
	req := httptest.NewRequest(http.MethodGet, "/operations", nil)
	rr := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, string(domain.CalDAVClientFantastical)) {
		t.Fatalf("response missing normalized fingerprint: %s", body)
	}
	for _, fragment := range privateFragments {
		if strings.Contains(body, fragment) {
			t.Fatalf("response leaked private fragment %q in %s", fragment, body)
		}
	}
}

func TestSyncHealthAPIClients(t *testing.T) {
	now := time.Now().UTC()
	handler := NewSyncHealthHandler(services.NewSyncHealthService(fakeAPIOperationLister{operations: []*domain.CalDAVOperation{
		{OccurredAt: now, ClientFingerprint: domain.CalDAVClientAppleCalendar, Outcome: domain.CalDAVOperationSuccess},
		{OccurredAt: now.Add(-time.Minute), ClientFingerprint: domain.CalDAVClientAppleCalendar, Outcome: domain.CalDAVOperationSuccess},
		{OccurredAt: now.Add(-2 * time.Minute), ClientFingerprint: domain.CalDAVClientThunderbird, Outcome: domain.CalDAVOperationSuccess},
	}}, services.UnknownGreenSyncProvider()))
	req := httptest.NewRequest(http.MethodGet, "/clients", nil)
	rr := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "apple-calendar") || !strings.Contains(body, "Apple Calendar") {
		t.Fatalf("response missing Apple Calendar client: %s", body)
	}
	if !strings.Contains(body, "thunderbird") || !strings.Contains(body, "Thunderbird") {
		t.Fatalf("response missing Thunderbird client: %s", body)
	}
}

func TestSyncHealthValidationEndpointsPlaceholder(t *testing.T) {
	handler := NewSyncHealthHandler(services.NewSyncHealthService(fakeAPIOperationLister{}, services.UnknownGreenSyncProvider()))
	for _, path := range []string{"/validation-event/start", "/validation-event/verify"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rr := httptest.NewRecorder()

		handler.Routes().ServeHTTP(rr, req)

		if rr.Code != http.StatusNotImplemented {
			t.Fatalf("%s status = %d, want 501; body=%s", path, rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "green_sync_validation_not_implemented") {
			t.Fatalf("%s response missing placeholder error: %s", path, rr.Body.String())
		}
	}
}
