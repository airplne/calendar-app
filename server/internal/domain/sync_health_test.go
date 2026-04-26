package domain

import (
	"testing"
	"time"
)

func TestSyncHealthEvaluator_UnknownWhenGreenSyncNeverCompleted(t *testing.T) {
	evaluator := NewDefaultSyncHealthEvaluator()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now: now,
		GreenSync: GreenSyncValidation{
			Status: GreenSyncNeverCompleted,
		},
		Operations: RecentOperationSummary{
			HasRecentOperationData: true,
		},
	})

	assertStatus(t, health, SyncHealthUnknown)
	assertReason(t, health, SyncHealthReasonGreenSyncNotCompleted)
	if health.GreenSyncCompleted {
		t.Fatal("GreenSyncCompleted should be false when validation has never completed")
	}
}

func TestSyncHealthEvaluator_UnknownWhenNoRecentOperationData(t *testing.T) {
	evaluator := NewDefaultSyncHealthEvaluator()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(-time.Hour)

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now:       now,
		GreenSync: passedGreenSync(completedAt),
		Operations: RecentOperationSummary{
			HasRecentOperationData: false,
		},
	})

	assertStatus(t, health, SyncHealthUnknown)
	assertReason(t, health, SyncHealthReasonNoRecentOperationData)
}

func TestSyncHealthEvaluator_HealthyWhenGreenSyncPassedAndNoRecentIssues(t *testing.T) {
	evaluator := NewDefaultSyncHealthEvaluator()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(-time.Hour)

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now:       now,
		GreenSync: passedGreenSync(completedAt),
		Operations: RecentOperationSummary{
			HasRecentOperationData: true,
			ETagConflicts:          5,
		},
	})

	assertStatus(t, health, SyncHealthHealthy)
	if len(health.Reasons) != 0 {
		t.Fatalf("expected no reasons for healthy state, got %+v", health.Reasons)
	}
	if !health.GreenSyncCompleted {
		t.Fatal("GreenSyncCompleted should be true when create/edit/delete validation passed")
	}
	if health.LastValidationAt == nil || !health.LastValidationAt.Equal(completedAt) {
		t.Fatalf("LastValidationAt mismatch: got %v want %v", health.LastValidationAt, completedAt)
	}
}

func TestSyncHealthEvaluator_WarningWhenValidationIsStale(t *testing.T) {
	evaluator := NewDefaultSyncHealthEvaluator()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(-(15 * 24 * time.Hour))

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now:       now,
		GreenSync: passedGreenSync(completedAt),
		Operations: RecentOperationSummary{
			HasRecentOperationData: true,
		},
	})

	assertStatus(t, health, SyncHealthWarning)
	assertReason(t, health, SyncHealthReasonValidationStale)
}

func TestSyncHealthEvaluator_WarningWhenETagConflictThresholdExceeded(t *testing.T) {
	evaluator := NewDefaultSyncHealthEvaluator()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(-time.Hour)

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now:       now,
		GreenSync: passedGreenSync(completedAt),
		Operations: RecentOperationSummary{
			HasRecentOperationData: true,
			ETagConflicts:          6,
		},
	})

	assertStatus(t, health, SyncHealthWarning)
	assertReason(t, health, SyncHealthReasonETagConflictThreshold)
}

func TestSyncHealthEvaluator_WarningWhenRecentRecoverableClientFailuresExist(t *testing.T) {
	evaluator := NewDefaultSyncHealthEvaluator()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(-time.Hour)

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now:       now,
		GreenSync: passedGreenSync(completedAt),
		Operations: RecentOperationSummary{
			HasRecentOperationData:        true,
			RecoverableClientSyncFailures: 1,
		},
	})

	assertStatus(t, health, SyncHealthWarning)
	assertReason(t, health, SyncHealthReasonRecoverableClientFailures)
}

func TestSyncHealthEvaluator_WarningWhenRecentWriteFailuresExist(t *testing.T) {
	evaluator := NewDefaultSyncHealthEvaluator()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(-time.Hour)

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now:       now,
		GreenSync: passedGreenSync(completedAt),
		Operations: RecentOperationSummary{
			HasRecentOperationData: true,
			WriteFailures:          1,
		},
	})

	assertStatus(t, health, SyncHealthWarning)
	assertReason(t, health, SyncHealthReasonRecentWriteFailures)
}

func TestSyncHealthEvaluator_CriticalWhenCorruptICSDetected(t *testing.T) {
	health := evaluateCriticalCondition(t, RecentOperationSummary{CorruptICSIncidents: 1}, GreenSyncValidationStatus(""))

	assertReason(t, health, SyncHealthReasonCorruptICSDetected)
}

func TestSyncHealthEvaluator_CriticalWhenDuplicateUIDUnresolved(t *testing.T) {
	health := evaluateCriticalCondition(t, RecentOperationSummary{UnresolvedDuplicateUIDs: 1}, GreenSyncValidationStatus(""))

	assertReason(t, health, SyncHealthReasonDuplicateUIDUnresolved)
}

func TestSyncHealthEvaluator_CriticalWhenRoundtripValidationFailed(t *testing.T) {
	health := evaluateCriticalCondition(t, RecentOperationSummary{RoundtripValidationFailed: true}, GreenSyncValidationStatus(""))

	assertReason(t, health, SyncHealthReasonRoundtripValidationFailed)
}

func TestSyncHealthEvaluator_CriticalWhenGreenSyncValidationFailed(t *testing.T) {
	health := evaluateCriticalCondition(t, RecentOperationSummary{}, GreenSyncFailed)

	assertReason(t, health, SyncHealthReasonGreenSyncFailed)
}

func TestSyncHealthEvaluator_CriticalWhenCalendarWritePathFailing(t *testing.T) {
	health := evaluateCriticalCondition(t, RecentOperationSummary{CalendarWritePathFailing: true}, GreenSyncValidationStatus(""))

	assertReason(t, health, SyncHealthReasonCalendarWritePathFailing)
}

func TestSyncHealthEvaluator_CriticalBeatsWarningAndUnknown(t *testing.T) {
	evaluator := NewDefaultSyncHealthEvaluator()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now: now,
		GreenSync: GreenSyncValidation{
			Status: GreenSyncNeverCompleted,
		},
		Operations: RecentOperationSummary{
			HasRecentOperationData: false,
			ETagConflicts:          10,
			CorruptICSIncidents:    1,
		},
	})

	assertStatus(t, health, SyncHealthCritical)
	assertReason(t, health, SyncHealthReasonCorruptICSDetected)
	assertNoReason(t, health, SyncHealthReasonGreenSyncNotCompleted)
	assertNoReason(t, health, SyncHealthReasonETagConflictThreshold)
}

func TestSyncHealthEvaluator_ReturnsReasonCodesForNonHealthyStates(t *testing.T) {
	evaluator := NewDefaultSyncHealthEvaluator()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(-time.Hour)

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now:       now,
		GreenSync: passedGreenSync(completedAt),
		Operations: RecentOperationSummary{
			HasRecentOperationData:        true,
			WriteFailures:                 1,
			RecoverableClientSyncFailures: 1,
		},
	})

	assertStatus(t, health, SyncHealthWarning)
	if len(health.Reasons) == 0 {
		t.Fatal("expected non-healthy status to include reasons")
	}
	for _, reason := range health.Reasons {
		if reason.Code == "" {
			t.Fatalf("expected reason code, got empty reason: %+v", reason)
		}
		if reason.Message == "" {
			t.Fatalf("expected reason message, got empty reason: %+v", reason)
		}
		if reason.Severity == "" {
			t.Fatalf("expected reason severity, got empty reason: %+v", reason)
		}
	}
}

func TestSyncHealthEvaluator_CustomThresholds(t *testing.T) {
	evaluator := NewSyncHealthEvaluator(SyncHealthEvaluationConfig{
		ETagConflictWarningThreshold: 2,
		ValidationStaleAfter:         48 * time.Hour,
	})
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(-(49 * time.Hour))

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now:       now,
		GreenSync: passedGreenSync(completedAt),
		Operations: RecentOperationSummary{
			HasRecentOperationData: true,
			ETagConflicts:          3,
		},
	})

	assertStatus(t, health, SyncHealthWarning)
	assertReason(t, health, SyncHealthReasonETagConflictThreshold)
	assertReason(t, health, SyncHealthReasonValidationStale)
}

func passedGreenSync(completedAt time.Time) GreenSyncValidation {
	return GreenSyncValidation{
		Status:           GreenSyncPassed,
		CompletedAt:      &completedAt,
		CreateStepPassed: true,
		EditStepPassed:   true,
		DeleteStepPassed: true,
	}
}

func evaluateCriticalCondition(t *testing.T, ops RecentOperationSummary, greenStatus GreenSyncValidationStatus) SyncHealth {
	t.Helper()
	evaluator := NewDefaultSyncHealthEvaluator()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(-time.Hour)
	greenSync := passedGreenSync(completedAt)
	if greenStatus != "" {
		greenSync = GreenSyncValidation{Status: greenStatus}
	}
	ops.HasRecentOperationData = true

	health := evaluator.Evaluate(SyncHealthEvaluationInput{
		Now:        now,
		GreenSync:  greenSync,
		Operations: ops,
	})

	assertStatus(t, health, SyncHealthCritical)
	return health
}

func assertStatus(t *testing.T, health SyncHealth, want SyncHealthStatus) {
	t.Helper()
	if health.Status != want {
		t.Fatalf("status mismatch: got %s, want %s; reasons=%+v", health.Status, want, health.Reasons)
	}
}

func assertReason(t *testing.T, health SyncHealth, code string) {
	t.Helper()
	for _, reason := range health.Reasons {
		if reason.Code == code {
			return
		}
	}
	t.Fatalf("expected reason %q in %+v", code, health.Reasons)
}

func assertNoReason(t *testing.T, health SyncHealth, code string) {
	t.Helper()
	for _, reason := range health.Reasons {
		if reason.Code == code {
			t.Fatalf("did not expect reason %q in %+v", code, health.Reasons)
		}
	}
}
