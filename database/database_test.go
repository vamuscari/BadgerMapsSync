package database

import (
	"badgermaps/app/state"
	"os"
	"path/filepath"
	"regexp"
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
		"CreateSchemaMigrationsTable.sql",
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
		"GetSchemaVersion.sql",
		"GetTableColumns.sql",
		"InsertAccountLocations.sql",
		"InsertDataSetValues.sql",
		"InsertDataSets.sql",
		"InsertCommandLog.sql",
		"InsertWebhookLog.sql",
		"InsertRouteWaypoints.sql",
		"InsertSchemaMigration.sql",
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
		"CreateAccountsIndexedView.sql",
		"CreateAccountsIndexedColumnsView.sql",
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
		"CountOrphanedAccountLocations.sql",
		"CountOrphanedCheckins.sql",
		"CountOrphanedRouteWaypoints.sql",
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
	sqliteExtraFiles := []string{
		"GetAccountsIndexedColumns.sql",
		"GetAccountsWithLabelsColumns.sql",
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
		checkFiles(t, filepath.Join("database", "sqlite3"), append(baseExpectedFiles, sqliteExtraFiles...))
	})

	t.Run("postgres", func(t *testing.T) {
		checkFiles(t, filepath.Join("database", "postgres"), append(baseExpectedFiles, postgresMssqlExtraFiles...))
	})

	t.Run("mssql", func(t *testing.T) {
		checkFiles(t, filepath.Join("database", "mssql"), append(append(baseExpectedFiles, postgresMssqlExtraFiles...), mssqlExtraFiles...))
	})
}

func TestPostgreSQLGetSQLRebindsPlaceholders(t *testing.T) {
	db := &PostgreSQLConfig{}
	sqlText := db.GetSQL("UpdateConfiguration")

	if strings.Contains(sqlText, "?") {
		t.Fatalf("expected PostgreSQL placeholders to be rebound, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "$1") || !strings.Contains(sqlText, "$2") {
		t.Fatalf("expected PostgreSQL placeholders $1 and $2, got %q", sqlText)
	}
}

func TestPostgreSQLAssetsUsePortableSourceSyntax(t *testing.T) {
	entries, err := postgresFS.ReadDir("postgres")
	if err != nil {
		t.Fatalf("failed to list PostgreSQL SQL assets: %v", err)
	}

	quotedMixedCase := regexp.MustCompile(`"[A-Z][A-Za-z0-9_]*"`)
	dollarPlaceholder := regexp.MustCompile(`\$[0-9]+`)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := postgresFS.ReadFile("postgres/" + entry.Name())
		if err != nil {
			t.Fatalf("failed to read %s: %v", entry.Name(), err)
		}
		sqlText := string(content)
		for _, match := range quotedMixedCase.FindAllString(sqlText, -1) {
			legacyViewReference := (entry.Name() == "CreateAccountsWithLabelsView.sql" && match == `"AccountsWithLabels"`) ||
				(entry.Name() == "CreateAccountsIndexedView.sql" && match == `"AccountsIndexed"`)
			if !legacyViewReference {
				t.Errorf("%s uses mixed-case quoted identifier %s", entry.Name(), match)
			}
		}
		if match := dollarPlaceholder.FindString(sqlText); match != "" {
			t.Errorf("%s uses driver-specific placeholder %s instead of ?", entry.Name(), match)
		}
	}
}

func TestDatabaseLoggingCommandsAvailableForEveryDialect(t *testing.T) {
	databases := []DB{
		&SQLiteConfig{},
		&PostgreSQLConfig{},
		&MSSQLConfig{},
	}
	for _, db := range databases {
		for _, command := range []string{"InsertCommandLog", "InsertWebhookLog"} {
			sqlText := db.GetSQL(command)
			if sqlText == "" {
				t.Errorf("%s is missing %s", db.GetType(), command)
			}
			if db.GetType() == "postgres" && strings.Contains(sqlText, "?") {
				t.Errorf("PostgreSQL %s placeholders were not rebound: %q", command, sqlText)
			}
		}
	}
}

func TestPostgreSQLAccountsWithLabelsDDLIsDeployable(t *testing.T) {
	db := &PostgreSQLConfig{}
	functionSQL := db.GetSQL("CreateAccountsWithLabelsView")
	triggerSQL := db.GetSQL("CreateDatasetsUpdateTrigger")

	for name, sqlText := range map[string]string{
		"view function":   functionSQL,
		"refresh trigger": triggerSQL,
	} {
		if !strings.Contains(sqlText, "AS $$") || !strings.Contains(sqlText, "$$ LANGUAGE plpgsql") {
			t.Errorf("%s must use valid PostgreSQL dollar quoting", name)
		}
		if strings.Contains(sqlText, `"Accounts"`) || strings.Contains(sqlText, `"DataSets"`) || strings.Contains(sqlText, `"Configurations"`) {
			t.Errorf("%s must reference the lowercase relations created by the schema", name)
		}
	}

	lowerFunctionSQL := strings.ToLower(functionSQL)
	if strings.Contains(lowerFunctionSQL, "drop view if exists accountswithlabels") {
		t.Fatal("refresh must preserve the existing PostgreSQL view and its grants")
	}
	if !strings.Contains(lowerFunctionSQL, "create or replace view accountswithlabels") {
		t.Fatal("expected refresh function to replace the existing view in place")
	}
	if !strings.Contains(functionSQL, `"AccountsWithLabels"`) || !strings.Contains(lowerFunctionSQL, "alter view") {
		t.Fatal("expected refresh function to migrate the legacy quoted view name")
	}
	for _, expected := range []string{"information_schema.table_privileges", "drop view accountswithlabels", "grant_statement"} {
		if !strings.Contains(lowerFunctionSQL, expected) {
			t.Errorf("expected PostgreSQL AccountsWithLabels fallback to contain %q", expected)
		}
	}
	if strings.Contains(lowerFunctionSQL, "drop view accountswithlabels cascade") {
		t.Fatal("AccountsWithLabels fallback must not remove dependent objects")
	}
	if !strings.Contains(strings.ToLower(functionSQL), "order by c.ordinal_position") {
		t.Fatal("expected deterministic account-column ordering")
	}
}

func TestMSSQLAccountsWithLabelsRefreshPreservesViewObject(t *testing.T) {
	sqlText := (&MSSQLConfig{}).GetSQL("CreateAccountsWithLabelsView")
	lowerSQL := strings.ToLower(sqlText)

	if strings.Contains(lowerSQL, "drop view dbo.accountswithlabels") {
		t.Fatal("refresh must not drop the view because doing so removes object grants")
	}
	if !strings.Contains(lowerSQL, "alter view dbo.accountswithlabels") {
		t.Fatal("expected existing views to be refreshed with ALTER VIEW")
	}
	if !strings.Contains(lowerSQL, "datalength(ds.label)") {
		t.Fatal("expected SQL Server labels to be checked before QUOTENAME")
	}
	if !strings.Contains(lowerSQL, "join information_schema.columns mapped_column") {
		t.Fatal("expected label limits to apply only to DataSets mapped to Accounts columns")
	}
}

func TestPostgreSQLAccountsIndexedDDLUsesProfileOrder(t *testing.T) {
	viewSQL := (&PostgreSQLConfig{}).GetSQL("CreateAccountsIndexedView")
	triggerSQL := (&PostgreSQLConfig{}).GetSQL("CreateDatasetsUpdateTrigger")
	lowerViewSQL := strings.ToLower(viewSQL)
	lowerTriggerSQL := strings.ToLower(triggerSQL)

	for _, expected := range []string{
		"create or replace function accountsindexedview()",
		"create or replace view accountsindexed",
		"ds.position",
		"customtext%",
		"customnumeric%",
	} {
		if !strings.Contains(lowerViewSQL, expected) {
			t.Errorf("expected PostgreSQL AccountsIndexed DDL to contain %q", expected)
		}
	}
	if strings.Contains(lowerViewSQL, "drop view if exists accountsindexed") {
		t.Fatal("AccountsIndexed refresh must preserve the existing PostgreSQL view and its grants")
	}
	if !strings.Contains(viewSQL, `"AccountsIndexed"`) || !strings.Contains(lowerViewSQL, "alter view") {
		t.Fatal("expected AccountsIndexed refresh to migrate the legacy quoted view name")
	}
	for _, expected := range []string{"information_schema.table_privileges", "drop view accountsindexed", "grant_statement"} {
		if !strings.Contains(lowerViewSQL, expected) {
			t.Errorf("expected PostgreSQL AccountsIndexed fallback to contain %q", expected)
		}
	}
	if strings.Contains(lowerViewSQL, "drop view accountsindexed cascade") {
		t.Fatal("AccountsIndexed fallback must not remove dependent objects")
	}
	if !strings.Contains(viewSQL, "AS $$") || !strings.Contains(viewSQL, "$$ LANGUAGE plpgsql") {
		t.Fatal("AccountsIndexed function must use valid PostgreSQL dollar quoting")
	}
	if strings.Contains(lowerTriggerSQL, "perform accountswithlabelsview()") || strings.Contains(lowerTriggerSQL, "perform accountsindexedview()") {
		t.Fatal("DataSets trigger must not rebuild generated views for every profile metadata statement")
	}
}

func TestMSSQLAccountsIndexedDDLUsesProfileOrder(t *testing.T) {
	viewSQL := (&MSSQLConfig{}).GetSQL("CreateAccountsIndexedView")
	triggerSQL := (&MSSQLConfig{}).GetSQL("CreateDatasetsUpdateTrigger")
	lowerViewSQL := strings.ToLower(viewSQL)
	lowerTriggerSQL := strings.ToLower(triggerSQL)

	for _, expected := range []string{
		"alter procedure dbo.accountsindexedview",
		"alter view dbo.accountsindexed",
		"ds.position",
		"customtext%",
		"customnumeric%",
	} {
		if !strings.Contains(lowerViewSQL, expected) {
			t.Errorf("expected SQL Server AccountsIndexed DDL to contain %q", expected)
		}
	}
	if strings.Contains(lowerViewSQL, "drop view dbo.accountsindexed") {
		t.Fatal("AccountsIndexed refresh must preserve the existing SQL Server view object")
	}
	if !strings.Contains(lowerViewSQL, "join information_schema.columns mapped_column") {
		t.Fatal("expected label limits to apply only to DataSets mapped to Accounts columns")
	}
	if strings.Contains(lowerTriggerSQL, "exec dbo.accountswithlabelsview") || strings.Contains(lowerTriggerSQL, "exec dbo.accountsindexedview") {
		t.Fatal("DataSets trigger must not rebuild generated views for every profile metadata statement")
	}
}

func TestMSSQLAccountsIndexedColumnsDDLUsesCompositePosition(t *testing.T) {
	sqlText := strings.ToLower((&MSSQLConfig{}).GetSQL("CreateAccountsIndexedColumnsView"))
	for _, expected := range []string{
		"create or alter view dbo.accountsindexedcolumns",
		"row_number() over",
		"account_column.lastcustom",
		"data_set.position",
		"account_column.ordinal_position",
		"as [position]",
	} {
		if !strings.Contains(sqlText, expected) {
			t.Errorf("expected SQL Server AccountsIndexedColumns DDL to contain %q", expected)
		}
	}
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

	jobLogExists, err := db.TableExists("JobLog")
	if err != nil {
		t.Fatalf("failed to check JobLog existence: %v", err)
	}
	if !jobLogExists {
		t.Fatalf("expected JobLog table to exist")
	}

	syncHistoryExists, err := db.TableExists("SyncHistory")
	if err != nil {
		t.Fatalf("failed to check SyncHistory existence: %v", err)
	}
	if syncHistoryExists {
		t.Fatalf("expected SyncHistory table to be omitted from enforced schema")
	}
}

func TestGetProfileUsesStoredSchemaColumns(t *testing.T) {
	db, err := NewDB(&DBConfig{Type: "sqlite3", Path: filepath.Join(t.TempDir(), "profile.db")})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()
	if err := db.EnforceSchema(&state.State{Quiet: true}); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}

	_, err = db.GetDB().Exec(`
		INSERT INTO UserProfiles (
			ProfileId, Email, FirstName, LastName, CRMEditableFieldsList,
			CompanyId, CompanyName, CompanyShortName
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, 42, "owner@example.com", "Test", "Owner", "email,phone_number", 7, "Example Company", "EX")
	if err != nil {
		t.Fatalf("failed to seed profile: %v", err)
	}

	profile, err := GetProfile(db)
	if err != nil {
		t.Fatalf("GetProfile returned an error: %v", err)
	}
	if got, want := profile.ProfileId.Int64, int64(42); got != want {
		t.Fatalf("expected profile id %d, got %d", want, got)
	}
	if got, want := profile.Company.Name.String, "Example Company"; got != want {
		t.Fatalf("expected company name %q, got %q", want, got)
	}
	if len(profile.CRMEditableFieldsList) != 2 || profile.CRMEditableFieldsList[0].String != "email" || profile.CRMEditableFieldsList[1].String != "phone_number" {
		t.Fatalf("unexpected CRM editable fields: %#v", profile.CRMEditableFieldsList)
	}
}

func TestEnforceSchemaRefreshesSQLiteAccountsWithLabels(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "labels.db")
	db, err := NewDB(&DBConfig{Type: "sqlite3", Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	quietState := &state.State{Quiet: true}
	if err := db.EnforceSchema(quietState); err != nil {
		t.Fatalf("failed to enforce initial schema: %v", err)
	}

	sqlDB := db.GetDB()
	if _, err := sqlDB.Exec("INSERT INTO UserProfiles (ProfileId) VALUES (?)", 42); err != nil {
		t.Fatalf("failed to seed profile: %v", err)
	}
	if _, err := sqlDB.Exec("UPDATE Configurations SET SettingValue = ? WHERE SettingKey = 'ApiProfileId'", "42"); err != nil {
		t.Fatalf("failed to select profile: %v", err)
	}
	if _, err := sqlDB.Exec("INSERT INTO DataSets (Name, ProfileId, Label, AccountField) VALUES (?, ?, ?, ?)", "ct", 42, "Customer Tier", "CustomText"); err != nil {
		t.Fatalf("failed to seed label: %v", err)
	}

	if err := db.EnforceSchema(quietState); err != nil {
		t.Fatalf("failed to re-enforce schema: %v", err)
	}

	var count int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('AccountsWithLabels') WHERE name = ?", "Customer Tier").Scan(&count); err != nil {
		t.Fatalf("failed to inspect labeled view: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected refreshed view to expose Customer Tier, found %d matching columns", count)
	}
}

func TestEnforceSchemaCreatesSQLiteAccountsIndexed(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "indexed.db")
	db, err := NewDB(&DBConfig{Type: "sqlite3", Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	quietState := &state.State{Quiet: true}
	if err := db.EnforceSchema(quietState); err != nil {
		t.Fatalf("failed to enforce initial schema: %v", err)
	}

	sqlDB := db.GetDB()
	if _, err := sqlDB.Exec("INSERT INTO UserProfiles (ProfileId) VALUES (?), (?)", 42, 43); err != nil {
		t.Fatalf("failed to seed profile: %v", err)
	}
	if _, err := sqlDB.Exec("UPDATE Configurations SET SettingValue = ? WHERE SettingKey = 'ApiProfileId'", "42"); err != nil {
		t.Fatalf("failed to select profile: %v", err)
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO DataSets (Name, ProfileId, Label, Position, AccountField) VALUES
			('email', 42, 'Work Email', 99, 'Email'),
			('ct2', 42, 'First Custom', 1, 'CustomText2'),
			('cn', 42, 'Second Custom', 2, 'CustomNumeric'),
			('ct', 43, 'Other Profile Custom', 0, 'CustomText')
	`); err != nil {
		t.Fatalf("failed to seed data sets: %v", err)
	}

	if err := db.EnforceSchema(quietState); err != nil {
		t.Fatalf("failed to refresh schema: %v", err)
	}

	rows, err := sqlDB.Query("SELECT name FROM pragma_table_info('AccountsIndexed') ORDER BY cid")
	if err != nil {
		t.Fatalf("failed to inspect AccountsIndexed: %v", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("failed to scan AccountsIndexed column: %v", err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to read AccountsIndexed columns: %v", err)
	}

	want := []string{
		"AccountId", "FirstName", "LastName", "FullName", "PhoneNumber", "Work Email", "CustomerId", "Notes",
		"OriginalAddress", "CrmId", "AccountOwner", "DaysSinceLastCheckin", "LastCheckinDate", "LastModifiedDate",
		"FollowUpDate", "First Custom", "Second Custom", "CreatedAt", "UpdatedAt",
	}
	if strings.Join(columns, "|") != strings.Join(want, "|") {
		t.Fatalf("unexpected AccountsIndexed columns:\n got: %v\nwant: %v", columns, want)
	}
}

func TestAccountsIndexedColumnsUsesActiveProfile(t *testing.T) {
	type expectedColumn struct {
		columnType string
		position   int
	}

	db, err := NewDB(&DBConfig{Type: "sqlite3", Path: filepath.Join(t.TempDir(), "indexed-columns.db")})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	quietState := &state.State{Quiet: true}
	if err := db.EnforceSchema(quietState); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}

	sqlDB := db.GetDB()
	if _, err := sqlDB.Exec("INSERT INTO UserProfiles (ProfileId) VALUES (?), (?)", 42, 43); err != nil {
		t.Fatalf("failed to seed profiles: %v", err)
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO DataSets (Name, ProfileId, Label, Position, Type, AccountField) VALUES
			('email', 42, 'Work Email', 1, 'email', 'Email'),
			('revenue', 42, 'Annual Revenue', 1, 'numeric', 'CustomNumeric'),
			('tier', 42, 'Customer Tier', 2, 'select', 'CustomText2'),
			('score', 43, 'Account Score', 1, 'numeric', 'CustomNumeric')
	`); err != nil {
		t.Fatalf("failed to seed profile fields: %v", err)
	}

	assertColumns := func(profileID string, expected map[string]expectedColumn, excluded map[string]struct{}) {
		t.Helper()
		if _, err := sqlDB.Exec("UPDATE Configurations SET SettingValue = ? WHERE SettingKey = 'ApiProfileId'", profileID); err != nil {
			t.Fatalf("failed to select profile %s: %v", profileID, err)
		}

		rows, err := sqlDB.Query("SELECT Name, Type, Position FROM AccountsIndexedColumns ORDER BY Position")
		if err != nil {
			t.Fatalf("failed to query AccountsIndexedColumns for profile %s: %v", profileID, err)
		}
		defer rows.Close()

		resultColumns, err := rows.Columns()
		if err != nil {
			t.Fatalf("failed to inspect AccountsIndexedColumns result: %v", err)
		}
		if strings.Join(resultColumns, "|") != "Name|Type|Position" {
			t.Fatalf("unexpected AccountsIndexedColumns result columns: %v", resultColumns)
		}

		got := make(map[string]expectedColumn)
		wantPosition := 1
		for rows.Next() {
			var name, columnType string
			var position int
			if err := rows.Scan(&name, &columnType, &position); err != nil {
				t.Fatalf("failed to scan AccountsIndexedColumns row: %v", err)
			}
			if position != wantPosition {
				t.Errorf("AccountsIndexedColumns position = %d, want contiguous position %d", position, wantPosition)
			}
			wantPosition++
			got[name] = expectedColumn{columnType: columnType, position: position}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("failed to read AccountsIndexedColumns rows: %v", err)
		}

		for name, column := range expected {
			if got[name] != column {
				t.Errorf("AccountsIndexedColumns[%q] = %#v, want %#v", name, got[name], column)
			}
		}
		for name := range excluded {
			if _, ok := got[name]; ok {
				t.Errorf("AccountsIndexedColumns unexpectedly contains %q for profile %s", name, profileID)
			}
		}
	}

	assertColumns("42", map[string]expectedColumn{
		"AccountId":      {columnType: "INTEGER", position: 1},
		"Work Email":     {columnType: "email", position: 6},
		"Annual Revenue": {columnType: "numeric", position: 16},
		"Customer Tier":  {columnType: "select", position: 17},
		"CreatedAt":      {columnType: "DATETIME", position: 18},
	}, map[string]struct{}{
		"Email":         {},
		"Account Score": {},
		"CustomText":    {},
	})
	assertColumns("43", map[string]expectedColumn{
		"AccountId":     {columnType: "INTEGER", position: 1},
		"Email":         {columnType: "TEXT", position: 6},
		"Account Score": {columnType: "numeric", position: 16},
		"CreatedAt":     {columnType: "DATETIME", position: 17},
	}, map[string]struct{}{
		"Work Email":    {},
		"Customer Tier": {},
		"CustomText":    {},
	})
}

func TestSQLiteAccountsIndexedSQLFallbackOmitsCustomFields(t *testing.T) {
	db, err := NewDB(&DBConfig{Type: "sqlite3", Path: filepath.Join(t.TempDir(), "indexed-fallback.db")})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	if _, err := db.GetDB().Exec(db.GetSQL("CreateAccountsTable")); err != nil {
		t.Fatalf("failed to create Accounts: %v", err)
	}
	if _, err := db.GetDB().Exec(db.GetSQL("CreateAccountsIndexedView")); err != nil {
		t.Fatalf("failed to create AccountsIndexed from SQL fallback: %v", err)
	}

	var customCount int
	if err := db.GetDB().QueryRow(`
		SELECT COUNT(*)
		FROM pragma_table_info('AccountsIndexed')
		WHERE lower(name) LIKE 'customtext%' OR lower(name) LIKE 'customnumeric%'
	`).Scan(&customCount); err != nil {
		t.Fatalf("failed to inspect AccountsIndexed fallback: %v", err)
	}
	if customCount != 0 {
		t.Fatalf("expected SQL fallback to omit custom fields, found %d", customCount)
	}
}

func TestValidateSchemaRequiresAccountsIndexed(t *testing.T) {
	tempDir := t.TempDir()
	db, err := NewDB(&DBConfig{Type: "sqlite3", Path: filepath.Join(tempDir, "validate-indexed.db")})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	quietState := &state.State{Quiet: true}
	if err := db.EnforceSchema(quietState); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}
	if _, err := db.GetDB().Exec("DROP VIEW AccountsIndexed"); err != nil {
		t.Fatalf("failed to remove AccountsIndexed: %v", err)
	}

	err = db.ValidateSchema(quietState)
	if err == nil {
		t.Fatal("expected validation to reject a missing AccountsIndexed view")
	}
	if !strings.Contains(err.Error(), "AccountsIndexed") {
		t.Fatalf("expected AccountsIndexed validation error, got %v", err)
	}
}

func TestEnsureJobLogSetupWithoutSyncHistoryTable(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "joblog_only.db")

	config := &DBConfig{
		Type: "sqlite3",
		Path: dbPath,
	}

	db, err := NewDB(config)
	if err != nil {
		t.Fatalf("Failed to load database settings: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := EnsureJobLogSetup(db); err != nil {
		t.Fatalf("EnsureJobLogSetup failed without SyncHistory table: %v", err)
	}

	jobLogExists, err := db.TableExists("JobLog")
	if err != nil {
		t.Fatalf("failed to check JobLog existence: %v", err)
	}
	if !jobLogExists {
		t.Fatalf("expected JobLog table to exist after setup")
	}

	syncHistoryExists, err := db.TableExists("SyncHistory")
	if err != nil {
		t.Fatalf("failed to check SyncHistory existence: %v", err)
	}
	if syncHistoryExists {
		t.Fatalf("expected SyncHistory table to remain absent")
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
