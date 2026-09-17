package database

import (
	"strings"
	"testing"
)

func TestPostgresCreateSyncHistoryUsesUTCStartedTimestamp(t *testing.T) {
	content, err := postgresFS.ReadFile("postgres/CreateSyncHistoryTable.sql")
	if err != nil {
		t.Fatalf("failed to read postgres create sync history SQL: %v", err)
	}

	sqlText := strings.ToUpper(string(content))
	if !strings.Contains(sqlText, "STARTEDAT TIMESTAMP DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')") {
		t.Fatalf("expected Postgres StartedAt default to be UTC")
	}
	if strings.Contains(sqlText, "STARTEDAT TIMESTAMP DEFAULT CURRENT_TIMESTAMP") {
		t.Fatalf("did not expect Postgres StartedAt default to use session-local CURRENT_TIMESTAMP")
	}
}

func TestPostgresCompleteSyncHistoryUsesUTCCompletionTimestamp(t *testing.T) {
	content, err := postgresFS.ReadFile("postgres/CompleteSyncHistory.sql")
	if err != nil {
		t.Fatalf("failed to read postgres complete sync history SQL: %v", err)
	}

	sqlText := strings.ToUpper(string(content))
	if !strings.Contains(sqlText, "COMPLETEDAT = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')") {
		t.Fatalf("expected Postgres CompletedAt to use UTC conversion")
	}
	if strings.Contains(sqlText, "COMPLETEDAT = CURRENT_TIMESTAMP") {
		t.Fatalf("did not expect Postgres CompletedAt to use session-local CURRENT_TIMESTAMP")
	}
}
