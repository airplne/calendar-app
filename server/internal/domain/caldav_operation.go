package domain

import "time"

// CalDAVOperationKind describes whether a CalDAV request primarily reads or writes data.
type CalDAVOperationKind string

const (
	CalDAVOperationRead  CalDAVOperationKind = "read"
	CalDAVOperationWrite CalDAVOperationKind = "write"
)

// CalDAVOperationOutcome classifies the result of a CalDAV operation for Sync Health.
type CalDAVOperationOutcome string

const (
	CalDAVOperationSuccess            CalDAVOperationOutcome = "success"
	CalDAVOperationRecoverableFailure CalDAVOperationOutcome = "recoverable_failure"
	CalDAVOperationIntegrityFailure   CalDAVOperationOutcome = "integrity_failure"
)

// CalDAVETagOutcome captures safe ETag/precondition metadata.
type CalDAVETagOutcome string

const (
	CalDAVETagNotApplicable CalDAVETagOutcome = "not_applicable"
	CalDAVETagMatched       CalDAVETagOutcome = "matched"
	CalDAVETagMismatched    CalDAVETagOutcome = "mismatched"
	CalDAVETagMissing       CalDAVETagOutcome = "missing"
	CalDAVETagGenerated     CalDAVETagOutcome = "generated"
)

// CalDAVErrorCode is a redacted diagnostic category. It must not include raw parser
// output, raw ICS, event descriptions, attendee emails, or task content.
type CalDAVErrorCode string

const (
	CalDAVErrorNone              CalDAVErrorCode = ""
	CalDAVErrorParse             CalDAVErrorCode = "parse_error"
	CalDAVErrorETagConflict      CalDAVErrorCode = "etag_conflict"
	CalDAVErrorDuplicateUID      CalDAVErrorCode = "duplicate_uid"
	CalDAVErrorCorruptICS        CalDAVErrorCode = "corrupt_ics"
	CalDAVErrorWriteFailed       CalDAVErrorCode = "write_failed"
	CalDAVErrorAuthFailed        CalDAVErrorCode = "auth_failed"
	CalDAVErrorUnsupportedMethod CalDAVErrorCode = "unsupported_method"
	CalDAVErrorUnknown           CalDAVErrorCode = "unknown_error"
)

// CalDAV client fingerprints are normalized, debug-bundle-safe identifiers.
const (
	CalDAVClientAppleCalendar = "apple-calendar"
	CalDAVClientFantastical   = "fantastical"
	CalDAVClientDAVx5         = "davx5"
	CalDAVClientThunderbird   = "thunderbird"
	CalDAVClientUnknown       = "unknown-caldav-client"
)

// CalDAVOperation is redacted request metadata used by Sync Health and diagnostics.
// It intentionally stores no raw ICS, event body, title, description, attendee,
// task, LLM, Todoist content, or raw user-agent by default.
type CalDAVOperation struct {
	ID                string
	OccurredAt        time.Time
	Method            string
	PathPattern       string
	StatusCode        int
	DurationMillis    int64
	ClientFingerprint string
	ETagOutcome       CalDAVETagOutcome
	OperationKind     CalDAVOperationKind
	Outcome           CalDAVOperationOutcome
	ErrorCode         CalDAVErrorCode
	RedactedError     string
	RequestSizeBytes  int64
	ResponseSizeBytes int64
}

func (op CalDAVOperation) IsWriteFailure() bool {
	return op.OperationKind == CalDAVOperationWrite && op.Outcome != CalDAVOperationSuccess
}

func (op CalDAVOperation) IsRecoverableFailure() bool {
	return op.Outcome == CalDAVOperationRecoverableFailure
}

func (op CalDAVOperation) IsIntegrityFailure() bool {
	return op.Outcome == CalDAVOperationIntegrityFailure
}

func (op CalDAVOperation) IsETagConflict() bool {
	return op.ErrorCode == CalDAVErrorETagConflict || op.ETagOutcome == CalDAVETagMismatched || op.StatusCode == 412
}

// SummarizeCalDAVOperations adapts redacted operation records into the issue #11
// Sync Health evaluator input shape. This keeps future APIs/debug bundles from
// needing raw event data to compute health.
func SummarizeCalDAVOperations(operations []*CalDAVOperation) RecentOperationSummary {
	summary := RecentOperationSummary{HasRecentOperationData: len(operations) > 0}
	for _, op := range operations {
		if op == nil {
			continue
		}

		if op.IsWriteFailure() {
			summary.WriteFailures++
		}
		if op.IsRecoverableFailure() {
			summary.RecoverableClientSyncFailures++
		}
		if op.IsETagConflict() {
			summary.ETagConflicts++
		}
		if op.OperationKind == CalDAVOperationWrite && op.StatusCode >= 500 {
			summary.CalendarWritePathFailing = true
		}

		switch op.ErrorCode {
		case CalDAVErrorCorruptICS, CalDAVErrorParse:
			summary.CorruptICSIncidents++
		case CalDAVErrorDuplicateUID:
			summary.UnresolvedDuplicateUIDs++
		case CalDAVErrorWriteFailed:
			if op.OperationKind == CalDAVOperationWrite {
				summary.CalendarWritePathFailing = true
			}
		}
	}
	return summary
}

// CalDAVOperationRepo persists redacted operation metadata. Implementations must
// enforce bounded retention and must not persist raw calendar data.
type CalDAVOperationRepo interface {
	Record(operation *CalDAVOperation) error
	Prune() error
}
