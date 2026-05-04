package connector

import (
	"context"
	"sync"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// syncMetrics aggregates diagnostic counters for **a single SDK call**
// (one List() or one Grants() invocation). It is intentionally
// per-call, not per-sync: storing per-sync state on the connector
// builder is unsafe in the production Lambda runtime where the
// container can recycle, reload, or be hit concurrently between SDK
// calls. See ~/.claude/skills/baton-runtime/SKILL.md.
//
// Aggregation across calls happens in Datadog — every WARN/INFO log
// emitted from this struct includes `sync_id` so log queries can sum
// counters per sync without any in-process accumulation.
//
// Per-event WARN logs are sampled (capped at sampledWarningCap per
// call) so a 10K-user tenant does not emit hundreds of individual
// WARN events. The end-of-call INFO summary captures the totals for
// that call.
type syncMetrics struct {
	mu sync.Mutex

	// syncID is stamped onto every log emitted from this metrics
	// instance, so Datadog can group across calls within the sync.
	syncID string

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

func newSyncMetrics(syncID string) *syncMetrics {
	return &syncMetrics{
		syncID:              syncID,
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

// hasEvents reports whether anything was counted. Callers use this to
// skip emitting an empty summary log on every Grants() call (one per
// resource → 10K log lines per sync would otherwise be common).
func (m *syncMetrics) hasEvents() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.wouldEmitNonEnabled) > 0 {
		return true
	}
	return m.cacheMissesInListUsers > 0 ||
		m.cacheMissesInGrants > 0 ||
		m.rehireCollisions > 0 ||
		m.droppedEmptyUserIDs > 0 ||
		m.terminatedSkipsInGrants > 0
}

// emitSummary writes one structured INFO log with the totals for this
// SDK call. Every log line includes `sync_id` so Datadog queries can
// aggregate across the per-call summaries within a single sync.
//
// Datadog query example (sum cascade-blast-radius events for a sync):
//
//	service:baton-rippling "sync metrics summary" @sync_id:<id>
//	| stats sum(@would_emit_non_enabled_total) as total
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
		zap.String("sync_id", m.syncID),
		zap.Int("would_emit_non_enabled_total", total),
		zap.Any("would_emit_non_enabled_by_reason", byReason),
		zap.Int("cache_misses_list_users", m.cacheMissesInListUsers),
		zap.Int("cache_misses_grants", m.cacheMissesInGrants),
		zap.Int("rehire_collisions", m.rehireCollisions),
		zap.Int("dropped_empty_user_ids", m.droppedEmptyUserIDs),
		zap.Int("terminated_skips_grants", m.terminatedSkipsInGrants),
	)
}

// syncIDField returns a zap.Field wrapping the sync_id for use on
// per-event WARN logs, so Datadog can group those events by sync too.
func (m *syncMetrics) syncIDField() zap.Field {
	if m == nil {
		return zap.Skip()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return zap.String("sync_id", m.syncID)
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
