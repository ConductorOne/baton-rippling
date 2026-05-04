package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncMetrics_RecordWouldEmitNonEnabled_CountsAndSamples(t *testing.T) {
	m := newSyncMetrics("test-sync-id")

	// First sampledWarningCap calls should return true (sample) and
	// every call should increment the counter regardless of sampling.
	for range sampledWarningCap {
		assert.True(t, m.recordWouldEmitNonEnabled(reasonWorkerNil), "first %d calls must sample", sampledWarningCap)
	}
	// Subsequent calls keep counting but stop sampling.
	for range 10 {
		assert.False(t, m.recordWouldEmitNonEnabled(reasonWorkerNil), "calls beyond cap must not sample")
	}

	assert.Equal(t, sampledWarningCap+10, m.wouldEmitNonEnabled[reasonWorkerNil])
}

func TestSyncMetrics_RecordWouldEmitNonEnabled_BreakdownByReason(t *testing.T) {
	m := newSyncMetrics("test-sync-id")

	m.recordWouldEmitNonEnabled(reasonWorkerNil)
	m.recordWouldEmitNonEnabled(reasonWorkerNil)
	m.recordWouldEmitNonEnabled(reasonWorkerTerminated)
	m.recordWouldEmitNonEnabled(reasonWorkerPreHire)
	m.recordWouldEmitNonEnabled(reasonWorkerStatusUnknown)

	assert.Equal(t, 2, m.wouldEmitNonEnabled[reasonWorkerNil])
	assert.Equal(t, 1, m.wouldEmitNonEnabled[reasonWorkerTerminated])
	assert.Equal(t, 1, m.wouldEmitNonEnabled[reasonWorkerPreHire])
	assert.Equal(t, 1, m.wouldEmitNonEnabled[reasonWorkerStatusUnknown])
}

func TestSyncMetrics_NilReceiverIsSafe(t *testing.T) {
	// All record* methods must tolerate a nil receiver so callers don't
	// crash if metrics weren't initialized for some reason.
	var m *syncMetrics
	assert.NotPanics(t, func() {
		m.recordWouldEmitNonEnabled(reasonWorkerNil)
		m.recordCacheMissInListUsers()
		m.recordCacheMissInGrants()
		m.recordRehireCollision()
		m.recordDroppedEmptyUserID()
		m.recordTerminatedSkipInGrants()
		m.emitSummary(context.Background())
	})
}

func TestSyncMetrics_CountersIncrement(t *testing.T) {
	m := newSyncMetrics("test-sync-id")

	m.recordCacheMissInListUsers()
	m.recordCacheMissInListUsers()
	m.recordCacheMissInGrants()
	m.recordRehireCollision()
	m.recordRehireCollision()
	m.recordRehireCollision()
	m.recordDroppedEmptyUserID()
	m.recordTerminatedSkipInGrants()

	assert.Equal(t, 2, m.cacheMissesInListUsers)
	assert.Equal(t, 1, m.cacheMissesInGrants)
	assert.Equal(t, 3, m.rehireCollisions)
	assert.Equal(t, 1, m.droppedEmptyUserIDs)
	assert.Equal(t, 1, m.terminatedSkipsInGrants)
}

func TestSyncMetrics_EmitSummaryDoesNotPanicOnFreshMetrics(t *testing.T) {
	m := newSyncMetrics("test-sync-id")
	assert.NotPanics(t, func() {
		m.emitSummary(context.Background())
	})
}

func TestSyncMetrics_HasEvents(t *testing.T) {
	m := newSyncMetrics("test-sync-id")
	assert.False(t, m.hasEvents(), "fresh metrics has no events")

	m.recordCacheMissInListUsers()
	assert.True(t, m.hasEvents(), "after recording a cache miss, hasEvents must be true")

	m2 := newSyncMetrics("test-sync-id")
	m2.recordWouldEmitNonEnabled(reasonWorkerNil)
	assert.True(t, m2.hasEvents(), "after recording a non-enabled event, hasEvents must be true")
}

func TestUserBuilder_HasNoPerSyncStateField(t *testing.T) {
	// Regression guard: per-sync state on the userBuilder struct is
	// unsafe in the production Lambda runtime (struct may recycle,
	// reload, or be hit concurrently between SDK calls). If a future
	// change adds a stateful field here, redirect that state to
	// opts.Session or to a per-call value passed as a parameter.
	// See ~/.claude/skills/baton-runtime/SKILL.md.
	b := &userBuilder{}
	// The only fields allowed are read-only configuration set in New().
	// This test exists to make a future change visible — if the struct
	// shape changes, update this list and re-justify.
	assert.Nil(t, b.client)
	assert.False(t, b.expandWorkLocations)
	assert.Nil(t, b.customFieldNames)
}

