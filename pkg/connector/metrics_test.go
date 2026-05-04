package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncMetrics_RecordWouldEmitNonEnabled_CountsAndSamples(t *testing.T) {
	m := newSyncMetrics()

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
	m := newSyncMetrics()

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
	m := newSyncMetrics()

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
	m := newSyncMetrics()
	assert.NotPanics(t, func() {
		m.emitSummary(context.Background())
	})
}
