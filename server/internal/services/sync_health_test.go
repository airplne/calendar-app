package services

import (
	"context"
	"testing"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

type fakeOperationLister struct {
	operations []*domain.CalDAVOperation
}

func (f fakeOperationLister) ListRecent(ctx context.Context, limit int) ([]*domain.CalDAVOperation, error) {
	if limit > 0 && len(f.operations) > limit {
		return f.operations[:limit], nil
	}
	return f.operations, nil
}

func TestSyncHealthServiceUnknownWithNoDataAndNoGreenSync(t *testing.T) {
	service := NewSyncHealthService(fakeOperationLister{}, UnknownGreenSyncProvider())

	summary, err := service.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}

	if summary.Health.Status != domain.SyncHealthUnknown {
		t.Fatalf("status = %q, want unknown", summary.Health.Status)
	}
	assertReason(t, summary.Health.Reasons, domain.SyncHealthReasonGreenSyncNotCompleted)
	assertReason(t, summary.Health.Reasons, domain.SyncHealthReasonNoRecentOperationData)
}

func TestSyncHealthServiceHealthyWithPassedGreenSyncAndSuccessfulOperation(t *testing.T) {
	now := time.Now().UTC()
	service := NewSyncHealthService(
		fakeOperationLister{operations: []*domain.CalDAVOperation{{
			OccurredAt:        now,
			Method:            "GET",
			StatusCode:        200,
			DurationMillis:    12,
			ClientFingerprint: domain.CalDAVClientAppleCalendar,
			OperationKind:     domain.CalDAVOperationRead,
			Outcome:           domain.CalDAVOperationSuccess,
		}}},
		StaticGreenSyncProvider{Validation: passedGreenSync(now)},
	)

	summary, err := service.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Health.Status != domain.SyncHealthHealthy {
		t.Fatalf("status = %q, want healthy; reasons=%+v", summary.Health.Status, summary.Health.Reasons)
	}
	if summary.OperationCounts.Success != 1 || summary.OperationCounts.Total != 1 {
		t.Fatalf("counts = %+v", summary.OperationCounts)
	}
	if len(summary.Clients) != 1 || summary.Clients[0].Fingerprint != domain.CalDAVClientAppleCalendar {
		t.Fatalf("clients = %+v", summary.Clients)
	}
}

func TestSyncHealthServiceWarningForETagConflicts(t *testing.T) {
	now := time.Now().UTC()
	ops := make([]*domain.CalDAVOperation, 0, 6)
	for i := 0; i < 6; i++ {
		ops = append(ops, &domain.CalDAVOperation{
			OccurredAt:        now.Add(time.Duration(-i) * time.Minute),
			Method:            "PUT",
			StatusCode:        412,
			ClientFingerprint: domain.CalDAVClientFantastical,
			ETagOutcome:       domain.CalDAVETagMismatched,
			OperationKind:     domain.CalDAVOperationWrite,
			Outcome:           domain.CalDAVOperationRecoverableFailure,
			ErrorCode:         domain.CalDAVErrorETagConflict,
		})
	}
	service := NewSyncHealthService(fakeOperationLister{operations: ops}, StaticGreenSyncProvider{Validation: passedGreenSync(now)})

	summary, err := service.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Health.Status != domain.SyncHealthWarning {
		t.Fatalf("status = %q, want warning; reasons=%+v", summary.Health.Status, summary.Health.Reasons)
	}
	if summary.OperationCounts.ETagConflicts != 6 {
		t.Fatalf("ETagConflicts = %d, want 6", summary.OperationCounts.ETagConflicts)
	}
	assertReason(t, summary.Health.Reasons, domain.SyncHealthReasonETagConflictThreshold)
}

func TestSyncHealthServiceCriticalForCorruptICSAndDuplicateUID(t *testing.T) {
	now := time.Now().UTC()
	service := NewSyncHealthService(
		fakeOperationLister{operations: []*domain.CalDAVOperation{
			{
				OccurredAt:        now,
				Method:            "GET",
				StatusCode:        500,
				ClientFingerprint: domain.CalDAVClientDAVx5,
				OperationKind:     domain.CalDAVOperationRead,
				Outcome:           domain.CalDAVOperationIntegrityFailure,
				ErrorCode:         domain.CalDAVErrorCorruptICS,
			},
			{
				OccurredAt:        now.Add(-time.Minute),
				Method:            "PUT",
				StatusCode:        409,
				ClientFingerprint: domain.CalDAVClientDAVx5,
				OperationKind:     domain.CalDAVOperationWrite,
				Outcome:           domain.CalDAVOperationIntegrityFailure,
				ErrorCode:         domain.CalDAVErrorDuplicateUID,
			},
		}},
		StaticGreenSyncProvider{Validation: passedGreenSync(now)},
	)

	summary, err := service.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Health.Status != domain.SyncHealthCritical {
		t.Fatalf("status = %q, want critical; reasons=%+v", summary.Health.Status, summary.Health.Reasons)
	}
	if summary.OperationCounts.CorruptICSIncidents != 1 || summary.OperationCounts.DuplicateUIDAttempts != 1 {
		t.Fatalf("counts = %+v", summary.OperationCounts)
	}
	assertReason(t, summary.Health.Reasons, domain.SyncHealthReasonCorruptICSDetected)
	assertReason(t, summary.Health.Reasons, domain.SyncHealthReasonDuplicateUIDUnresolved)
}

func passedGreenSync(completedAt time.Time) domain.GreenSyncValidation {
	return domain.GreenSyncValidation{
		Status:           domain.GreenSyncPassed,
		CompletedAt:      &completedAt,
		CreateStepPassed: true,
		EditStepPassed:   true,
		DeleteStepPassed: true,
	}
}

func assertReason(t *testing.T, reasons []domain.SyncHealthReason, code string) {
	t.Helper()
	for _, reason := range reasons {
		if reason.Code == code {
			return
		}
	}
	t.Fatalf("expected reason %q in %+v", code, reasons)
}
