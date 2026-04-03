package app

import (
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/events"
	"path/filepath"
	"testing"
)

func newJobLogTrackingTestApp(t *testing.T) *App {
	t.Helper()

	a := NewApp()
	dbPath := filepath.Join(t.TempDir(), "job_log_tracking.db")
	db, err := database.NewDB(&database.DBConfig{Type: "sqlite3", Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect test db: %v", err)
	}
	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}
	db.SetConnected(true)
	a.DB = db
	a.ensureJobLogTracking()

	t.Cleanup(func() {
		_ = db.Close()
	})

	return a
}

func syncJobPayload(jobID, source, mode, status string) events.GenericPayload {
	data := map[string]interface{}{
		"job_id":      jobID,
		"root_job_id": jobID,
		"job_kind":    "sync",
		"mode":        mode,
		"source":      source,
	}
	if status != "" {
		data["status"] = status
	}
	return events.GenericPayload{
		Type: "sync.job",
		Data: data,
	}
}

func findJobLogByCorrelation(t *testing.T, entries []database.JobLogEntry, correlationID string) database.JobLogEntry {
	t.Helper()
	for _, entry := range entries {
		if entry.CorrelationID == correlationID {
			return entry
		}
	}
	t.Fatalf("expected job log entry for correlation %q", correlationID)
	return database.JobLogEntry{}
}

func TestQueuedCompletionWaitsForMetricTerminalEvent(t *testing.T) {
	a := newJobLogTrackingTestApp(t)
	jobID := "sync_job_wait_metrics"

	a.recordSyncJobEvent(events.Event{
		Type:    "sync.job.start",
		Source:  "accounts",
		Payload: syncJobPayload(jobID, "accounts", "pull_accounts", "running"),
	})
	a.recordSyncJobEvent(events.Event{
		Type:    "sync.job.complete",
		Source:  "accounts",
		Payload: syncJobPayload(jobID, "accounts", "pull_accounts", "completed"),
	})

	entries, err := database.GetRecentJobLog(a.DB, 20)
	if err != nil {
		t.Fatalf("failed to read job log entries: %v", err)
	}
	preMetric := findJobLogByCorrelation(t, entries, jobID)
	if preMetric.CompletedAt != nil {
		t.Fatalf("expected job to remain unfinalized until metric terminal event")
	}

	a.recordJobLogMetricEvent(events.Event{
		Type:    "pull.start",
		Source:  "accounts",
		Payload: events.PullStartPayload{JobID: jobID},
	})
	a.recordJobLogMetricEvent(events.Event{
		Type:    "pull.ids_fetched",
		Source:  "accounts",
		Payload: events.ResourceIDsFetchedPayload{Count: 3, JobID: jobID},
	})
	a.recordJobLogMetricEvent(events.Event{
		Type:   "pull.group.complete",
		Source: "accounts",
		Payload: events.CompletionPayload{
			Success: true,
			Count:   2,
			JobID:   jobID,
		},
	})

	entries, err = database.GetRecentJobLog(a.DB, 20)
	if err != nil {
		t.Fatalf("failed to read finalized job log entries: %v", err)
	}
	finalized := findJobLogByCorrelation(t, entries, jobID)
	if finalized.CompletedAt == nil {
		t.Fatalf("expected job to be finalized after terminal metric event")
	}
	if finalized.Status != "completed_with_errors" {
		t.Fatalf("expected status %q, got %q", "completed_with_errors", finalized.Status)
	}
	if finalized.ItemsProcessed != 2 {
		t.Fatalf("expected items processed %d, got %d", 2, finalized.ItemsProcessed)
	}
	if finalized.ErrorCount != 1 {
		t.Fatalf("expected error count %d, got %d", 1, finalized.ErrorCount)
	}
}

func TestMetricEventsUsePayloadCorrelationForTargetBinding(t *testing.T) {
	a := newJobLogTrackingTestApp(t)
	jobA := "sync_job_a"
	jobB := "sync_job_b"

	a.recordSyncJobEvent(events.Event{
		Type:    "sync.job.start",
		Source:  "accounts",
		Payload: syncJobPayload(jobA, "accounts", "pull_accounts", "running"),
	})
	a.recordSyncJobEvent(events.Event{
		Type:    "sync.job.start",
		Source:  "accounts",
		Payload: syncJobPayload(jobB, "accounts", "pull_accounts", "running"),
	})

	a.recordSyncJobEvent(events.Event{
		Type:    "sync.job.complete",
		Source:  "accounts",
		Payload: syncJobPayload(jobA, "accounts", "pull_accounts", "completed"),
	})
	a.recordJobLogMetricEvent(events.Event{
		Type:    "pull.start",
		Source:  "accounts",
		Payload: events.PullStartPayload{JobID: jobA},
	})
	a.recordJobLogMetricEvent(events.Event{
		Type:    "pull.ids_fetched",
		Source:  "accounts",
		Payload: events.ResourceIDsFetchedPayload{Count: 4, JobID: jobA},
	})
	a.recordJobLogMetricEvent(events.Event{
		Type:   "pull.group.complete",
		Source: "accounts",
		Payload: events.CompletionPayload{
			Success: true,
			Count:   4,
			JobID:   jobA,
		},
	})

	a.recordSyncJobEvent(events.Event{
		Type:    "sync.job.complete",
		Source:  "accounts",
		Payload: syncJobPayload(jobB, "accounts", "pull_accounts", "completed"),
	})
	a.recordJobLogMetricEvent(events.Event{
		Type:    "pull.start",
		Source:  "accounts",
		Payload: events.PullStartPayload{JobID: jobB},
	})
	a.recordJobLogMetricEvent(events.Event{
		Type:    "pull.ids_fetched",
		Source:  "accounts",
		Payload: events.ResourceIDsFetchedPayload{Count: 1, JobID: jobB},
	})
	a.recordJobLogMetricEvent(events.Event{
		Type:   "pull.group.complete",
		Source: "accounts",
		Payload: events.CompletionPayload{
			Success: true,
			Count:   1,
			JobID:   jobB,
		},
	})

	entries, err := database.GetRecentJobLog(a.DB, 20)
	if err != nil {
		t.Fatalf("failed to read job log entries: %v", err)
	}
	entryA := findJobLogByCorrelation(t, entries, jobA)
	entryB := findJobLogByCorrelation(t, entries, jobB)

	if entryA.ItemsProcessed != 4 {
		t.Fatalf("expected job A processed count %d, got %d", 4, entryA.ItemsProcessed)
	}
	if entryB.ItemsProcessed != 1 {
		t.Fatalf("expected job B processed count %d, got %d", 1, entryB.ItemsProcessed)
	}
}
