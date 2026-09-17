package database

import (
	"strings"
	"testing"
)

func TestJobLogCreateTableIncludesActionAndTimezoneColumns(t *testing.T) {
	tests := []struct {
		name       string
		read       func(string) ([]byte, error)
		path       string
		expectsAll []string
	}{
		{
			name: "sqlite",
			read: sqlite3FS.ReadFile,
			path: "sqlite3/CreateJobLogTable.sql",
			expectsAll: []string{
				"ACTIONTYPE",
				"COMMANDTEXT",
				"STARTEDATTIMEZONE",
				"COMPLETEDATTIMEZONE",
			},
		},
		{
			name: "postgres",
			read: postgresFS.ReadFile,
			path: "postgres/CreateJobLogTable.sql",
			expectsAll: []string{
				"ACTIONTYPE",
				"COMMANDTEXT",
				"STARTEDATTIMEZONE",
				"COMPLETEDATTIMEZONE",
			},
		},
		{
			name: "mssql",
			read: mssqlFS.ReadFile,
			path: "mssql/CreateJobLogTable.sql",
			expectsAll: []string{
				"ACTIONTYPE",
				"COMMANDTEXT",
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

func TestJobLogInsertAndCompleteSQLUseTimezoneColumns(t *testing.T) {
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
			insertPath:          "sqlite3/InsertJobLog.sql",
			completePath:        "sqlite3/CompleteJobLog.sql",
			insertMustContain:   "STARTEDATTIMEZONE",
			completeMustContain: "COMPLETEDATTIMEZONE",
		},
		{
			name:                "postgres",
			read:                postgresFS.ReadFile,
			insertPath:          "postgres/InsertJobLog.sql",
			completePath:        "postgres/CompleteJobLog.sql",
			insertMustContain:   "STARTEDATTIMEZONE",
			completeMustContain: "COMPLETEDATTIMEZONE",
		},
		{
			name:                "mssql",
			read:                mssqlFS.ReadFile,
			insertPath:          "mssql/InsertJobLog.sql",
			completePath:        "mssql/CompleteJobLog.sql",
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

func TestPostgresJobLogUsesUTCTimestamps(t *testing.T) {
	createContent, err := postgresFS.ReadFile("postgres/CreateJobLogTable.sql")
	if err != nil {
		t.Fatalf("failed to read postgres create job log SQL: %v", err)
	}

	createSQL := strings.ToUpper(string(createContent))
	if !strings.Contains(createSQL, "STARTEDAT TIMESTAMP DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')") {
		t.Fatalf("expected Postgres JobLog StartedAt default to be UTC")
	}
	if strings.Contains(createSQL, "STARTEDAT TIMESTAMP DEFAULT CURRENT_TIMESTAMP") {
		t.Fatalf("did not expect Postgres JobLog StartedAt default to use session-local CURRENT_TIMESTAMP")
	}

	completeContent, err := postgresFS.ReadFile("postgres/CompleteJobLog.sql")
	if err != nil {
		t.Fatalf("failed to read postgres complete job log SQL: %v", err)
	}

	completeSQL := strings.ToUpper(string(completeContent))
	if !strings.Contains(completeSQL, "COMPLETEDAT = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')") {
		t.Fatalf("expected Postgres JobLog CompletedAt to use UTC conversion")
	}
	if strings.Contains(completeSQL, "COMPLETEDAT = CURRENT_TIMESTAMP") {
		t.Fatalf("did not expect Postgres JobLog CompletedAt to use session-local CURRENT_TIMESTAMP")
	}
}

func TestMSSQLJobLogCompleteUsesUTCCompletionTimestamp(t *testing.T) {
	content, err := mssqlFS.ReadFile("mssql/CompleteJobLog.sql")
	if err != nil {
		t.Fatalf("failed to read MSSQL complete job log SQL: %v", err)
	}

	sqlText := strings.ToUpper(string(content))
	if !strings.Contains(sqlText, "COMPLETEDAT = SYSUTCDATETIME()") {
		t.Fatalf("expected MSSQL JobLog completion timestamp to use SYSUTCDATETIME()")
	}
	if strings.Contains(sqlText, "COMPLETEDAT = SYSDATETIME()") {
		t.Fatalf("did not expect MSSQL JobLog completion timestamp to use SYSDATETIME()")
	}
}
