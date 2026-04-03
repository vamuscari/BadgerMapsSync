package database

import (
	"strings"
	"testing"
)

func TestSyncHistoryCreateTableIncludesTimezoneColumns(t *testing.T) {
	tests := []struct {
		name       string
		read       func(string) ([]byte, error)
		path       string
		expectsAll []string
	}{
		{
			name: "sqlite",
			read: sqlite3FS.ReadFile,
			path: "sqlite3/CreateSyncHistoryTable.sql",
			expectsAll: []string{
				"STARTEDATTIMEZONE",
				"COMPLETEDATTIMEZONE",
			},
		},
		{
			name: "postgres",
			read: postgresFS.ReadFile,
			path: "postgres/CreateSyncHistoryTable.sql",
			expectsAll: []string{
				"STARTEDATTIMEZONE",
				"COMPLETEDATTIMEZONE",
			},
		},
		{
			name: "mssql",
			read: mssqlFS.ReadFile,
			path: "mssql/CreateSyncHistoryTable.sql",
			expectsAll: []string{
				"STARTEDATTIMEZONE",
				"COMPLETEDATTIMEZONE",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content, err := tc.read(tc.path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tc.path, err)
			}
			sqlText := strings.ToUpper(string(content))
			for _, expected := range tc.expectsAll {
				if !strings.Contains(sqlText, expected) {
					t.Fatalf("expected %s to contain %q", tc.path, expected)
				}
			}
		})
	}
}

func TestSyncHistoryInsertAndCompleteSQLUseTimezoneColumns(t *testing.T) {
	tests := []struct {
		name                string
		read                func(string) ([]byte, error)
		insertPath          string
		completePath        string
		insertMustContain   string
		completeMustContain string
	}{
		{
			name:                "sqlite",
			read:                sqlite3FS.ReadFile,
			insertPath:          "sqlite3/InsertSyncHistory.sql",
			completePath:        "sqlite3/CompleteSyncHistory.sql",
			insertMustContain:   "STARTEDATTIMEZONE",
			completeMustContain: "COMPLETEDATTIMEZONE",
		},
		{
			name:                "postgres",
			read:                postgresFS.ReadFile,
			insertPath:          "postgres/InsertSyncHistory.sql",
			completePath:        "postgres/CompleteSyncHistory.sql",
			insertMustContain:   "\"STARTEDATTIMEZONE\"",
			completeMustContain: "\"COMPLETEDATTIMEZONE\"",
		},
		{
			name:                "mssql",
			read:                mssqlFS.ReadFile,
			insertPath:          "mssql/InsertSyncHistory.sql",
			completePath:        "mssql/CompleteSyncHistory.sql",
			insertMustContain:   "STARTEDATTIMEZONE",
			completeMustContain: "COMPLETEDATTIMEZONE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			insertContent, err := tc.read(tc.insertPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tc.insertPath, err)
			}
			completeContent, err := tc.read(tc.completePath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tc.completePath, err)
			}

			insertSQL := strings.ToUpper(string(insertContent))
			completeSQL := strings.ToUpper(string(completeContent))

			if !strings.Contains(insertSQL, strings.ToUpper(tc.insertMustContain)) {
				t.Fatalf("expected %s to contain %q", tc.insertPath, tc.insertMustContain)
			}
			if !strings.Contains(completeSQL, strings.ToUpper(tc.completeMustContain)) {
				t.Fatalf("expected %s to contain %q", tc.completePath, tc.completeMustContain)
			}
		})
	}
}
