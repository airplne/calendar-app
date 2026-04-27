package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/airplne/calendar-app/server/internal/domain"
	"github.com/airplne/calendar-app/server/internal/services"
)

type SyncHealthHandler struct {
	service *services.SyncHealthService
}

func NewSyncHealthHandler(service *services.SyncHealthService) *SyncHealthHandler {
	return &SyncHealthHandler{service: service}
}

func (h *SyncHealthHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.handleSummary)
	r.Get("/operations", h.handleOperations)
	r.Get("/clients", h.handleClients)
	r.Post("/validation-event/start", h.handleValidationNotImplemented)
	r.Post("/validation-event/verify", h.handleValidationNotImplemented)
	return r
}

type syncHealthResponse struct {
	Status               string                   `json:"status"`
	EvaluatedAt          time.Time                `json:"evaluated_at"`
	Reasons              []syncHealthReasonJSON   `json:"reasons"`
	LastSuccessAt        *time.Time               `json:"last_success_at"`
	LastFailureAt        *time.Time               `json:"last_failure_at"`
	GreenSyncCompleted   bool                     `json:"green_sync_completed"`
	GreenSyncCompletedAt *time.Time               `json:"green_sync_completed_at"`
	GreenSyncState       string                   `json:"green_sync_state"`
	OperationCounts      operationCountsJSON      `json:"operation_counts"`
	LatencyMS            latencyJSON              `json:"latency_ms"`
	Clients              []clientSummaryJSON      `json:"clients"`
	RecentOperations     []recentOperationJSON    `json:"recent_operations,omitempty"`
}

type syncHealthReasonJSON struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type operationCountsJSON struct {
	Total                int `json:"total"`
	Success              int `json:"success"`
	Failure              int `json:"failure"`
	ETagConflicts        int `json:"etag_conflicts"`
	DuplicateUIDAttempts int `json:"duplicate_uid_attempts"`
	ParseFailures        int `json:"parse_failures"`
	CorruptICSIncidents  int `json:"corrupt_ics_incidents"`
	WriteFailures        int `json:"write_failures"`
}

type latencyJSON struct {
	Median int64 `json:"median"`
	P95    int64 `json:"p95"`
}

type clientSummaryJSON struct {
	Fingerprint       string    `json:"fingerprint"`
	DisplayName       string    `json:"display_name"`
	LastSeenAt        time.Time `json:"last_seen_at"`
	OperationCount    int       `json:"operation_count"`
	OperationCount24h int       `json:"operation_count_24h"`
}

type recentOperationJSON struct {
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

func (h *SyncHealthHandler) handleSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.Summary(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "sync_health_unavailable", "Sync Health is unavailable.")
		return
	}
	writeJSON(w, http.StatusOK, toSyncHealthResponse(summary, false))
}

func (h *SyncHealthHandler) handleOperations(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, services.DefaultSyncHealthOperationLimit)
	operations, err := h.service.RecentOperations(r.Context(), limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "sync_health_operations_unavailable", "Recent operations are unavailable.")
		return
	}
	response := make([]recentOperationJSON, 0, len(operations))
	for _, op := range operations {
		response = append(response, toRecentOperationJSON(op))
	}
	writeJSON(w, http.StatusOK, map[string]any{"operations": response})
}

func (h *SyncHealthHandler) handleClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.service.Clients(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "sync_health_clients_unavailable", "Client summaries are unavailable.")
		return
	}
	response := make([]clientSummaryJSON, 0, len(clients))
	for _, client := range clients {
		response = append(response, toClientSummaryJSON(client))
	}
	writeJSON(w, http.StatusOK, map[string]any{"clients": response})
}

func (h *SyncHealthHandler) handleValidationNotImplemented(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"status": "not_implemented",
		"error_code": "green_sync_validation_not_implemented",
		"message": "Green-sync validation flow is implemented in issue #15.",
	})
}

func toSyncHealthResponse(summary *services.SyncHealthSummary, includeOperations bool) syncHealthResponse {
	response := syncHealthResponse{
		Status:               string(summary.Health.Status),
		EvaluatedAt:          summary.Health.EvaluatedAt,
		Reasons:              toReasonsJSON(summary.Health.Reasons),
		LastSuccessAt:        summary.LastSuccessAt,
		LastFailureAt:        summary.LastFailureAt,
		GreenSyncCompleted:   summary.Health.GreenSyncCompleted,
		GreenSyncCompletedAt: summary.Health.LastValidationAt,
		GreenSyncState:       string(summary.Health.LastValidationState),
		OperationCounts:      toOperationCountsJSON(summary.OperationCounts),
		LatencyMS:            latencyJSON{Median: summary.Latency.MedianMillis, P95: summary.Latency.P95Millis},
		Clients:              toClientsJSON(summary.Clients),
	}
	if includeOperations {
		for _, op := range summary.Operations {
			response.RecentOperations = append(response.RecentOperations, toRecentOperationJSON(op))
		}
	}
	return response
}

func toReasonsJSON(reasons []domain.SyncHealthReason) []syncHealthReasonJSON {
	out := make([]syncHealthReasonJSON, 0, len(reasons))
	for _, reason := range reasons {
		out = append(out, syncHealthReasonJSON{Code: reason.Code, Severity: string(reason.Severity), Message: reason.Message})
	}
	return out
}

func toOperationCountsJSON(counts services.SyncOperationCounts) operationCountsJSON {
	return operationCountsJSON{
		Total:                counts.Total,
		Success:              counts.Success,
		Failure:              counts.Failure,
		ETagConflicts:        counts.ETagConflicts,
		DuplicateUIDAttempts: counts.DuplicateUIDAttempts,
		ParseFailures:        counts.ParseFailures,
		CorruptICSIncidents:  counts.CorruptICSIncidents,
		WriteFailures:        counts.WriteFailures,
	}
}

func toClientsJSON(clients []services.SyncClientSummary) []clientSummaryJSON {
	out := make([]clientSummaryJSON, 0, len(clients))
	for _, client := range clients {
		out = append(out, toClientSummaryJSON(client))
	}
	return out
}

func toClientSummaryJSON(client services.SyncClientSummary) clientSummaryJSON {
	return clientSummaryJSON{
		Fingerprint:       client.Fingerprint,
		DisplayName:       client.DisplayName,
		LastSeenAt:        client.LastSeenAt,
		OperationCount:    client.OperationCount,
		OperationCount24h: client.OperationCount24h,
	}
}

func toRecentOperationJSON(op *domain.CalDAVOperation) recentOperationJSON {
	if op == nil {
		return recentOperationJSON{}
	}
	return recentOperationJSON{
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
	}
}

func parseLimit(r *http.Request, fallback int) int {
	value := r.URL.Query().Get("limit")
	if value == "" {
		return fallback
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return fallback
	}
	if limit > services.DefaultSyncHealthOperationLimit {
		return services.DefaultSyncHealthOperationLimit
	}
	return limit
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeJSONError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{"error_code": code, "message": message})
}
