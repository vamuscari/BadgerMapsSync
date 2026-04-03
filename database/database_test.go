package database

import (
	"badgermaps/app/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			break
		}
		if wd == filepath.Dir(wd) {
			os.Exit(1)
		}
		wd = filepath.Dir(wd)
	}
	if err := os.Chdir(wd); err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestSQLFiles(t *testing.T) {
	baseExpectedFiles := []string{
		"AddSyncHistoryStartedAtTimezoneColumn.sql",
		"AddSyncHistoryCompletedAtTimezoneColumn.sql",
		"CheckColumnExists.sql",
		"CheckIndexExists.sql",
		"CheckTableExists.sql",
		"CreateAccountCheckinsPendingChangesTable.sql",
		"CreateAccountCheckinsTable.sql",
		"CreateAccountLocationsTable.sql",
		"CreateAccountsPendingChangesTable.sql",
		"CreateAccountsTable.sql",
		"CreateDataSetValuesTable.sql",
		"CreateDataSetsTable.sql",
		"CreateIndexes.sql",
		"CreateRouteWaypointsTable.sql",
		"CreateRoutesTable.sql",
		"CreateSyncHistoryTable.sql",
		"CreateJobLogTable.sql",
		"CreateUserProfilesTable.sql",
		"DeleteAccountLocations.sql",
		"DeleteDataSetValues.sql",
		"DeleteDataSets.sql",
		"DeleteRouteWaypoints.sql",
		"GetAccountById.sql",
		"GetAllAccountIds.sql",
		"GetCheckinById.sql",
		"GetPendingAccountChanges.sql",
		"GetPendingCheckinChanges.sql",
		"GetProfile.sql",
		"GetRouteById.sql",
		"GetTableColumns.sql",
		"InsertAccountLocations.sql",
		"InsertDataSetValues.sql",
		"InsertDataSets.sql",
		"InsertRouteWaypoints.sql",
		"MergeAccountCheckins.sql",
		"MergeAccountsBasic.sql",
		"MergeAccountsDetailed.sql",
		"MergeRoutes.sql",
		"MergeUserProfiles.sql",
		"SearchAccounts.sql",
		"SearchRoutes.sql",
		"SearchCheckins.sql",
		"UpdatePendingChangeStatus.sql",
		"CreateAccountsWithLabelsView.sql",
		"CreateFieldMapsTable.sql",
		"InsertFieldMaps.sql",
		"UpdateFieldMapsFromDatasets.sql",
		"CreateFieldMapsUpdateTrigger.sql",
		"CheckViewExists.sql",
		"CreateConfigurationsTable.sql",
		"InsertConfigurations.sql",
		"UpdateConfiguration.sql",
		"CreateCommandLogTable.sql",
		"CreateJobLogIndexes.sql",
		"CompleteSyncHistory.sql",
		"CompleteJobLog.sql",
		"GetRecentSyncHistory.sql",
		"GetRecentJobLog.sql",
		"InsertSyncHistory.sql",
		"InsertJobLog.sql",
		"BackfillJobLogFromSyncHistory.sql",
		"CreateWebhookLogTable.sql",
		"GetWebhookLog.sql",
		"UpdateSyncHistoryMetrics.sql",
		"UpdateJobLogMetrics.sql",
	}

	postgresMssqlExtraFiles := []string{
		"CreateDatasetsUpdateTrigger.sql",
		"CheckProcedureExists.sql",
		"CheckTriggerExists.sql",
	}
	mssqlExtraFiles := []string{
		"AddAccountCheckinsPendingChangesAccountIdColumn.sql",
		"AddAccountCheckinsEndpointTypeColumn.sql",
		"AddAccountCheckinsPendingChangesEndpointTypeColumn.sql",
	}

	checkFiles := func(t *testing.T, dir string, expected []string) {
		actualFiles := make(map[string]bool)
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read directory %s: %v", dir, err)
		}

		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
				actualFiles[file.Name()] = true
			}
		}

		for _, f := range expected {
			if _, ok := actualFiles[f]; !ok {
				t.Errorf("missing file in %s: %s", dir, f)
			}
		}

		if len(expected) != len(actualFiles) {
			for f := range actualFiles {
				found := false
				for _, e := range expected {
					if f == e {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("unexpected file in %s: %s", dir, f)
				}
			}
		}
	}

	t.Run("sqlite3", func(t *testing.T) {
		checkFiles(t, filepath.Join("database", "sqlite3"), baseExpectedFiles)
	})

	t.Run("postgres", func(t *testing.T) {
		checkFiles(t, filepath.Join("database", "postgres"), append(baseExpectedFiles, postgresMssqlExtraFiles...))
	})

	t.Run("mssql", func(t *testing.T) {
		checkFiles(t, filepath.Join("database", "mssql"), append(append(baseExpectedFiles, postgresMssqlExtraFiles...), mssqlExtraFiles...))
	})
}

func TestExtractMSSQLCreateTableColumnDefinitions(t *testing.T) {
	mssqlDB := &MSSQLConfig{}
	sqlText := mssqlDB.GetSQL("CreateAccountCheckinsPendingChangesTable")
	if sqlText == "" {
		t.Fatalf("failed to load MSSQL create table SQL for AccountCheckinsPendingChanges")
	}

	definitions, err := extractMSSQLCreateTableColumnDefinitions(sqlText)
	if err != nil {
		t.Fatalf("failed to parse MSSQL create table SQL: %v", err)
	}

	definitionsByColumn := make(map[string]string, len(definitions))
	for _, definition := range definitions {
		definitionsByColumn[strings.ToLower(definition.Name)] = definition.Definition
	}

	if _, ok := definitionsByColumn["accountid"]; !ok {
		t.Fatalf("expected parsed definitions to include AccountId")
	}

	if crmDef, ok := definitionsByColumn["crmid"]; !ok {
		t.Fatalf("expected parsed definitions to include CrmId")
	} else if !strings.Contains(strings.ToUpper(crmDef), "NVARCHAR") {
		t.Fatalf("expected CrmId definition to include NVARCHAR, got: %s", crmDef)
	}
}

func TestBuildMSSQLFallbackColumnDefinition(t *testing.T) {
	tests := []struct {
		name        string
		definition  string
		contains    []string
		notContains []string
	}{
		{
			name:        "not null to nullable",
			definition:  "INT NOT NULL",
			contains:    []string{"INT", "NULL"},
			notContains: []string{"NOT NULL"},
		},
		{
			name:        "strip identity and primary key",
			definition:  "INT IDENTITY(1,1) PRIMARY KEY",
			contains:    []string{"INT", "NULL"},
			notContains: []string{"IDENTITY", "PRIMARY KEY"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fallback := buildMSSQLFallbackColumnDefinition(testCase.definition)
			upperFallback := strings.ToUpper(fallback)

			for _, expected := range testCase.contains {
				if !strings.Contains(upperFallback, strings.ToUpper(expected)) {
					t.Fatalf("expected fallback definition %q to contain %q", fallback, expected)
				}
			}

			for _, unexpected := range testCase.notContains {
				if strings.Contains(upperFallback, strings.ToUpper(unexpected)) {
					t.Fatalf("expected fallback definition %q to not contain %q", fallback, unexpected)
				}
			}
		})
	}
}

func TestEnforceSchema(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := &DBConfig{
		Type: "sqlite3",
		Path: dbPath,
	}

	s := state.NewState()
	db, err := NewDB(config)
	if err != nil {
		t.Fatalf("Failed to load database settings: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.EnforceSchema(s); err != nil {
		t.Fatalf("EnforceSchema failed: %v", err)
	}

	sqlDB := db.GetDB()
	rows, err := sqlDB.Query("SELECT COUNT(*) FROM FieldMaps")
	if err != nil {
		t.Fatalf("Failed to query FieldMaps table: %v", err)
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			t.Fatalf("Failed to scan count from FieldMaps table: %v", err)
		}
	}

	if count < 5 {
		t.Errorf("FieldMaps table has %d rows, expected at least 5", count)
	}
}

func TestIsConnected(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := &DBConfig{
		Type: "sqlite3",
		Path: dbPath,
	}

	db, err := NewDB(config)
	if err != nil {
		t.Fatalf("Failed to load database settings: %v", err)
	}

	if db.IsConnected() {
		t.Errorf("Expected IsConnected to be false for a new database connection")
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.TestConnection(); err != nil {
		t.Fatalf("Failed to test database connection: %v", err)
	}

	if !db.IsConnected() {
		t.Errorf("Expected IsConnected to be true after a successful connection")
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database connection: %v", err)
	}

	if db.IsConnected() {
		t.Errorf("Expected IsConnected to be false after closing the connection")
	}
}
