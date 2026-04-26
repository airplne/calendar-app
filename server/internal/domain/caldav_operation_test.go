package domain

import "testing"

func TestSummarizeCalDAVOperations_CorruptICS(t *testing.T) {
	summary := SummarizeCalDAVOperations([]*CalDAVOperation{{
		OperationKind: CalDAVOperationRead,
		Outcome:       CalDAVOperationIntegrityFailure,
		ErrorCode:     CalDAVErrorCorruptICS,
	}})

	if summary.CorruptICSIncidents != 1 {
		t.Fatalf("CorruptICSIncidents = %d, want 1", summary.CorruptICSIncidents)
	}
	if summary.UnresolvedDuplicateUIDs != 0 {
		t.Fatalf("UnresolvedDuplicateUIDs = %d, want 0", summary.UnresolvedDuplicateUIDs)
	}
}

func TestSummarizeCalDAVOperations_ParseErrorCountsAsCorruptICS(t *testing.T) {
	summary := SummarizeCalDAVOperations([]*CalDAVOperation{{
		OperationKind: CalDAVOperationRead,
		Outcome:       CalDAVOperationIntegrityFailure,
		ErrorCode:     CalDAVErrorParse,
	}})

	if summary.CorruptICSIncidents != 1 {
		t.Fatalf("CorruptICSIncidents = %d, want 1", summary.CorruptICSIncidents)
	}
}

func TestSummarizeCalDAVOperations_DuplicateUIDDoesNotCountAsCorruptICS(t *testing.T) {
	summary := SummarizeCalDAVOperations([]*CalDAVOperation{{
		OperationKind: CalDAVOperationWrite,
		Outcome:       CalDAVOperationIntegrityFailure,
		ErrorCode:     CalDAVErrorDuplicateUID,
	}})

	if summary.UnresolvedDuplicateUIDs != 1 {
		t.Fatalf("UnresolvedDuplicateUIDs = %d, want 1", summary.UnresolvedDuplicateUIDs)
	}
	if summary.CorruptICSIncidents != 0 {
		t.Fatalf("CorruptICSIncidents = %d, want 0", summary.CorruptICSIncidents)
	}
}

func TestSummarizeCalDAVOperations_ETagConflictCountsOnce(t *testing.T) {
	summary := SummarizeCalDAVOperations([]*CalDAVOperation{{
		StatusCode:    412,
		OperationKind: CalDAVOperationWrite,
		Outcome:       CalDAVOperationRecoverableFailure,
		ErrorCode:     CalDAVErrorETagConflict,
		ETagOutcome:   CalDAVETagMismatched,
	}})

	if summary.ETagConflicts != 1 {
		t.Fatalf("ETagConflicts = %d, want 1", summary.ETagConflicts)
	}
	if summary.CorruptICSIncidents != 0 {
		t.Fatalf("CorruptICSIncidents = %d, want 0", summary.CorruptICSIncidents)
	}
}

func TestSummarizeCalDAVOperations_GenericWriteFailureDoesNotCountAsCorruptICS(t *testing.T) {
	summary := SummarizeCalDAVOperations([]*CalDAVOperation{{
		StatusCode:    500,
		OperationKind: CalDAVOperationWrite,
		Outcome:       CalDAVOperationIntegrityFailure,
		ErrorCode:     CalDAVErrorWriteFailed,
	}})

	if summary.WriteFailures != 1 {
		t.Fatalf("WriteFailures = %d, want 1", summary.WriteFailures)
	}
	if !summary.CalendarWritePathFailing {
		t.Fatal("CalendarWritePathFailing = false, want true")
	}
	if summary.CorruptICSIncidents != 0 {
		t.Fatalf("CorruptICSIncidents = %d, want 0", summary.CorruptICSIncidents)
	}
}

func TestSummarizeCalDAVOperations_Generic5xxIntegrityFailureDoesNotCountAsCorruptICS(t *testing.T) {
	summary := SummarizeCalDAVOperations([]*CalDAVOperation{{
		StatusCode:    500,
		OperationKind: CalDAVOperationRead,
		Outcome:       CalDAVOperationIntegrityFailure,
		ErrorCode:     CalDAVErrorUnknown,
	}})

	if summary.CorruptICSIncidents != 0 {
		t.Fatalf("CorruptICSIncidents = %d, want 0", summary.CorruptICSIncidents)
	}
	if summary.UnresolvedDuplicateUIDs != 0 {
		t.Fatalf("UnresolvedDuplicateUIDs = %d, want 0", summary.UnresolvedDuplicateUIDs)
	}
}
