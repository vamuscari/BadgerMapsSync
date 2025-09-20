package server

import (
	"badgermaps/events"
	"testing"
	"time"
)

func TestCalculatePercentile(t *testing.T) {
	tests := []struct {
		name       string
		values     []time.Duration
		percentile float64
		expected   time.Duration
	}{
		{
			name:       "Empty slice",
			values:     []time.Duration{},
			percentile: 50,
			expected:   0,
		},
		{
			name: "P50 (median) of odd count",
			values: []time.Duration{
				100 * time.Millisecond,
				200 * time.Millisecond,
				300 * time.Millisecond,
				400 * time.Millisecond,
				500 * time.Millisecond,
			},
			percentile: 50,
			expected:   300 * time.Millisecond,
		},
		{
			name: "P95 of sorted values",
			values: []time.Duration{
				100 * time.Millisecond,
				200 * time.Millisecond,
				300 * time.Millisecond,
				400 * time.Millisecond,
				500 * time.Millisecond,
				600 * time.Millisecond,
				700 * time.Millisecond,
				800 * time.Millisecond,
				900 * time.Millisecond,
				1000 * time.Millisecond,
			},
			percentile: 95,
			expected:   1000 * time.Millisecond,
		},
		{
			name: "P99 of unsorted values",
			values: []time.Duration{
				500 * time.Millisecond,
				100 * time.Millisecond,
				900 * time.Millisecond,
				300 * time.Millisecond,
				700 * time.Millisecond,
				200 * time.Millisecond,
				800 * time.Millisecond,
				400 * time.Millisecond,
				600 * time.Millisecond,
				1000 * time.Millisecond,
			},
			percentile: 99,
			expected:   1000 * time.Millisecond,
		},
		{
			name:       "Invalid percentile (negative)",
			values:     []time.Duration{100 * time.Millisecond},
			percentile: -1,
			expected:   0,
		},
		{
			name:       "Invalid percentile (> 100)",
			values:     []time.Duration{100 * time.Millisecond},
			percentile: 101,
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePercentile(tt.values, tt.percentile)
			if result != tt.expected {
				t.Errorf("calculatePercentile(%v, %.1f) = %v, want %v",
					tt.values, tt.percentile, result, tt.expected)
			}
		})
	}
}

func TestMetricsCollector_RecordAPICall(t *testing.T) {
	mc := NewMetricsCollector(nil, nil)

	// Record some API calls
	mc.RecordAPICall("/api/test", true, 100*time.Millisecond)
	mc.RecordAPICall("/api/test", false, 200*time.Millisecond)
	mc.RecordAPICall("/api/test", true, 150*time.Millisecond)

	// Check counters
	if mc.apiCallsTotal.Load() != 3 {
		t.Errorf("Expected 3 total API calls, got %d", mc.apiCallsTotal.Load())
	}
	if mc.apiCallsSuccess.Load() != 2 {
		t.Errorf("Expected 2 successful API calls, got %d", mc.apiCallsSuccess.Load())
	}
	if mc.apiCallsFailed.Load() != 1 {
		t.Errorf("Expected 1 failed API call, got %d", mc.apiCallsFailed.Load())
	}

	// Check latency recording
	mc.latencyMu.Lock()
	latencyCount := len(mc.apiLatencies)
	mc.latencyMu.Unlock()

	if latencyCount != 3 {
		t.Errorf("Expected 3 latency recordings, got %d", latencyCount)
	}
}

func TestMetricsCollector_LatencyArrayLimiting(t *testing.T) {
	mc := NewMetricsCollector(nil, nil)

	// Record more than 1000 API calls
	for i := 0; i < 1500; i++ {
		mc.RecordAPICall("/api/test", true, time.Duration(i)*time.Millisecond)
	}

	// Check that array is limited to 1000
	mc.latencyMu.Lock()
	latencyCount := len(mc.apiLatencies)
	mc.latencyMu.Unlock()

	if latencyCount != 1000 {
		t.Errorf("Expected latency array to be limited to 1000, got %d", latencyCount)
	}

	// Verify it keeps the most recent values
	mc.latencyMu.Lock()
	lastLatency := mc.apiLatencies[len(mc.apiLatencies)-1]
	mc.latencyMu.Unlock()

	// The last latency should be from the 1499th call (0-indexed)
	if lastLatency != 1499*time.Millisecond {
		t.Errorf("Expected last latency to be 1499ms, got %v", lastLatency)
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	// Create a real event dispatcher for testing
	eventDispatcher := events.NewEventDispatcher()
	mc := NewMetricsCollector(eventDispatcher, nil)

	// Record some metrics
	mc.RecordAPICall("/api/test", true, 100*time.Millisecond)
	mc.RecordDBQuery("users", "SELECT", true, 50*time.Millisecond)
	mc.RecordSyncOperation("accounts", true, 100)

	// Reset
	mc.Reset()

	// Check all counters are zero
	if mc.apiCallsTotal.Load() != 0 {
		t.Errorf("Expected API calls to be reset to 0, got %d", mc.apiCallsTotal.Load())
	}
	if mc.dbQueriesTotal.Load() != 0 {
		t.Errorf("Expected DB queries to be reset to 0, got %d", mc.dbQueriesTotal.Load())
	}
	if mc.syncOpsTotal.Load() != 0 {
		t.Errorf("Expected sync ops to be reset to 0, got %d", mc.syncOpsTotal.Load())
	}

	// Check latency arrays are empty
	mc.latencyMu.Lock()
	apiLatencyCount := len(mc.apiLatencies)
	dbLatencyCount := len(mc.dbLatencies)
	mc.latencyMu.Unlock()

	if apiLatencyCount != 0 {
		t.Errorf("Expected API latencies to be empty, got %d", apiLatencyCount)
	}
	if dbLatencyCount != 0 {
		t.Errorf("Expected DB latencies to be empty, got %d", dbLatencyCount)
	}
}

func TestMetricsCollector_GetSnapshot(t *testing.T) {
	mc := NewMetricsCollector(nil, nil)

	// Record various metrics
	mc.RecordAPICall("/api/test", true, 100*time.Millisecond)
	mc.RecordAPICall("/api/test", false, 200*time.Millisecond)
	mc.RecordDBQuery("users", "SELECT", true, 50*time.Millisecond)
	mc.RecordSyncOperation("accounts", true, 50)
	mc.RecordWebhook(true)
	mc.RecordJob(true, false)

	// Get snapshot
	snapshot := mc.GetSnapshot()

	// Verify snapshot contains correct data
	if snapshot.Performance.APICallsTotal != 2 {
		t.Errorf("Expected 2 API calls in snapshot, got %d", snapshot.Performance.APICallsTotal)
	}
	if snapshot.Performance.DBQueriesTotal != 1 {
		t.Errorf("Expected 1 DB query in snapshot, got %d", snapshot.Performance.DBQueriesTotal)
	}
	if snapshot.Performance.SyncOpsTotal != 1 {
		t.Errorf("Expected 1 sync operation in snapshot, got %d", snapshot.Performance.SyncOpsTotal)
	}
	if snapshot.Performance.WebhooksReceived != 1 {
		t.Errorf("Expected 1 webhook received in snapshot, got %d", snapshot.Performance.WebhooksReceived)
	}
	if snapshot.Performance.JobsScheduled != 1 {
		t.Errorf("Expected 1 job scheduled in snapshot, got %d", snapshot.Performance.JobsScheduled)
	}

	// Verify system metrics are populated
	if snapshot.System.GoroutineCount <= 0 {
		t.Errorf("Expected positive goroutine count, got %d", snapshot.System.GoroutineCount)
	}
	if snapshot.System.MemoryUsed <= 0 {
		t.Errorf("Expected positive memory usage, got %d", snapshot.System.MemoryUsed)
	}
}
