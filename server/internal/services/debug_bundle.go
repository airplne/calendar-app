package services

import (
	"context"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

const DebugBundleSchemaVersion = "debug_bundle.v1"

type DebugBundleService struct {
	syncHealth *SyncHealthService
	options    DebugBundleOptions
}

type DebugBundleOptions struct {
	Version       string
	Environment   string
	DataDir       string
	MigrationsDir string
}

type DebugBundle struct {
	SchemaVersion string                 `json:"schema_version"`
	GeneratedAt   time.Time              `json:"generated_at"`
	Redaction     DebugBundleRedaction   `json:"redaction"`
	Server        DebugBundleServer      `json:"server"`
	Config        DebugBundleConfig      `json:"config"`
	Runtime       DebugBundleRuntime     `json:"runtime"`
	SyncHealth    DebugBundleSyncHealth  `json:"sync_health"`
	CalDAV        DebugBundleCalDAV      `json:"caldav"`
	ErrorTraces   []DebugBundleErrorItem `json:"error_traces"`
	Notes         []string               `json:"notes"`
}

type DebugBundleRedaction struct {
	Mode                 string `json:"mode"`
	RawICSIncluded       bool   `json:"raw_ics_included"`
	RawUserAgentIncluded bool   `json:"raw_user_agent_included"`
	SecretsIncluded      bool   `json:"secrets_included"`
}

type DebugBundleServer struct {
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type DebugBundleConfig struct {
	DataDirConfigured       bool     `json:"data_dir_configured"`
	MigrationsDirConfigured bool     `json:"migrations_dir_configured"`
	AuthMode                string   `json:"auth_mode"`
	SecretsRedacted         []string `json:"secrets_redacted"`
}

type DebugBundleRuntime struct {
	GoVersion string `json:"go_version"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

type DebugBundleSyncHealth struct {
	Status             string              `json:"status"`
	EvaluatedAt         time.Time           `json:"evaluated_at"`
	Reasons             []DebugBundleReason `json:"reasons"`
	GreenSyncCompleted bool                `json:"green_sync_completed"`
	GreenSyncState     string              `json:"green_sync_state"`
	OperationCounts    SyncOperationCounts  `json:"operation_counts"`
	Latency            SyncLatency          `json:"latency_ms"`
	Clients            []SyncClientSummary  `json:"clients"`
}

type DebugBundleReason struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type DebugBundleCalDAV struct {
	RecentOperations []DebugBundleOperation `json:"recent_operations"`
	ClientSummaries  []SyncClientSummary    `json:"client_summaries"`
}

type DebugBundleOperation struct {
	OccurredAt        time.Time `json:"occurred_at"`
	Method            string    `json:"method"`
	PathPattern       string    `json:"path_pattern"`
	StatusCode        int       `json:"status_code"`
	DurationMillis    int64     `json:"duration_ms"`
	ClientFingerprint string    `json:"client_fingerprint"`
	ETagOutcome       string    `json:"etag_outcome"`
	OperationKind     string    `json:"operation_kind"`
	Outcome           string    `json:"outcome"`
	ErrorCode         string    `json:"error_code,omitempty"`
	RedactedError     string    `json:"redacted_error,omitempty"`
}

type DebugBundleErrorItem struct {
	OccurredAt        time.Time `json:"occurred_at"`
	ClientFingerprint string    `json:"client_fingerprint"`
	Method            string    `json:"method"`
	StatusCode        int       `json:"status_code"`
	ErrorCode         string    `json:"error_code"`
	RedactedError     string    `json:"redacted_error"`
}

func NewDebugBundleService(syncHealth *SyncHealthService, options DebugBundleOptions) *DebugBundleService {
	if options.Version == "" {
		options.Version = "unknown"
	}
	if options.Environment == "" {
		options.Environment = "self-hosted"
	}
	return &DebugBundleService{syncHealth: syncHealth, options: options}
}

func (s *DebugBundleService) Build(ctx context.Context) (*DebugBundle, error) {
	summary, err := s.syncHealth.Summary(ctx)
	if err != nil {
		return nil, err
	}
	operations, err := s.syncHealth.RecentOperations(ctx, DefaultSyncHealthOperationLimit)
	if err != nil {
		return nil, err
	}

	bundle := &DebugBundle{
		SchemaVersion: DebugBundleSchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		Redaction: DebugBundleRedaction{
			Mode:                 "default",
			RawICSIncluded:       false,
			RawUserAgentIncluded: false,
			SecretsIncluded:      false,
		},
		Server: DebugBundleServer{Version: s.options.Version, Environment: s.options.Environment},
		Config: DebugBundleConfig{
			DataDirConfigured:       s.options.DataDir != "",
			MigrationsDirConfigured: s.options.MigrationsDir != "",
			AuthMode:                "basic_auth_env_credentials",
			SecretsRedacted:         RedactedSecretNames(),
		},
		Runtime: DebugBundleRuntime{GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH},
		SyncHealth: DebugBundleSyncHealth{
			Status:             string(summary.Health.Status),
			EvaluatedAt:         summary.Health.EvaluatedAt,
			Reasons:             toDebugBundleReasons(summary.Health.Reasons),
			GreenSyncCompleted: summary.Health.GreenSyncCompleted,
			GreenSyncState:     string(summary.Health.LastValidationState),
			OperationCounts:    summary.OperationCounts,
			Latency:            summary.Latency,
			Clients:            summary.Clients,
		},
		CalDAV: DebugBundleCalDAV{
			RecentOperations: toDebugBundleOperations(operations),
			ClientSummaries:  summary.Clients,
		},
		ErrorTraces: toDebugBundleErrors(operations),
		Notes: []string{
			"Default debug bundles contain redacted metadata only.",
			"Raw ICS, event descriptions, attendees, raw user-agents, tokens, passwords, LLM context, and Todoist data are excluded.",
			"Green-sync validation remains incomplete until issue #15 implements the create/edit/delete validation flow.",
		},
	}
	return bundle, nil
}

func toDebugBundleReasons(reasons []domain.SyncHealthReason) []DebugBundleReason {
	out := make([]DebugBundleReason, 0, len(reasons))
	for _, reason := range reasons {
		out = append(out, DebugBundleReason{Code: reason.Code, Severity: string(reason.Severity), Message: reason.Message})
	}
	return out
}

func toDebugBundleOperations(operations []*domain.CalDAVOperation) []DebugBundleOperation {
	out := make([]DebugBundleOperation, 0, len(operations))
	for _, op := range operations {
		if op == nil {
			continue
		}
		out = append(out, DebugBundleOperation{
			OccurredAt:        op.OccurredAt,
			Method:            op.Method,
			PathPattern:       op.PathPattern,
			StatusCode:        op.StatusCode,
			DurationMillis:    op.DurationMillis,
			ClientFingerprint: op.ClientFingerprint,
			ETagOutcome:       string(op.ETagOutcome),
			OperationKind:     string(op.OperationKind),
			Outcome:           string(op.Outcome),
			ErrorCode:         string(op.ErrorCode),
			RedactedError:     op.RedactedError,
		})
	}
	return out
}

func toDebugBundleErrors(operations []*domain.CalDAVOperation) []DebugBundleErrorItem {
	items := make([]DebugBundleErrorItem, 0)
	for _, op := range operations {
		if op == nil || op.Outcome == domain.CalDAVOperationSuccess {
			continue
		}
		items = append(items, DebugBundleErrorItem{
			OccurredAt:        op.OccurredAt,
			ClientFingerprint: op.ClientFingerprint,
			Method:            op.Method,
			StatusCode:        op.StatusCode,
			ErrorCode:         string(op.ErrorCode),
			RedactedError:     op.RedactedError,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt.After(items[j].OccurredAt) })
	return items
}

func RedactedSecretNames() []string {
	names := []string{}
	for _, pair := range os.Environ() {
		name := pair
		if idx := strings.Index(pair, "="); idx >= 0 {
			name = pair[:idx]
		}
		upper := strings.ToUpper(name)
		if strings.Contains(upper, "PASS") || strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "KEY") || strings.Contains(upper, "AUTH") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
