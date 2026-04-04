package database

import (
	"badgermaps/app/state"
	"path/filepath"
	"testing"
)

func TestBackfillJobLogFromSyncHistoryIsIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "job_log_backfill.db")

	db, err := NewDB(&DBConfig{
		Type: "sqlite3",
		Path: dbPath,
	})
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect db: %v", err)
	}
	defer db.Close()
	db.SetConnected(true)

	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}
	if err := RunCommand(db, "CreateSyncHistoryTable"); err != nil {
		t.Fatalf("failed to create legacy sync history table: %v", err)
	}

	correlationID := "sync_backfill_1"
	if _, err := InsertSyncHistory(db, &SyncHistoryEntry{
		CorrelationID:     correlationID,
		RunType:           "sync",
		Direction:         "pull",
		Source:            "accounts",
		Initiator:         "manual",
		Status:            "completed_with_errors",
		ItemsProcessed:    7,
		ErrorCount:        1,
		StartedAtTimezone: "UTC",
		Summary:           "Pulled 7 accounts",
		Details:           "1 account failed",
	}); err != nil {
		t.Fatalf("failed to insert sync history seed row: %v", err)
	}

	rowsInserted, err := BackfillJobLogFromSyncHistory(db)
	if err != nil {
		t.Fatalf("first backfill failed: %v", err)
	}
	if rowsInserted != 1 {
		t.Fatalf("expected first backfill to insert 1 row, got %d", rowsInserted)
	}

	rowsInserted, err = BackfillJobLogFromSyncHistory(db)
	if err != nil {
		t.Fatalf("second backfill failed: %v", err)
	}
	if rowsInserted != 0 {
		t.Fatalf("expected second backfill to insert 0 rows, got %d", rowsInserted)
	}

	entries, err := GetRecentJobLog(db, 10)
	if err != nil {
		t.Fatalf("failed to read job log rows: %v", err)
	}

	var found *JobLogEntry
	for i := range entries {
		if entries[i].CorrelationID == correlationID {
			found = &entries[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected to find job log row for correlation %q", correlationID)
	}

	if found.ParentCorrelationID != "" {
		t.Fatalf("expected empty parent correlation id, got %q", found.ParentCorrelationID)
	}
	if found.RootCorrelationID != correlationID {
		t.Fatalf("expected root correlation id %q, got %q", correlationID, found.RootCorrelationID)
	}
	if found.JobKind != "sync" {
		t.Fatalf("expected job kind %q, got %q", "sync", found.JobKind)
	}
	if found.Mode != "pull" {
		t.Fatalf("expected mode %q, got %q", "pull", found.Mode)
	}
	if found.ActionType != "" {
		t.Fatalf("expected empty action type, got %q", found.ActionType)
	}
	if found.CommandText != "" {
		t.Fatalf("expected empty command text, got %q", found.CommandText)
	}
	if found.ItemsProcessed != 7 {
		t.Fatalf("expected items processed %d, got %d", 7, found.ItemsProcessed)
	}
	if found.ErrorCount != 1 {
		t.Fatalf("expected error count %d, got %d", 1, found.ErrorCount)
	}
}
