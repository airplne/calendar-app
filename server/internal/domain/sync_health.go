package domain

import "time"

// SyncHealthStatus is the deterministic health state used by onboarding,
// diagnostics, and later planning Apply gating. Values are lower-case so the
// same model can be reused by future JSON APIs without translation.
type SyncHealthStatus string

const (
	SyncHealthHealthy  SyncHealthStatus = "healthy"
	SyncHealthWarning  SyncHealthStatus = "warning"
	SyncHealthCritical SyncHealthStatus = "critical"
	SyncHealthUnknown  SyncHealthStatus = "unknown"
)

// GreenSyncValidationStatus captures the latest create/edit/delete validation
// result from an external CalDAV client. The validation flow itself is added in
// a later issue; this model gives Sync Health a stable input shape.
type GreenSyncValidationStatus string

const (
	GreenSyncNeverCompleted GreenSyncValidationStatus = "never_completed"
	GreenSyncPassed         GreenSyncValidationStatus = "passed"
	GreenSyncFailed         GreenSyncValidationStatus = "failed"
)

// SyncHealthReasonSeverity mirrors SyncHealthStatus for individual reasons.
type SyncHealthReasonSeverity string

const (
	SyncHealthReasonWarning  SyncHealthReasonSeverity = "warning"
	SyncHealthReasonCritical SyncHealthReasonSeverity = "critical"
	SyncHealthReasonUnknown  SyncHealthReasonSeverity = "unknown"
)

const (
	SyncHealthReasonGreenSyncNotCompleted       = "green_sync_not_completed"
	SyncHealthReasonGreenSyncFailed             = "green_sync_failed"
	SyncHealthReasonValidationStale             = "validation_stale"
	SyncHealthReasonRecentWriteFailures         = "recent_write_failures"
	SyncHealthReasonETagConflictThreshold       = "etag_conflict_threshold_exceeded"
	SyncHealthReasonRecoverableClientFailures   = "recoverable_client_sync_failures"
	SyncHealthReasonCorruptICSDetected          = "corrupt_ics_detected"
	SyncHealthReasonDuplicateUIDUnresolved      = "duplicate_uid_unresolved"
	SyncHealthReasonRoundtripValidationFailed   = "roundtrip_validation_failed"
	SyncHealthReasonCalendarWritePathFailing    = "calendar_write_path_failing"
	SyncHealthReasonNoRecentOperationData       = "no_recent_operation_data"
	SyncHealthReasonServerCannotDetermineHealth = "server_cannot_determine_health"
)

// SyncHealthReason explains why a non-healthy status was selected. Messages are
// intentionally generic and must not include raw ICS, event descriptions,
// attendee data, or other calendar content.
type SyncHealthReason struct {
	Code     string
	Severity SyncHealthReasonSeverity
	Message  string
}

// GreenSyncValidation models the latest external-client create/edit/delete
// validation state. All three step flags must be true for a passed green sync.
type GreenSyncValidation struct {
	Status            GreenSyncValidationStatus
	CompletedAt       *time.Time
	CreateStepPassed  bool
	EditStepPassed    bool
	DeleteStepPassed  bool
}

// Completed returns true only when green-sync validation successfully completed
// and all create/edit/delete steps passed.
func (g GreenSyncValidation) Completed() bool {
	return g.Status == GreenSyncPassed && g.CreateStepPassed && g.EditStepPassed && g.DeleteStepPassed
}

// RecentOperationSummary is a redacted aggregate of recent CalDAV operation
// outcomes used by the health evaluator. It deliberately stores counters and
// booleans only, not event payloads or descriptions.
type RecentOperationSummary struct {
	HasRecentOperationData        bool
	WriteFailures                 int
	ETagConflicts                 int
	RecoverableClientSyncFailures int
	CorruptICSIncidents           int
	UnresolvedDuplicateUIDs       int
	RoundtripValidationFailed     bool
	CalendarWritePathFailing      bool
	ServerCannotDetermineHealth   bool
}

// SyncHealthEvaluationInput contains all inputs needed to compute a health
// status. Later persistence/API work should adapt repository records into this
// redacted shape before calling the evaluator.
type SyncHealthEvaluationInput struct {
	Now        time.Time
	GreenSync  GreenSyncValidation
	Operations RecentOperationSummary
}

// SyncHealth is the deterministic evaluation result.
type SyncHealth struct {
	Status             SyncHealthStatus
	GreenSyncCompleted bool
	LastValidationAt   *time.Time
	LastValidationState GreenSyncValidationStatus
	Reasons            []SyncHealthReason
	EvaluatedAt        time.Time
}

// SyncHealthEvaluationConfig contains implementation-configurable thresholds.
type SyncHealthEvaluationConfig struct {
	ETagConflictWarningThreshold int
	ValidationStaleAfter         time.Duration
}

// DefaultSyncHealthEvaluationConfig returns the MVP defaults from the PRP:
// ETag conflicts warn when more than five occur in the recent window, and green
// sync validation is stale after 14 days.
func DefaultSyncHealthEvaluationConfig() SyncHealthEvaluationConfig {
	return SyncHealthEvaluationConfig{
		ETagConflictWarningThreshold: 5,
		ValidationStaleAfter:         14 * 24 * time.Hour,
	}
}

// SyncHealthEvaluator evaluates Sync Health using deterministic rules.
type SyncHealthEvaluator struct {
	config SyncHealthEvaluationConfig
}

// NewSyncHealthEvaluator creates an evaluator. Zero-value config fields use MVP
// defaults so callers can opt into only the threshold values they need.
func NewSyncHealthEvaluator(config SyncHealthEvaluationConfig) SyncHealthEvaluator {
	defaults := DefaultSyncHealthEvaluationConfig()
	if config.ETagConflictWarningThreshold == 0 {
		config.ETagConflictWarningThreshold = defaults.ETagConflictWarningThreshold
	}
	if config.ValidationStaleAfter == 0 {
		config.ValidationStaleAfter = defaults.ValidationStaleAfter
	}
	return SyncHealthEvaluator{config: config}
}

// NewDefaultSyncHealthEvaluator creates an evaluator with MVP default thresholds.
func NewDefaultSyncHealthEvaluator() SyncHealthEvaluator {
	return NewSyncHealthEvaluator(DefaultSyncHealthEvaluationConfig())
}

// Evaluate returns the most severe applicable status. Critical conditions take
// priority over warning and unknown conditions. Unknown is used when the server
// lacks enough trustworthy data to make a healthy/warning/critical claim.
func (e SyncHealthEvaluator) Evaluate(input SyncHealthEvaluationInput) SyncHealth {
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}

	result := SyncHealth{
		Status:              SyncHealthHealthy,
		GreenSyncCompleted:  input.GreenSync.Completed(),
		LastValidationAt:    input.GreenSync.CompletedAt,
		LastValidationState: input.GreenSync.Status,
		EvaluatedAt:         now,
	}

	criticalReasons := e.criticalReasons(input)
	if len(criticalReasons) > 0 {
		result.Status = SyncHealthCritical
		result.Reasons = criticalReasons
		return result
	}

	unknownReasons := e.unknownReasons(input)
	if len(unknownReasons) > 0 {
		result.Status = SyncHealthUnknown
		result.Reasons = unknownReasons
		return result
	}

	warningReasons := e.warningReasons(input, now)
	if len(warningReasons) > 0 {
		result.Status = SyncHealthWarning
		result.Reasons = warningReasons
		return result
	}

	return result
}

func (e SyncHealthEvaluator) criticalReasons(input SyncHealthEvaluationInput) []SyncHealthReason {
	var reasons []SyncHealthReason

	if input.GreenSync.Status == GreenSyncFailed || greenSyncMarkedPassedButIncomplete(input.GreenSync) {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonGreenSyncFailed,
			Severity: SyncHealthReasonCritical,
			Message:  "Green-sync validation failed.",
		})
	}
	if input.Operations.CorruptICSIncidents > 0 {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonCorruptICSDetected,
			Severity: SyncHealthReasonCritical,
			Message:  "Corrupt stored calendar data was detected.",
		})
	}
	if input.Operations.UnresolvedDuplicateUIDs > 0 {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonDuplicateUIDUnresolved,
			Severity: SyncHealthReasonCritical,
			Message:  "An unresolved duplicate UID incident exists.",
		})
	}
	if input.Operations.RoundtripValidationFailed {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonRoundtripValidationFailed,
			Severity: SyncHealthReasonCritical,
			Message:  "PUT/GET roundtrip validation failed.",
		})
	}
	if input.Operations.CalendarWritePathFailing {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonCalendarWritePathFailing,
			Severity: SyncHealthReasonCritical,
			Message:  "Calendar write path is failing.",
		})
	}

	return reasons
}

func (e SyncHealthEvaluator) unknownReasons(input SyncHealthEvaluationInput) []SyncHealthReason {
	var reasons []SyncHealthReason

	if !input.GreenSync.Completed() {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonGreenSyncNotCompleted,
			Severity: SyncHealthReasonUnknown,
			Message:  "Green-sync validation has not completed.",
		})
	}
	if !input.Operations.HasRecentOperationData {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonNoRecentOperationData,
			Severity: SyncHealthReasonUnknown,
			Message:  "No recent CalDAV operation data is available.",
		})
	}
	if input.Operations.ServerCannotDetermineHealth {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonServerCannotDetermineHealth,
			Severity: SyncHealthReasonUnknown,
			Message:  "Server cannot determine sync health from current data.",
		})
	}

	return reasons
}

func (e SyncHealthEvaluator) warningReasons(input SyncHealthEvaluationInput, now time.Time) []SyncHealthReason {
	var reasons []SyncHealthReason

	if input.GreenSync.CompletedAt != nil && now.Sub(*input.GreenSync.CompletedAt) > e.config.ValidationStaleAfter {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonValidationStale,
			Severity: SyncHealthReasonWarning,
			Message:  "Green-sync validation is stale and should be rerun.",
		})
	}
	if input.Operations.ETagConflicts > e.config.ETagConflictWarningThreshold {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonETagConflictThreshold,
			Severity: SyncHealthReasonWarning,
			Message:  "Recent ETag conflicts exceeded the warning threshold.",
		})
	}
	if input.Operations.WriteFailures > 0 {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonRecentWriteFailures,
			Severity: SyncHealthReasonWarning,
			Message:  "Recent recoverable CalDAV write failures were observed.",
		})
	}
	if input.Operations.RecoverableClientSyncFailures > 0 {
		reasons = append(reasons, SyncHealthReason{
			Code:     SyncHealthReasonRecoverableClientFailures,
			Severity: SyncHealthReasonWarning,
			Message:  "Recent recoverable client sync failures were observed.",
		})
	}

	return reasons
}

func greenSyncMarkedPassedButIncomplete(g GreenSyncValidation) bool {
	return g.Status == GreenSyncPassed && !g.Completed()
}
