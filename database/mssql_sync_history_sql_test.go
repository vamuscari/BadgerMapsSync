package database

import (
	"strings"
	"testing"
)

func TestMSSQLCompleteSyncHistoryUsesUTCCompletionTimestamp(t *testing.T) {
	content, err := mssqlFS.ReadFile("mssql/CompleteSyncHistory.sql")
	if err != nil {
		t.Fatalf("failed to read MSSQL complete sync history SQL: %v", err)
	}

	sqlText := strings.ToUpper(string(content))
	if !strings.Contains(sqlText, "COMPLETEDAT = SYSUTCDATETIME()") {
		t.Fatalf("expected MSSQL completion timestamp to use SYSUTCDATETIME()")
	}
	if strings.Contains(sqlText, "COMPLETEDAT = SYSDATETIME()") {
		t.Fatalf("did not expect MSSQL completion timestamp to use SYSDATETIME()")
	}
}
