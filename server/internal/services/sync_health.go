package services

import (
	"context"
	"sort"
	"time"

	"github.com/airplne/calendar-app/server/internal/domain"
)

const DefaultSyncHealthOperationLimit = 50

// CalDAVOperationLister is the read side needed from operation metadata storage.
type CalDAVOperationLister interface {
	ListRecent(ctx context.Context, limit int) ([]*domain.CalDAVOperation, error)
}

// GreenSyncProvider supplies the latest green-sync validation state. Issue #15
// will replace the placeholder provider with persisted create/edit/delete state.
type GreenSyncProvider interface {
	Current(ctx context.Context) (domain.GreenSyncValidation, error)
}

type StaticGreenSyncProvider struct {
	Validation domain.GreenSyncValidation
}

func (p StaticGreenSyncProvider) Current(ctx context.Context) (domain.GreenSyncValidation, error) {
	return p.Validation, nil
}

func UnknownGreenSyncProvider() StaticGreenSyncProvider {
	return StaticGreenSyncProvider{Validation: domain.GreenSyncValidation{Status: domain.GreenSyncNeverCompleted}}
}

type SyncHealthService struct {
	operations CalDAVOperationLister
	greenSync  GreenSyncProvider
	evaluator   domain.SyncHealthEvaluator
	limit       int
}

func NewSyncHealthService(operations CalDAVOperationLister, greenSync GreenSyncProvider) *SyncHealthService {
	if greenSync == nil {
		greenSync = UnknownGreenSyncProvider()
	}
	return &SyncHealthService{
		operations: operations,
		greenSync:  greenSync,
		evaluator:  domain.NewDefaultSyncHealthEvaluator(),
		limit:      DefaultSyncHealthOperationLimit,
	}
}

func NewSyncHealthServiceWithEvaluator(operations CalDAVOperationLister, greenSync GreenSyncProvider, evaluator domain.SyncHealthEvaluator, limit int) *SyncHealthService {
	if greenSync == nil {
		greenSync = UnknownGreenSyncProvider()
	}
	if limit <= 0 {
		limit = DefaultSyncHealthOperationLimit
	}
	return &SyncHealthService{operations: operations, greenSync: greenSync, evaluator: evaluator, limit: limit}
}

type SyncHealthSummary struct {
	Health          domain.SyncHealth
	GreenSync       domain.GreenSyncValidation
	Operations      []*domain.CalDAVOperation
	OperationCounts SyncOperationCounts
	Latency         SyncLatency
	Clients         []SyncClientSummary
	LastSuccessAt   *time.Time
	LastFailureAt   *time.Time
}

type SyncOperationCounts struct {
	Total                int
	Success              int
	Failure              int
	ETagConflicts        int
	DuplicateUIDAttempts int
	ParseFailures        int
	CorruptICSIncidents  int
	WriteFailures        int
}

type SyncLatency struct {
	MedianMillis int64
	P95Millis    int64
}

type SyncClientSummary struct {
	Fingerprint       string
	DisplayName       string
	LastSeenAt        time.Time
	OperationCount    int
	OperationCount24h int
}

func (s *SyncHealthService) Summary(ctx context.Context) (*SyncHealthSummary, error) {
	operations, err := s.operations.ListRecent(ctx, s.limit)
	if err != nil {
		return nil, err
	}
	greenSync, err := s.greenSync.Current(ctx)
	if err != nil {
		return nil, err
	}

	health := s.evaluator.Evaluate(domain.SyncHealthEvaluationInput{
		Now:        time.Now().UTC(),
		GreenSync:  greenSync,
		Operations: domain.SummarizeCalDAVOperations(operations),
	})

	return &SyncHealthSummary{
		Health:          health,
		GreenSync:       greenSync,
		Operations:      operations,
		OperationCounts: CountOperations(operations),
		Latency:         ComputeLatency(operations),
		Clients:         SummarizeClients(operations, time.Now().UTC().Add(-24*time.Hour)),
		LastSuccessAt:   lastOperationTime(operations, true),
		LastFailureAt:   lastOperationTime(operations, false),
	}, nil
}

func (s *SyncHealthService) RecentOperations(ctx context.Context, limit int) ([]*domain.CalDAVOperation, error) {
	if limit <= 0 || limit > s.limit {
		limit = s.limit
	}
	return s.operations.ListRecent(ctx, limit)
}

func (s *SyncHealthService) Clients(ctx context.Context) ([]SyncClientSummary, error) {
	operations, err := s.operations.ListRecent(ctx, s.limit)
	if err != nil {
		return nil, err
	}
	return SummarizeClients(operations, time.Now().UTC().Add(-24*time.Hour)), nil
}

func CountOperations(operations []*domain.CalDAVOperation) SyncOperationCounts {
	counts := SyncOperationCounts{Total: len(operations)}
	for _, op := range operations {
		if op == nil {
			continue
		}
		if op.Outcome == domain.CalDAVOperationSuccess {
			counts.Success++
		} else {
			counts.Failure++
		}
		if op.IsETagConflict() {
			counts.ETagConflicts++
		}
		if op.ErrorCode == domain.CalDAVErrorDuplicateUID {
			counts.DuplicateUIDAttempts++
		}
		if op.ErrorCode == domain.CalDAVErrorParse {
			counts.ParseFailures++
		}
		if op.ErrorCode == domain.CalDAVErrorCorruptICS {
			counts.CorruptICSIncidents++
		}
		if op.IsWriteFailure() {
			counts.WriteFailures++
		}
	}
	return counts
}

func ComputeLatency(operations []*domain.CalDAVOperation) SyncLatency {
	if len(operations) == 0 {
		return SyncLatency{}
	}
	durations := make([]int64, 0, len(operations))
	for _, op := range operations {
		if op != nil {
			durations = append(durations, op.DurationMillis)
		}
	}
	if len(durations) == 0 {
		return SyncLatency{}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	return SyncLatency{MedianMillis: percentile(durations, 50), P95Millis: percentile(durations, 95)}
}

func SummarizeClients(operations []*domain.CalDAVOperation, since time.Time) []SyncClientSummary {
	byClient := map[string]*SyncClientSummary{}
	for _, op := range operations {
		if op == nil || op.ClientFingerprint == "" {
			continue
		}
		client := byClient[op.ClientFingerprint]
		if client == nil {
			client = &SyncClientSummary{Fingerprint: op.ClientFingerprint, DisplayName: DisplayNameForClient(op.ClientFingerprint)}
			byClient[op.ClientFingerprint] = client
		}
		client.OperationCount++
		if !op.OccurredAt.Before(since) {
			client.OperationCount24h++
		}
		if client.LastSeenAt.IsZero() || op.OccurredAt.After(client.LastSeenAt) {
			client.LastSeenAt = op.OccurredAt
		}
	}

	clients := make([]SyncClientSummary, 0, len(byClient))
	for _, client := range byClient {
		clients = append(clients, *client)
	}
	sort.Slice(clients, func(i, j int) bool { return clients[i].LastSeenAt.After(clients[j].LastSeenAt) })
	return clients
}

func DisplayNameForClient(fingerprint string) string {
	switch fingerprint {
	case domain.CalDAVClientAppleCalendar:
		return "Apple Calendar"
	case domain.CalDAVClientFantastical:
		return "Fantastical"
	case domain.CalDAVClientDAVx5:
		return "DAVx5"
	case domain.CalDAVClientThunderbird:
		return "Thunderbird"
	default:
		return "Unknown CalDAV client"
	}
}

func lastOperationTime(operations []*domain.CalDAVOperation, success bool) *time.Time {
	for _, op := range operations {
		if op == nil {
			continue
		}
		if success && op.Outcome == domain.CalDAVOperationSuccess {
			return &op.OccurredAt
		}
		if !success && op.Outcome != domain.CalDAVOperationSuccess {
			return &op.OccurredAt
		}
	}
	return nil
}

func percentile(sorted []int64, pct int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	idx := (pct*(len(sorted)-1) + 99) / 100
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
