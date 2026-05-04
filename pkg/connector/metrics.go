package connector

import (
	"context"
	"sync"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// syncMetrics aggregates per-sync diagnostic counters for the user
// builder. It is allocated at the start of a sync (List token == "")
// and flushed as a single INFO summary at the end (NextPageToken == "").
//
// Per-event WARN logs at the same sites are sampled (capped at
// sampledWarningCap) so a 10K-user tenant with a 5% miss rate does not
// emit 500 individual WARN events to Datadog. The summary at end of
// sync provides the totals; the sampled events provide spot-check
// payloads for diagnosis.
//
// Methods are safe for concurrent use; the underlying connector flow
// is currently sequential, but Grants() may be called by the SDK in a
// different goroutine context than List().
type syncMetrics struct {
	mu sync.Mutex

	// Diagnostic counters. Each increment represents one user that the
	// connector is about to emit in a state that puts them at risk of
	// the CEL revocation cascade described in the plan.
	wouldEmitNonEnabled map[string]int

	// Cache and pagination diagnostics — useful for understanding root
	// causes B, C, E, G before the behavior-change PRs land.
	cacheMissesInListUsers  int
	cacheMissesInGrants     int
	rehireCollisions        int
	droppedEmptyUserIDs     int
	terminatedSkipsInGrants int

	// Sampling state for per-event WARN logs.
	sampledEmits int
}

const sampledWarningCap = 50

func newSyncMetrics() *syncMetrics {
	return &syncMetrics{
		wouldEmitNonEnabled: make(map[string]int),
	}
}

// recordWouldEmitNonEnabled increments the cascade-blast-radius
// counter for a given diagnostic reason and returns true if the caller
// is permitted to emit a sampled WARN log for this event. Callers
// should respect the return value to avoid log floods.
func (m *syncMetrics) recordWouldEmitNonEnabled(reason string) (sample bool) {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wouldEmitNonEnabled[reason]++
	if m.sampledEmits < sampledWarningCap {
		m.sampledEmits++
		return true
	}
	return false
}

func (m *syncMetrics) recordCacheMissInListUsers() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.cacheMissesInListUsers++
	m.mu.Unlock()
}

func (m *syncMetrics) recordCacheMissInGrants() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.cacheMissesInGrants++
	m.mu.Unlock()
}

func (m *syncMetrics) recordRehireCollision() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.rehireCollisions++
	m.mu.Unlock()
}

func (m *syncMetrics) recordDroppedEmptyUserID() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.droppedEmptyUserIDs++
	m.mu.Unlock()
}

func (m *syncMetrics) recordTerminatedSkipInGrants() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.terminatedSkipsInGrants++
	m.mu.Unlock()
}

// emitSummary writes one structured INFO log with the totals. The
// would_emit_non_enabled_total is the directly incident-relevant
// metric: every increment represents one user whose all-user-access
// CEL evaluation would flip to fail under the customer's current
// directory-status check.
func (m *syncMetrics) emitSummary(ctx context.Context) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	total := 0
	byReason := make(map[string]int, len(m.wouldEmitNonEnabled))
	for reason, count := range m.wouldEmitNonEnabled {
		total += count
		byReason[reason] = count
	}

	ctxzap.Extract(ctx).Info("baton-rippling: sync metrics summary",
		zap.Int("would_emit_non_enabled_total", total),
		zap.Any("would_emit_non_enabled_by_reason", byReason),
		zap.Int("cache_misses_list_users", m.cacheMissesInListUsers),
		zap.Int("cache_misses_grants", m.cacheMissesInGrants),
		zap.Int("rehire_collisions", m.rehireCollisions),
		zap.Int("dropped_empty_user_ids", m.droppedEmptyUserIDs),
		zap.Int("terminated_skips_grants", m.terminatedSkipsInGrants),
	)
}

// Diagnostic reasons. Used as label values for the
// would_emit_non_enabled counter and as zap fields on sampled WARN
// logs.
const (
	reasonWorkerNil           = "worker_nil"
	reasonWorkerTerminated    = "worker_terminated"
	reasonWorkerPreHire       = "worker_pre_hire"
	reasonWorkerInit          = "worker_init"
	reasonWorkerStatusEmpty   = "worker_status_empty"
	reasonWorkerStatusUnknown = "worker_status_unknown"
)
