package database

import (
	"badgermaps/app/state"
	"badgermaps/utils"
	"bufio"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	_ "github.com/lib/pq"               // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"     // SQLite driver
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
)

type DBConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
	Path     string `yaml:"path"`
}

//go:embed mssql/*.sql
var mssqlFS embed.FS

//go:embed postgres/*.sql
var postgresFS embed.FS

//go:embed sqlite3/*.sql
var sqlite3FS embed.FS

type columnMigration struct {
	Table   string
	Column  string
	Command string
}

type schemaMigration struct {
	Version int
	Name    string
	Apply   func(DB, *sql.Tx) error
}

type mssqlColumnDefinition struct {
	Name       string
	Definition string
}

var legacyColumnMigrations = []columnMigration{
	{
		Table:   "AccountCheckins",
		Column:  "EndpointType",
		Command: "AddAccountCheckinsEndpointTypeColumn",
	},
	{
		Table:   "AccountCheckinsPendingChanges",
		Column:  "AccountId",
		Command: "AddAccountCheckinsPendingChangesAccountIdColumn",
	},
	{
		Table:   "AccountCheckinsPendingChanges",
		Column:  "EndpointType",
		Command: "AddAccountCheckinsPendingChangesEndpointTypeColumn",
	},
}

var syncHistoryTimezoneColumnMigrations = []columnMigration{
	{
		Table:   "SyncHistory",
		Column:  "StartedAtTimezone",
		Command: "AddSyncHistoryStartedAtTimezoneColumn",
	},
	{
		Table:   "SyncHistory",
		Column:  "CompletedAtTimezone",
		Command: "AddSyncHistoryCompletedAtTimezoneColumn",
	},
}

func applyColumnMigrations(db DB, migrations []columnMigration, s *state.State) error {
	if db.GetDB() == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	return applyColumnMigrationsWithExecutor(db, db.GetDB(), migrations, s)
}

func applyColumnMigrationsWithExecutor(db DB, executor sqlSchemaExecutor, migrations []columnMigration, s *state.State) error {

	checkSQL := db.GetSQL("CheckColumnExists")
	if checkSQL == "" {
		return fmt.Errorf("failed to load SQL command 'CheckColumnExists' for database type '%s'", db.GetType())
	}

	for _, migration := range migrations {
		tableExists, err := tableExistsWithExecutor(db, executor, migration.Table)
		if err != nil {
			return fmt.Errorf("failed to inspect table '%s' for migration '%s': %w", migration.Table, migration.Command, err)
		}
		if !tableExists {
			continue
		}

		var count int
		if err := executor.QueryRow(checkSQL, migration.Table, migration.Column).Scan(&count); err != nil {
			return fmt.Errorf("failed to inspect column '%s' on table '%s': %w", migration.Column, migration.Table, err)
		}
		if count > 0 {
			continue
		}

		alterSQL := db.GetSQL(migration.Command)
		if alterSQL == "" {
			return fmt.Errorf("failed to load SQL command '%s' for database type '%s'", migration.Command, db.GetType())
		}

		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Printf("Applying schema migration: %s.%s... ", migration.Table, migration.Column)
		}
		if _, err := executor.Exec(alterSQL); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to apply schema migration for column '%s' on table '%s': %w", migration.Column, migration.Table, err)
		}
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}

	return nil
}

func addMSSQLMissingColumnsRecursively(db *MSSQLConfig, s *state.State) error {
	if db.GetDB() == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	return addMSSQLMissingColumnsRecursivelyWithExecutor(db, db.GetDB(), s)
}

func addMSSQLMissingColumnsRecursivelyWithExecutor(db *MSSQLConfig, executor sqlSchemaExecutor, s *state.State) error {

	definitionsByTable := make(map[string][]mssqlColumnDefinition, len(RequiredTables()))
	for _, tableName := range RequiredTables() {
		createCmd := CreateCommandForTable(tableName)
		createSQL := db.GetSQL(createCmd)
		if createSQL == "" {
			return fmt.Errorf("failed to load SQL command '%s' for database type '%s'", createCmd, db.GetType())
		}

		definitions, err := extractMSSQLCreateTableColumnDefinitions(createSQL)
		if err != nil {
			return fmt.Errorf("failed to parse column definitions for table '%s': %w", tableName, err)
		}
		definitionsByTable[tableName] = definitions
	}

	const maxPasses = 8
	for pass := 1; pass <= maxPasses; pass++ {
		changed := false

		for _, tableName := range RequiredTables() {
			definitions := definitionsByTable[tableName]
			if len(definitions) == 0 {
				continue
			}

			columns, err := getTableColumnsWithExecutor(db, executor, tableName)
			if err != nil {
				return fmt.Errorf("failed to get columns for table '%s': %w", tableName, err)
			}

			existingColumns := make(map[string]struct{}, len(columns))
			for _, column := range columns {
				existingColumns[normalizeSQLIdentifier(column)] = struct{}{}
			}

			for _, definition := range definitions {
				normalizedColumn := normalizeSQLIdentifier(definition.Name)
				if _, exists := existingColumns[normalizedColumn]; exists {
					continue
				}

				if (s.Verbose || s.Debug) && !s.Quiet {
					fmt.Printf("Adding missing column: %s.%s... ", tableName, definition.Name)
				}
				if err := addMSSQLColumnFromDefinition(executor, tableName, definition); err != nil {
					if (s.Verbose || s.Debug) && !s.Quiet {
						fmt.Println(color.RedString("ERROR"))
					}
					return fmt.Errorf("failed to add missing column '%s' to table '%s': %w", definition.Name, tableName, err)
				}
				if (s.Verbose || s.Debug) && !s.Quiet {
					fmt.Println(color.GreenString("OK"))
				}

				existingColumns[normalizedColumn] = struct{}{}
				changed = true
			}
		}

		if !changed {
			return nil
		}
	}

	return fmt.Errorf("schema migration exceeded maximum number of passes while adding missing columns")
}

func addMSSQLColumnFromDefinition(executor sqlSchemaExecutor, tableName string, definition mssqlColumnDefinition) error {
	quotedTable := fmt.Sprintf("[%s]", escapeMSSQLIdentifier(tableName))
	quotedColumn := fmt.Sprintf("[%s]", escapeMSSQLIdentifier(definition.Name))
	fullDefinition := strings.TrimSpace(definition.Definition)
	primarySQL := fmt.Sprintf("ALTER TABLE %s ADD %s %s;", quotedTable, quotedColumn, fullDefinition)

	if _, err := executor.Exec(primarySQL); err == nil {
		return nil
	} else {
		fallbackDefinition := buildMSSQLFallbackColumnDefinition(fullDefinition)
		if fallbackDefinition == "" || strings.EqualFold(fallbackDefinition, fullDefinition) {
			return err
		}

		fallbackSQL := fmt.Sprintf("ALTER TABLE %s ADD %s %s;", quotedTable, quotedColumn, fallbackDefinition)
		if _, fallbackErr := executor.Exec(fallbackSQL); fallbackErr != nil {
			return fmt.Errorf("%w (fallback failed: %v)", err, fallbackErr)
		}
	}

	return nil
}

func buildMSSQLFallbackColumnDefinition(definition string) string {
	fallback := strings.TrimSpace(definition)
	fallback = mssqlPrimaryKeyRegex.ReplaceAllString(fallback, "")
	fallback = mssqlUniqueRegex.ReplaceAllString(fallback, "")
	fallback = mssqlIdentityRegex.ReplaceAllString(fallback, "")
	if mssqlNotNullRegex.MatchString(fallback) {
		fallback = mssqlNotNullRegex.ReplaceAllString(fallback, "NULL")
	} else if !mssqlNullRegex.MatchString(fallback) {
		fallback += " NULL"
	}
	return strings.Join(strings.Fields(fallback), " ")
}

func extractMSSQLCreateTableColumnDefinitions(createSQL string) ([]mssqlColumnDefinition, error) {
	columnsBlock, err := extractCreateTableColumnsBlock(createSQL)
	if err != nil {
		return nil, err
	}

	clauses := splitTopLevelSQLClauses(columnsBlock)
	definitions := make([]mssqlColumnDefinition, 0, len(clauses))
	for _, clause := range clauses {
		clause = strings.TrimSpace(clause)
		if clause == "" || isMSSQLTableConstraintClause(clause) {
			continue
		}

		columnName, columnDefinition, ok := splitMSSQLColumnDefinition(clause)
		if !ok || strings.TrimSpace(columnDefinition) == "" {
			continue
		}
		definitions = append(definitions, mssqlColumnDefinition{
			Name:       columnName,
			Definition: strings.TrimSpace(columnDefinition),
		})
	}

	return definitions, nil
}

func extractCreateTableColumnsBlock(createSQL string) (string, error) {
	upperSQL := strings.ToUpper(createSQL)
	createTableIndex := strings.Index(upperSQL, "CREATE TABLE")
	if createTableIndex == -1 {
		return "", fmt.Errorf("CREATE TABLE statement not found")
	}

	openParenOffset := strings.Index(createSQL[createTableIndex:], "(")
	if openParenOffset == -1 {
		return "", fmt.Errorf("opening parenthesis for CREATE TABLE not found")
	}
	openParenIndex := createTableIndex + openParenOffset

	depth := 0
	inString := false
	for i := openParenIndex; i < len(createSQL); i++ {
		ch := createSQL[i]
		if ch == '\'' {
			if inString && i+1 < len(createSQL) && createSQL[i+1] == '\'' {
				i++
				continue
			}
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch ch {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return createSQL[openParenIndex+1 : i], nil
			}
		}
	}

	return "", fmt.Errorf("unclosed CREATE TABLE column definition list")
}

func splitTopLevelSQLClauses(input string) []string {
	clauses := make([]string, 0, 8)
	start := 0
	depth := 0
	inString := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		if ch == '\'' {
			if inString && i+1 < len(input) && input[i+1] == '\'' {
				i++
				continue
			}
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch ch {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				clauses = append(clauses, input[start:i])
				start = i + 1
			}
		}
	}

	if start < len(input) {
		clauses = append(clauses, input[start:])
	}

	return clauses
}

func isMSSQLTableConstraintClause(clause string) bool {
	upperClause := strings.ToUpper(strings.TrimSpace(clause))
	return strings.HasPrefix(upperClause, "PRIMARY KEY") ||
		strings.HasPrefix(upperClause, "FOREIGN KEY") ||
		strings.HasPrefix(upperClause, "UNIQUE") ||
		strings.HasPrefix(upperClause, "CONSTRAINT") ||
		strings.HasPrefix(upperClause, "CHECK")
}

func splitMSSQLColumnDefinition(clause string) (string, string, bool) {
	clause = strings.TrimSpace(clause)
	if clause == "" {
		return "", "", false
	}

	if strings.HasPrefix(clause, "[") {
		endIndex := strings.Index(clause, "]")
		if endIndex == -1 {
			return "", "", false
		}

		columnName := clause[1:endIndex]
		columnDefinition := strings.TrimSpace(clause[endIndex+1:])
		return columnName, columnDefinition, columnDefinition != ""
	}

	for i, r := range clause {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			columnName := clause[:i]
			columnDefinition := strings.TrimSpace(clause[i+1:])
			return columnName, columnDefinition, columnDefinition != ""
		}
	}

	return "", "", false
}

func normalizeSQLIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	identifier = strings.Trim(identifier, "[]`\"")
	return strings.ToLower(identifier)
}

func escapeMSSQLIdentifier(identifier string) string {
	return strings.ReplaceAll(identifier, "]", "]]")
}

type DB interface {
	GetType() string
	DatabaseConnection() string
	LoadConfig(config *DBConfig) error
	GetUsername() string
	SaveConfig(config *DBConfig) error
	PromptDatabaseSettings()
	TableExists(tableName string) (bool, error)
	ViewExists(viewName string) (bool, error)
	ProcedureExists(procedureName string) (bool, error)
	TriggerExists(triggerName string) (bool, error)
	GetTableColumns(tableName string) ([]string, error)
	ValidateSchema(s *state.State) error
	EnforceSchema(s *state.State) error
	ResetSchema(s *state.State) error
	TestConnection() error
	DropAllTables() error
	Connect() error
	Close() error
	GetDB() *sql.DB
	GetSQL(command string) string
	RunAction(action ActionConfig) error
	GetTables() ([]string, error)
	ExecuteQuery(query string) (*sql.Rows, error)
	IsConnected() bool
	SetConnected(connected bool)
}

type sqlQueryExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
}

type sqlSchemaExecutor interface {
	sqlQueryExecer
	QueryRow(query string, args ...any) *sql.Row
}

func tableExistsWithExecutor(db DB, executor sqlSchemaExecutor, tableName string) (bool, error) {
	query := db.GetSQL("CheckTableExists")
	if query == "" {
		return false, fmt.Errorf("failed to load SQL command 'CheckTableExists' for database type '%s'", db.GetType())
	}
	var count int
	if err := executor.QueryRow(query, tableName).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func getTableColumnsWithExecutor(db DB, executor sqlSchemaExecutor, tableName string) ([]string, error) {
	query := db.GetSQL("GetTableColumns")
	if query == "" {
		return nil, fmt.Errorf("failed to load SQL command 'GetTableColumns' for database type '%s'", db.GetType())
	}

	var (
		rows *sql.Rows
		err  error
	)
	if db.GetType() == "sqlite3" {
		query = fmt.Sprintf(query, strings.ReplaceAll(tableName, "'", "''"))
		rows, err = executor.Query(query)
	} else {
		rows, err = executor.Query(query, tableName)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		if db.GetType() == "sqlite3" {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				return nil, err
			}
			columns = append(columns, name)
			continue
		}

		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columns, nil
}

const (
	accountsWithLabelsColumnsMarker = "/* ACCOUNTS_WITH_LABELS_COLUMNS */ *"
	accountsIndexedFallbackColumns  = "a.AccountId, a.FirstName, a.LastName, a.FullName, a.PhoneNumber, a.Email, a.CustomerId, a.Notes, a.OriginalAddress, a.CrmId, a.AccountOwner, a.DaysSinceLastCheckin, a.LastCheckinDate, a.LastModifiedDate, a.FollowUpDate, a.CreatedAt, a.UpdatedAt"
	accountsIndexedColumnsMarker    = "/* ACCOUNTS_INDEXED_COLUMNS */ " + accountsIndexedFallbackColumns
)

// RefreshAccountsWithLabels rebuilds the account view after profile metadata changes.
func RefreshAccountsWithLabels(db DB, executor sqlQueryExecer) error {
	switch db.GetType() {
	case "sqlite3":
		return refreshSQLiteAccountView(
			db,
			executor,
			"AccountsWithLabels",
			"GetAccountsWithLabelsColumns",
			"CreateAccountsWithLabelsView",
			accountsWithLabelsColumnsMarker,
		)
	case "postgres":
		_, err := executor.Exec("SELECT AccountsWithLabelsView()")
		return err
	case "mssql":
		_, err := executor.Exec("EXEC AccountsWithLabelsView")
		return err
	default:
		return fmt.Errorf("unsupported database type %q", db.GetType())
	}
}

// RefreshAccountsIndexed rebuilds the profile-indexed account view.
func RefreshAccountsIndexed(db DB, executor sqlQueryExecer) error {
	switch db.GetType() {
	case "sqlite3":
		return refreshSQLiteAccountView(
			db,
			executor,
			"AccountsIndexed",
			"GetAccountsIndexedColumns",
			"CreateAccountsIndexedView",
			accountsIndexedColumnsMarker,
		)
	case "postgres":
		_, err := executor.Exec("SELECT AccountsIndexedView()")
		return err
	case "mssql":
		_, err := executor.Exec("EXEC AccountsIndexedView")
		return err
	default:
		return fmt.Errorf("unsupported database type %q", db.GetType())
	}
}

func createAccountsIndexedColumnsView(db DB, executor sqlQueryExecer) error {
	sqlText := db.GetSQL("CreateAccountsIndexedColumnsView")
	if sqlText == "" {
		return fmt.Errorf("failed to load SQL command 'CreateAccountsIndexedColumnsView' for database type '%s'", db.GetType())
	}
	_, err := executor.Exec(sqlText)
	return err
}

// RefreshGeneratedViews performs one refresh pass after profile metadata is stored.
func RefreshGeneratedViews(db DB, executor sqlQueryExecer) error {
	if err := RefreshAccountsWithLabels(db, executor); err != nil {
		return fmt.Errorf("failed to refresh AccountsWithLabels: %w", err)
	}
	if err := RefreshAccountsIndexed(db, executor); err != nil {
		return fmt.Errorf("failed to refresh AccountsIndexed: %w", err)
	}
	return nil
}

func refreshSQLiteAccountView(db DB, executor sqlQueryExecer, viewName, columnsCommand, createCommand, columnsMarker string) error {
	if db.GetType() != "sqlite3" {
		return nil
	}

	columnsSQL := db.GetSQL(columnsCommand)
	if columnsSQL == "" {
		return fmt.Errorf("failed to load SQL command '%s' for database type '%s'", columnsCommand, db.GetType())
	}
	rows, err := executor.Query(columnsSQL)
	if err != nil {
		return fmt.Errorf("failed to load %s columns: %w", viewName, err)
	}
	defer rows.Close()

	selectColumns := make([]string, 0, 80)
	aliases := make(map[string]string)
	sources := make(map[string]struct{})
	for rows.Next() {
		var column string
		var label sql.NullString
		if err := rows.Scan(&column, &label); err != nil {
			return fmt.Errorf("failed to scan %s column: %w", viewName, err)
		}

		normalizedSource := strings.ToLower(column)
		if _, exists := sources[normalizedSource]; exists {
			return fmt.Errorf("multiple data sets map to account field %q", column)
		}
		sources[normalizedSource] = struct{}{}

		alias := column
		if label.Valid && label.String != "" {
			alias = label.String
		}
		normalizedAlias := strings.ToLower(alias)
		if existingColumn, exists := aliases[normalizedAlias]; exists {
			return fmt.Errorf("duplicate account view label %q for fields %q and %q", alias, existingColumn, column)
		}
		aliases[normalizedAlias] = column

		expression := "a." + quoteSQLiteIdentifier(column)
		if alias != column {
			expression += " AS " + quoteSQLiteIdentifier(alias)
		}
		selectColumns = append(selectColumns, expression)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read %s columns: %w", viewName, err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("failed to close %s columns: %w", viewName, err)
	}
	if len(selectColumns) == 0 {
		return fmt.Errorf("cannot create %s because Accounts has no columns", viewName)
	}

	viewSQL := db.GetSQL(createCommand)
	if strings.Count(viewSQL, columnsMarker) != 1 {
		return fmt.Errorf("%s SQL is missing its column marker", createCommand)
	}
	viewSQL = strings.Replace(viewSQL, columnsMarker, strings.Join(selectColumns, ", "), 1)
	if _, err := executor.Exec(viewSQL); err != nil {
		return fmt.Errorf("failed to rebuild %s view: %w", viewName, err)
	}
	return nil
}

func quoteSQLiteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// SQLiteConfig represents a SQLite database configuration
type SQLiteConfig struct {
	db        *sql.DB
	Path      string `mapstructure:"DB_PATH"`
	connected bool
}

func (db *SQLiteConfig) IsConnected() bool {
	return db.connected
}

func (db *SQLiteConfig) SetConnected(connected bool) {
	db.connected = connected
}

func (db *SQLiteConfig) GetSQL(command string) string {
	path := fmt.Sprintf("sqlite3/%s.sql", command)
	data, err := sqlite3FS.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func (db *SQLiteConfig) Connect() error {
	// Ensure the parent directory exists before attempting to create the database file
	dir := filepath.Dir(db.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		db.connected = false
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	var err error
	db.db, err = sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		db.connected = false
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	return nil
}

func (db *SQLiteConfig) Close() error {
	db.connected = false
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}

func (db *SQLiteConfig) GetDB() *sql.DB {
	return db.db
}

func (db *SQLiteConfig) GetTableColumns(tableName string) ([]string, error) {
	sqlDB := db.GetDB()
	queryTemplate := db.GetSQL("GetTableColumns")
	escapedTableName := strings.ReplaceAll(tableName, "'", "''")
	query := fmt.Sprintf(queryTemplate, escapedTableName)

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name string
		var type_ string
		var notnull int
		var dflt_value any
		var pk int
		if err := rows.Scan(&cid, &name, &type_, &notnull, &dflt_value, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, nil
}

func (db *SQLiteConfig) EnforceSchema(s *state.State) error {
	sqlDB := db.GetDB()
	if sqlDB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	return db.enforceSchema(s, sqlDB)
}

func (db *SQLiteConfig) enforceSchema(s *state.State, executor sqlSchemaExecutor) error {
	for _, tableName := range RequiredTables() {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Printf("Creating table: %s... ", tableName)
		}
		createCmd := CreateCommandForTable(tableName)
		sqlText := db.GetSQL(createCmd)
		if sqlText == "" {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to load SQL command '%s' for database type '%s'", createCmd, db.GetType())
		}
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}

	if err := applyColumnMigrationsWithExecutor(db, executor, syncHistoryTimezoneColumnMigrations, s); err != nil {
		return err
	}

	// Insert initial data for FieldMaps
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Inserting initial data for FieldMaps... ")
	}
	sqlText := db.GetSQL("InsertFieldMaps")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for FieldMaps: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Insert initial data for Configurations
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Inserting initial data for Configurations... ")
	}
	sqlText = db.GetSQL("InsertConfigurations")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for Configurations: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create view
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating view: AccountsWithLabels... ")
	}
	if err := RefreshAccountsWithLabels(db, executor); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to create view AccountsWithLabels: %w", err)
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating view: AccountsIndexed... ")
	}
	if err := RefreshAccountsIndexed(db, executor); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to create view AccountsIndexed: %w", err)
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating view: AccountsIndexedColumns... ")
	}
	if err := createAccountsIndexedColumnsView(db, executor); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to create view AccountsIndexedColumns: %w", err)
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}

func (db *SQLiteConfig) TestConnection() error {
	sqlDB := db.GetDB()
	if sqlDB == nil {
		db.connected = false
		return fmt.Errorf("database connection is not initialized")
	}
	err := sqlDB.Ping()
	if err != nil {
		db.connected = false
		return err
	}
	db.connected = true
	return nil
}

func (db *SQLiteConfig) ValidateSchema(s *state.State) error {
	if db.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	expectedSchema := GetExpectedSchema()
	for _, tableName := range RequiredTables() {
		if s.Verbose && !s.Quiet {
			fmt.Printf("Checking table: %s... ", tableName)
		}
		exists, err := db.TableExists(tableName)
		if err != nil {
			if s.Verbose && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			if s.Verbose && !s.Quiet {
				fmt.Println(color.RedString("MISSING"))
			}
			return fmt.Errorf("required table %s does not exist", tableName)
		}
		if s.Verbose && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}

		columns, err := db.GetTableColumns(tableName)
		if err != nil {
			return fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}

		expectedColumns := expectedSchema[tableName]
		for _, expectedColumn := range expectedColumns {
			found := false
			for _, column := range columns {
				if column == expectedColumn {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("missing column '%s' in table '%s'", expectedColumn, tableName)
			}
		}
	}

	if err := validateRequiredViews(db, s); err != nil {
		return err
	}

	return nil
}

func (db *SQLiteConfig) TableExists(tableName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("CheckTableExists")
	var count int
	err := sqlDB.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *SQLiteConfig) ViewExists(viewName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("CheckViewExists")
	var count int
	err := sqlDB.QueryRow(query, viewName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *SQLiteConfig) ProcedureExists(procedureName string) (bool, error) {
	return true, nil
}

func (db *SQLiteConfig) TriggerExists(triggerName string) (bool, error) {
	return true, nil
}

func (db *SQLiteConfig) GetType() string {
	return "sqlite3"
}

func (db *SQLiteConfig) LoadConfig(config *DBConfig) error {
	db.Path = config.Path
	return nil
}

func (db *SQLiteConfig) SaveConfig(config *DBConfig) error {
	config.Path = db.Path
	return nil
}

func (db *SQLiteConfig) GetUsername() string {
	return ""
}

func (db *SQLiteConfig) DatabaseConnection() string {
	return fmt.Sprintf("file:%s?mode=rwc", db.Path)
}

func (db *SQLiteConfig) PromptDatabaseSettings() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(utils.Colors.Cyan("SQLite Database Configuration"))
	db.Path = utils.PromptString(reader, "Database Path", db.Path)
}

func (db *SQLiteConfig) DropAllTables() error {
	sqlDB := db.GetDB()
	if sqlDB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	for _, viewName := range requiredViews() {
		query := fmt.Sprintf("DROP VIEW IF EXISTS %s", viewName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop view %s: %w", viewName, err)
		}
	}

	for _, tableName := range dropTableOrder() {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}
	return nil
}

func (db *SQLiteConfig) ResetSchema(s *state.State) error {
	if s == nil {
		s = &state.State{}
	}
	if !s.Quiet {
		fmt.Println(color.YellowString("Warning: Re-initializing the database schema will delete all existing data."))
	}

	if err := db.DropAllTables(); err != nil {
		return err
	}

	return db.EnforceSchema(s)
}

func (db *SQLiteConfig) RunAction(action ActionConfig) error {
	var query string
	var args []interface{}

	if cmd, ok := action.Args["command"].(string); ok {
		query = db.GetSQL(cmd)
	} else if q, ok := action.Args["query"].(string); ok {
		query = q
	} else {
		return fmt.Errorf("sqlite action requires 'command' or 'query'")
	}

	if query == "" {
		return fmt.Errorf("SQL command not found or query is empty")
	}

	if params, ok := action.Args["args"].([]interface{}); ok {
		args = params
	}

	_, err := db.db.Exec(query, args...)
	return err
}

func (db *SQLiteConfig) GetTables() ([]string, error) {
	rows, err := db.db.Query("SELECT name FROM sqlite_master WHERE type IN ('table','view') ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func (db *SQLiteConfig) ExecuteQuery(query string) (*sql.Rows, error) {
	return db.db.Query(query)
}

// PostgreSQLConfig represents a PostgreSQL database configuration
type PostgreSQLConfig struct {
	db        *sql.DB
	Host      string `mapstructure:"DB_HOST"`
	Port      int    `mapstructure:"DB_PORT"`
	Database  string `mapstructure:"DB_NAME"`
	Username  string `mapstructure:"DB_USER"`
	Password  string `mapstructure:"DB_PASSWORD"`
	SSLMode   string `mapstructure:"DB_SSL_MODE"`
	connected bool
}

func (db *PostgreSQLConfig) IsConnected() bool {
	return db.connected
}

func (db *PostgreSQLConfig) SetConnected(connected bool) {
	db.connected = connected
}

func (db *PostgreSQLConfig) GetSQL(command string) string {
	path := fmt.Sprintf("postgres/%s.sql", command)
	data, err := postgresFS.ReadFile(path)
	if err != nil {
		return ""
	}
	return rebindPostgreSQLPlaceholders(string(data))
}

func rebindPostgreSQLPlaceholders(query string) string {
	if !strings.Contains(query, "?") {
		return query
	}

	var rebound strings.Builder
	parameter := 1
	for _, character := range query {
		if character == '?' {
			fmt.Fprintf(&rebound, "$%d", parameter)
			parameter++
			continue
		}
		rebound.WriteRune(character)
	}
	return rebound.String()
}

func (db *PostgreSQLConfig) Connect() error {
	var err error
	db.db, err = sql.Open("postgres", db.DatabaseConnection())
	if err != nil {
		db.connected = false
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	return nil
}

func (db *PostgreSQLConfig) Close() error {
	db.connected = false
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}

func (db *PostgreSQLConfig) GetDB() *sql.DB {
	return db.db
}

func (db *PostgreSQLConfig) GetTableColumns(tableName string) ([]string, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("GetTableColumns")

	rows, err := sqlDB.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, nil
}

func (db *PostgreSQLConfig) EnforceSchema(s *state.State) error {
	sqlDB := db.GetDB()
	if sqlDB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	return db.enforceSchema(s, sqlDB)
}

func (db *PostgreSQLConfig) enforceSchema(s *state.State, executor sqlSchemaExecutor) error {
	for _, tableName := range RequiredTables() {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Printf("Creating table: %s... ", tableName)
		}
		createCmd := CreateCommandForTable(tableName)
		sqlText := db.GetSQL(createCmd)
		if sqlText == "" {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to load SQL command '%s' for database type '%s'", createCmd, db.GetType())
		}
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}

	if err := applyColumnMigrationsWithExecutor(db, executor, syncHistoryTimezoneColumnMigrations, s); err != nil {
		return err
	}

	// Insert initial data for FieldMaps
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Inserting initial data for FieldMaps... ")
	}
	sqlText := db.GetSQL("InsertFieldMaps")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for FieldMaps: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Insert initial data for Configurations
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Inserting initial data for Configurations... ")
	}
	sqlText = db.GetSQL("InsertConfigurations")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for Configurations: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create function
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating function: AccountsWithLabelsView... ")
	}
	sqlText = db.GetSQL("CreateAccountsWithLabelsView")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create function AccountsWithLabelsView: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Call function to create view
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating view: AccountsWithLabels... ")
	}
	if _, err := executor.Exec("SELECT AccountsWithLabelsView()"); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to execute AccountsWithLabelsView function: %w", err)
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create the profile-indexed account view function.
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating function: AccountsIndexedView... ")
	}
	sqlText = db.GetSQL("CreateAccountsIndexedView")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create function AccountsIndexedView: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating view: AccountsIndexed... ")
	}
	if _, err := executor.Exec("SELECT AccountsIndexedView()"); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to execute AccountsIndexedView function: %w", err)
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating view: AccountsIndexedColumns... ")
	}
	if err := createAccountsIndexedColumnsView(db, executor); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to create view AccountsIndexedColumns: %w", err)
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create trigger
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating trigger: datasets_update_trigger... ")
	}
	sqlText = db.GetSQL("CreateDatasetsUpdateTrigger")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create trigger datasets_update_trigger: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create function to update field mappings
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating function: UpdateFieldMapsFromDatasets... ")
	}
	sqlText = db.GetSQL("UpdateFieldMapsFromDatasets")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create function UpdateFieldMapsFromDatasets: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create trigger to update field mappings
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating trigger: DatasetsFieldMapsUpdateTrigger... ")
	}
	sqlText = db.GetSQL("CreateFieldMapsUpdateTrigger")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create trigger DatasetsFieldMapsUpdateTrigger: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}
func (db *PostgreSQLConfig) TestConnection() error {
	sqlDB := db.GetDB()
	if sqlDB == nil {
		db.connected = false
		return fmt.Errorf("database connection is not initialized")
	}
	err := sqlDB.Ping()
	if err != nil {
		db.connected = false
		return err
	}
	db.connected = true
	return nil
}
func (db *PostgreSQLConfig) ValidateSchema(s *state.State) error {
	if db.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	expectedSchema := GetExpectedSchema()
	for _, tableName := range RequiredTables() {
		if s.Verbose && !s.Quiet {
			fmt.Printf("Checking table: %s... ", tableName)
		}
		exists, err := db.TableExists(tableName)
		if err != nil {
			if s.Verbose && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			if s.Verbose && !s.Quiet {
				fmt.Println(color.RedString("MISSING"))
			}
			return fmt.Errorf("required table %s does not exist", tableName)
		}
		if s.Verbose && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}

		columns, err := db.GetTableColumns(tableName)
		if err != nil {
			return fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}

		expectedColumns := expectedSchema[tableName]
		for _, expectedColumn := range expectedColumns {
			found := false
			for _, column := range columns {
				if strings.EqualFold(column, expectedColumn) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("missing column '%s' in table '%s'", expectedColumn, tableName)
			}
		}
	}

	if err := validateRequiredViews(db, s); err != nil {
		return err
	}

	if s.Verbose && !s.Quiet {
		fmt.Printf("Checking function: UpdateFieldMapsFromDatasets... ")
	}
	procExists, err := db.ProcedureExists("UpdateFieldMapsFromDatasets")
	if err != nil {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("error checking if function UpdateFieldMapsFromDatasets exists: %w", err)
	}
	if !procExists {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required function UpdateFieldMapsFromDatasets does not exist")
	}
	if s.Verbose && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	if s.Verbose && !s.Quiet {
		fmt.Printf("Checking trigger: DatasetsUpdateTrigger... ")
	}
	triggerExists, err := db.TriggerExists("DatasetsUpdateTrigger")
	if err != nil {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("error checking if trigger DatasetsUpdateTrigger exists: %w", err)
	}
	if !triggerExists {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required trigger DatasetsUpdateTrigger does not exist")
	}
	if s.Verbose && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}
func (db *PostgreSQLConfig) TableExists(tableName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("CheckTableExists")
	var count int
	err := sqlDB.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *PostgreSQLConfig) ViewExists(viewName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("CheckViewExists")
	var count int
	err := sqlDB.QueryRow(query, viewName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *PostgreSQLConfig) ProcedureExists(procedureName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("CheckProcedureExists")
	var count int
	err := sqlDB.QueryRow(query, procedureName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *PostgreSQLConfig) TriggerExists(triggerName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("CheckTriggerExists")
	var count int
	err := sqlDB.QueryRow(query, triggerName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
func (db *PostgreSQLConfig) GetType() string {
	return "postgres"
}

func (db *PostgreSQLConfig) LoadConfig(config *DBConfig) error {
	db.Host = config.Host
	db.Port = config.Port
	db.Database = config.Database
	db.Username = config.Username
	db.Password = config.Password
	db.SSLMode = config.SSLMode
	return nil
}

func (db *PostgreSQLConfig) SaveConfig(config *DBConfig) error {
	config.Host = db.Host
	config.Port = db.Port
	config.Database = db.Database
	config.Username = db.Username
	config.Password = db.Password
	config.SSLMode = db.SSLMode
	return nil
}

func (db *PostgreSQLConfig) GetUsername() string {
	return db.Username
}
func (db *PostgreSQLConfig) DatabaseConnection() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(db.Username, db.Password),
		Host:   fmt.Sprintf("%s:%d", db.Host, db.Port),
		Path:   db.Database,
	}
	q := u.Query()
	q.Set("sslmode", db.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}
func (db *PostgreSQLConfig) PromptDatabaseSettings() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(utils.Colors.Cyan("PostgreSQL Database Configuration"))
	db.Host = utils.PromptString(reader, "Database Host", db.Host)
	db.Port = utils.PromptInt(reader, "Database Port", db.Port)
	db.Database = utils.PromptString(reader, "Database Name", db.Database)
	db.Username = utils.PromptString(reader, "Database Username", db.Username)
	db.Password = utils.PromptPassword(reader, "Database Password", db.Password)
	db.SSLMode = utils.PromptString(reader, "Database SSL Mode", db.SSLMode)
}

func (db *PostgreSQLConfig) DropAllTables() error {
	sqlDB := db.GetDB()
	if sqlDB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	for _, viewName := range requiredViews() {
		query := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE", viewName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop view %s: %w", viewName, err)
		}
	}

	for _, tableName := range dropTableOrder() {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}
	return nil
}

func (db *PostgreSQLConfig) ResetSchema(s *state.State) error {
	if s == nil {
		s = &state.State{}
	}
	if !s.Quiet {
		fmt.Println(color.YellowString("Warning: Re-initializing the database schema will delete all existing data."))
	}

	if err := db.DropAllTables(); err != nil {
		return err
	}

	return db.EnforceSchema(s)
}

func (db *PostgreSQLConfig) RunAction(action ActionConfig) error {
	var query string
	var args []interface{}

	if cmd, ok := action.Args["command"].(string); ok {
		query = db.GetSQL(cmd)
	} else if fn, ok := action.Args["function"].(string); ok {
		// Note: This is a simplified approach. For functions with arguments,
		// a more robust solution would be needed to handle placeholders.
		query = fmt.Sprintf("SELECT %s()", fn)
	} else if proc, ok := action.Args["procedure"].(string); ok {
		query = fmt.Sprintf("CALL %s()", proc)
	} else if q, ok := action.Args["query"].(string); ok {
		query = q
	} else {
		return fmt.Errorf("postgres action requires 'command', 'function', 'procedure', or 'query'")
	}

	if query == "" {
		return fmt.Errorf("SQL command not found or query is empty")
	}

	if params, ok := action.Args["args"].([]interface{}); ok {
		args = params
	}

	_, err := db.db.Exec(query, args...)
	return err
}

func (db *PostgreSQLConfig) GetTables() ([]string, error) {
	query := `
		SELECT tablename AS name
		FROM pg_catalog.pg_tables
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		UNION
		SELECT viewname AS name
		FROM pg_catalog.pg_views
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY name`
	rows, err := db.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func (db *PostgreSQLConfig) ExecuteQuery(query string) (*sql.Rows, error) {
	return db.db.Query(query)
}

// MSSQLConfig represents a Microsoft SQL Server database configuration
type MSSQLConfig struct {
	db        *sql.DB
	Host      string `mapstructure:"DB_HOST"`
	Port      int    `mapstructure:"DB_PORT"`
	Database  string `mapstructure:"DB_NAME"`
	Username  string `mapstructure:"DB_USER"`
	Password  string `mapstructure:"DB_PASSWORD"`
	connected bool
}

func (db *MSSQLConfig) IsConnected() bool {
	return db.connected
}

func (db *MSSQLConfig) SetConnected(connected bool) {
	db.connected = connected
}

func (db *MSSQLConfig) GetSQL(command string) string {
	path := fmt.Sprintf("mssql/%s.sql", command)
	data, err := mssqlFS.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func (db *MSSQLConfig) Connect() error {
	var err error
	db.db, err = sql.Open("mssql", db.DatabaseConnection())
	if err != nil {
		db.connected = false
		return fmt.Errorf("failed to open MSSQL database: %w", err)
	}
	return nil
}

func (db *MSSQLConfig) Close() error {
	db.connected = false
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}

func (db *MSSQLConfig) GetDB() *sql.DB {
	return db.db
}

func (db *MSSQLConfig) GetTableColumns(tableName string) ([]string, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("GetTableColumns")

	rows, err := sqlDB.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, nil
}

func (db *MSSQLConfig) EnforceSchema(s *state.State) error {
	sqlDB := db.GetDB()
	if sqlDB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	return db.enforceSchema(s, sqlDB)
}

func (db *MSSQLConfig) enforceSchema(s *state.State, executor sqlSchemaExecutor) error {
	for _, tableName := range RequiredTables() {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Printf("Creating table: %s... ", tableName)
		}
		createCmd := CreateCommandForTable(tableName)
		sqlText := db.GetSQL(createCmd)
		if sqlText == "" {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to load SQL command '%s' for database type '%s'", createCmd, db.GetType())
		}
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}

	if err := applyColumnMigrationsWithExecutor(db, executor, legacyColumnMigrations, s); err != nil {
		return err
	}
	if err := applyColumnMigrationsWithExecutor(db, executor, syncHistoryTimezoneColumnMigrations, s); err != nil {
		return err
	}
	if err := addMSSQLMissingColumnsRecursivelyWithExecutor(db, executor, s); err != nil {
		return err
	}

	// Insert initial data for FieldMaps
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Inserting initial data for FieldMaps... ")
	}
	sqlText := db.GetSQL("InsertFieldMaps")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for FieldMaps: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Insert initial data for Configurations
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Inserting initial data for Configurations... ")
	}
	sqlText = db.GetSQL("InsertConfigurations")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for Configurations: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create procedure
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating procedure: AccountsWithLabelsView... ")
	}
	sqlText = db.GetSQL("CreateAccountsWithLabelsView")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create procedure AccountsWithLabelsView: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Call procedure to create view
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating view: AccountsWithLabels... ")
	}
	if _, err := executor.Exec("EXEC AccountsWithLabelsView"); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to execute AccountsWithLabelsView procedure: %w", err)
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create the profile-indexed account view procedure.
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating procedure: AccountsIndexedView... ")
	}
	sqlText = db.GetSQL("CreateAccountsIndexedView")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create procedure AccountsIndexedView: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating view: AccountsIndexed... ")
	}
	if _, err := executor.Exec("EXEC AccountsIndexedView"); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to execute AccountsIndexedView procedure: %w", err)
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating view: AccountsIndexedColumns... ")
	}
	if err := createAccountsIndexedColumnsView(db, executor); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to create view AccountsIndexedColumns: %w", err)
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create trigger
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating trigger: datasets_update_trigger... ")
	}
	sqlText = db.GetSQL("CreateDatasetsUpdateTrigger")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create trigger datasets_update_trigger: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create procedure to update field maps
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating procedure: update_field_maps_from_datasets... ")
	}
	sqlText = db.GetSQL("UpdateFieldMapsFromDatasets")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create procedure update_field_maps_from_datasets: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create trigger to update field maps
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Creating trigger: datasets_field_maps_update_trigger... ")
	}
	sqlText = db.GetSQL("CreateFieldMapsUpdateTrigger")
	if sqlText != "" {
		if _, err := executor.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create trigger datasets_field_maps_update_trigger: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}
func (db *MSSQLConfig) TestConnection() error {
	sqlDB := db.GetDB()
	if sqlDB == nil {
		db.connected = false
		return fmt.Errorf("database connection is not initialized")
	}
	err := sqlDB.Ping()
	if err != nil {
		db.connected = false
		return err
	}
	db.connected = true
	return nil
}
func (db *MSSQLConfig) ValidateSchema(s *state.State) error {
	if db.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	expectedSchema := GetExpectedSchema()
	for _, tableName := range RequiredTables() {
		if s.Verbose && !s.Quiet {
			fmt.Printf("Checking table: %s... ", tableName)
		}
		exists, err := db.TableExists(tableName)
		if err != nil {
			if s.Verbose && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			if s.Verbose && !s.Quiet {
				fmt.Println(color.RedString("MISSING"))
			}
			return fmt.Errorf("required table %s does not exist", tableName)
		}
		if s.Verbose && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}

		columns, err := db.GetTableColumns(tableName)
		if err != nil {
			return fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}

		expectedColumns := expectedSchema[tableName]
		for _, expectedColumn := range expectedColumns {
			found := false
			for _, column := range columns {
				if column == expectedColumn {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("missing column '%s' in table '%s'", expectedColumn, tableName)
			}
		}
	}

	if err := validateRequiredViews(db, s); err != nil {
		return err
	}

	if s.Verbose && !s.Quiet {
		fmt.Printf("Checking procedure: UpdateFieldMapsFromDatasets... ")
	}
	procExists, err := db.ProcedureExists("UpdateFieldMapsFromDatasets")
	if err != nil {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("error checking if procedure UpdateFieldMapsFromDatasets exists: %w", err)
	}
	if !procExists {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required procedure UpdateFieldMapsFromDatasets does not exist")
	}
	if s.Verbose && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	if s.Verbose && !s.Quiet {
		fmt.Printf("Checking trigger: DatasetsUpdateTrigger... ")
	}
	triggerExists, err := db.TriggerExists("DatasetsUpdateTrigger")
	if err != nil {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("error checking if trigger DatasetsUpdateTrigger exists: %w", err)
	}
	if !triggerExists {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required trigger DatasetsUpdateTrigger does not exist")
	}
	if s.Verbose && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}
func (db *MSSQLConfig) TableExists(tableName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("CheckTableExists")
	var count int
	err := sqlDB.QueryRow(query, tableName).Scan(&count)

	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *MSSQLConfig) ViewExists(viewName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("CheckViewExists")
	var count int
	err := sqlDB.QueryRow(query, viewName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *MSSQLConfig) ProcedureExists(procedureName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("CheckProcedureExists")
	var count int
	err := sqlDB.QueryRow(query, procedureName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *MSSQLConfig) TriggerExists(triggerName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("CheckTriggerExists")
	var count int
	err := sqlDB.QueryRow(query, triggerName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *MSSQLConfig) GetType() string {
	return "mssql"
}

func (db *MSSQLConfig) LoadConfig(config *DBConfig) error {
	db.Host = config.Host
	db.Port = config.Port
	db.Database = config.Database
	db.Username = config.Username
	db.Password = config.Password
	return nil
}

func (db *MSSQLConfig) SaveConfig(config *DBConfig) error {
	config.Host = db.Host
	config.Port = db.Port
	config.Database = db.Database
	config.Username = db.Username
	config.Password = db.Password
	return nil
}

func (db *MSSQLConfig) GetUsername() string {
	return db.Username
}
func (db *MSSQLConfig) DatabaseConnection() string {
	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(db.Username, db.Password),
		Host:   fmt.Sprintf("%s:%d", db.Host, db.Port),
	}
	q := u.Query()
	q.Set("database", db.Database)
	u.RawQuery = q.Encode()
	return u.String()
}
func (db *MSSQLConfig) PromptDatabaseSettings() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(utils.Colors.Cyan("Microsoft SQL Server Database Configuration"))
	db.Host = utils.PromptString(reader, "Database Host", db.Host)
	db.Port = utils.PromptInt(reader, "Database Port", db.Port)
	db.Database = utils.PromptString(reader, "Database Name", db.Database)
	db.Username = utils.PromptString(reader, "Database Username", db.Username)
	db.Password = utils.PromptPassword(reader, "Database Password", db.Password)
}

func (db *MSSQLConfig) DropAllTables() error {
	sqlDB := db.GetDB()
	if sqlDB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	// First, drop all foreign key constraints
	// This is a bit of a heavy-handed approach, but it's reliable
	// A more elegant solution would be to drop tables in the correct order
	// but that requires parsing the schema, which is complex.

	rows, err := sqlDB.Query(`
		SELECT
			fk.name,
			OBJECT_SCHEMA_NAME(fk.parent_object_id) AS schema_name,
			OBJECT_NAME(fk.parent_object_id) AS table_name
		FROM sys.foreign_keys fk
	`)
	if err != nil {
		return fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, schemaName, tableName string
		if err := rows.Scan(&name, &schemaName, &tableName); err != nil {
			return fmt.Errorf("failed to scan foreign key: %w", err)
		}
		query := fmt.Sprintf(
			"ALTER TABLE [%s].[%s] DROP CONSTRAINT [%s]",
			strings.ReplaceAll(schemaName, "]", "]]"),
			strings.ReplaceAll(tableName, "]", "]]"),
			strings.ReplaceAll(name, "]", "]]"),
		)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop foreign key %s: %w", name, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating foreign keys: %w", err)
	}

	for _, viewName := range requiredViews() {
		query := fmt.Sprintf("IF OBJECT_ID('%s', 'V') IS NOT NULL DROP VIEW %s", viewName, viewName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop view %s: %w", viewName, err)
		}
	}

	for _, tableName := range dropTableOrder() {
		query := fmt.Sprintf("IF OBJECT_ID('%s', 'U') IS NOT NULL DROP TABLE %s", tableName, tableName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}
	return nil
}

func (db *MSSQLConfig) ResetSchema(s *state.State) error {
	if s == nil {
		s = &state.State{}
	}
	if !s.Quiet {
		fmt.Println(color.YellowString("Warning: Re-initializing the database schema will delete all existing data."))
	}

	if err := db.DropAllTables(); err != nil {
		return err
	}

	return db.EnforceSchema(s)
}

func (db *MSSQLConfig) RunAction(action ActionConfig) error {
	var query string
	var args []interface{}

	if cmd, ok := action.Args["command"].(string); ok {
		query = db.GetSQL(cmd)
	} else if proc, ok := action.Args["procedure"].(string); ok {
		query = fmt.Sprintf("EXEC %s", proc)
	} else if q, ok := action.Args["query"].(string); ok {
		query = q
	} else {
		return fmt.Errorf("mssql action requires 'command', 'procedure', or 'query'")
	}

	if query == "" {
		return fmt.Errorf("SQL command not found or query is empty")
	}

	if params, ok := action.Args["args"].([]interface{}); ok {
		args = params
	}

	_, err := db.db.Exec(query, args...)
	return err
}

func (db *MSSQLConfig) GetTables() ([]string, error) {
	rows, err := db.db.Query("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE IN ('BASE TABLE','VIEW')")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func (db *MSSQLConfig) ExecuteQuery(query string) (*sql.Rows, error) {
	return db.db.Query(query)
}

// NewDBFromConfig creates a new DB instance from a config struct.
func NewDB(config *DBConfig) (DB, error) {
	var db DB
	switch config.Type {
	case "sqlite3":
		db = &SQLiteConfig{}
	case "postgres":
		db = &PostgreSQLConfig{
			Port: 5432,
		}
	case "mssql":
		db = &MSSQLConfig{
			Port: 1433,
		}
	default:
		db = &SQLiteConfig{}
	}

	db.LoadConfig(config)

	return db, nil
}

func (db *PostgreSQLConfig) DatabaseConnectionWithTimeout() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(db.Username, db.Password),
		Host:   fmt.Sprintf("%s:%d", db.Host, db.Port),
		Path:   db.Database,
	}
	q := u.Query()
	q.Set("sslmode", db.SSLMode)
	q.Set("connect_timeout", "5")
	u.RawQuery = q.Encode()
	return u.String()
}

func (db *MSSQLConfig) DatabaseConnectionWithTimeout() string {
	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(db.Username, db.Password),
		Host:   fmt.Sprintf("%s:%d", db.Host, db.Port),
	}
	q := u.Query()
	q.Set("database", db.Database)
	q.Set("connect timeout", "5")
	u.RawQuery = q.Encode()
	return u.String()
}

func (db *SQLiteConfig) DatabaseConnectionWithTimeout() string {
	return db.DatabaseConnection()
}

func RequiredTables() []string {
	return []string{
		"Accounts",
		"AccountCheckins",
		"AccountLocations",
		"AccountsPendingChanges",
		"AccountCheckinsPendingChanges",
		"Routes",
		"RouteWaypoints",
		"UserProfiles",
		"DataSets",
		"DataSetValues",
		"FieldMaps",
		"Configurations",
		"JobLog",
		"CommandLog",
		"WebhookLog",
		"SchemaMigrations",
	}
}

func dropTableOrder() []string {
	tables := RequiredTables()
	reversed := make([]string, len(tables))
	for i := range tables {
		reversed[i] = tables[len(tables)-1-i]
	}
	return reversed
}

func requiredViews() []string {
	return []string{
		"AccountsWithLabels",
		"AccountsIndexed",
		"AccountsIndexedColumns",
	}
}

func validateRequiredViews(db DB, s *state.State) error {
	for _, viewName := range requiredViews() {
		if s.Verbose && !s.Quiet {
			fmt.Printf("Checking view: %s... ", viewName)
		}
		exists, err := db.ViewExists(viewName)
		if err != nil {
			if s.Verbose && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("error checking if view %s exists: %w", viewName, err)
		}
		if !exists {
			if s.Verbose && !s.Quiet {
				fmt.Println(color.RedString("MISSING"))
			}
			return fmt.Errorf("required view %s does not exist", viewName)
		}
		if s.Verbose && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}
	return nil
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")
var mssqlNotNullRegex = regexp.MustCompile(`(?i)\bNOT\s+NULL\b`)
var mssqlNullRegex = regexp.MustCompile(`(?i)\bNULL\b`)
var mssqlPrimaryKeyRegex = regexp.MustCompile(`(?i)\bPRIMARY\s+KEY\b`)
var mssqlUniqueRegex = regexp.MustCompile(`(?i)\bUNIQUE\b`)
var mssqlIdentityRegex = regexp.MustCompile(`(?i)\bIDENTITY\s*\([^)]*\)`)

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func ToPascalCase(str string) string {
	str = strings.ReplaceAll(str, "_", " ")
	str = strings.Title(str)
	return strings.ReplaceAll(str, " ", "")
}

func CreateCommandForTable(tableName string) string {
	return "Create" + ToPascalCase(tableName) + "Table"
}

func RunCommand(db DB, command string, args ...any) error {
	return runCommand(db, db.GetDB(), command, args...)
}

func RunCommandTx(db DB, tx *sql.Tx, command string, args ...any) error {
	return runCommand(db, tx, command, args...)
}

func runCommand(db DB, executor interface {
	Exec(query string, args ...any) (sql.Result, error)
}, command string, args ...any) error {
	sqlText := db.GetSQL(command)
	if sqlText == "" {
		return fmt.Errorf("unknown or unavailable SQL command: %s", command)
	}
	_, err := executor.Exec(sqlText, args...)
	return err
}

func UpdateConfiguration(db DB, key string, value string) error {
	return RunCommand(db, "UpdateConfiguration", value, key)
}

func UpdateConfigurationTx(db DB, tx *sql.Tx, key string, value string) error {
	return RunCommandTx(db, tx, "UpdateConfiguration", value, key)
}

var existingSchemaMigrations = []schemaMigration{
	{
		Version: 1,
		Name:    "enforce current schema",
		Apply: func(db DB, tx *sql.Tx) error {
			s := &state.State{Quiet: true}
			switch typedDB := db.(type) {
			case *SQLiteConfig:
				return typedDB.enforceSchema(s, tx)
			case *PostgreSQLConfig:
				return typedDB.enforceSchema(s, tx)
			case *MSSQLConfig:
				return typedDB.enforceSchema(s, tx)
			default:
				return fmt.Errorf("unsupported database type %q", db.GetType())
			}
		},
	},
	{
		Version: 2,
		Name:    "create AccountsIndexedColumns view",
		Apply: func(db DB, tx *sql.Tx) error {
			return createAccountsIndexedColumnsView(db, tx)
		},
	},
}

func runSchemaMigrations(db DB, migrations []schemaMigration) error {
	if db == nil || db.GetDB() == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	createSQL := db.GetSQL("CreateSchemaMigrationsTable")
	if createSQL == "" {
		return fmt.Errorf("failed to load SQL command 'CreateSchemaMigrationsTable' for database type '%s'", db.GetType())
	}
	setupTx, err := db.GetDB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin schema migration setup: %w", err)
	}
	if _, err := setupTx.Exec(createSQL); err != nil {
		_ = setupTx.Rollback()
		return fmt.Errorf("failed to create schema migration metadata: %w", err)
	}
	if err := setupTx.Commit(); err != nil {
		return fmt.Errorf("failed to commit schema migration metadata: %w", err)
	}

	versionSQL := db.GetSQL("GetSchemaVersion")
	if versionSQL == "" {
		return fmt.Errorf("failed to load SQL command 'GetSchemaVersion' for database type '%s'", db.GetType())
	}
	var currentVersion int
	if err := db.GetDB().QueryRow(versionSQL).Scan(&currentVersion); err != nil {
		return fmt.Errorf("failed to read schema version: %w", err)
	}

	previousVersion := 0
	for _, migration := range migrations {
		if migration.Version <= previousVersion {
			return fmt.Errorf("schema migrations must have strictly increasing versions: %d follows %d", migration.Version, previousVersion)
		}
		previousVersion = migration.Version
		if migration.Version <= currentVersion {
			continue
		}
		if migration.Apply == nil {
			return fmt.Errorf("schema migration %d (%s) has no implementation", migration.Version, migration.Name)
		}

		if err := applySchemaMigration(db, migration); err != nil {
			return err
		}
		currentVersion = migration.Version
	}
	return nil
}

func applySchemaMigration(db DB, migration schemaMigration) error {
	tx, err := db.GetDB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin schema migration %d (%s): %w", migration.Version, migration.Name, err)
	}
	defer tx.Rollback()

	if err := migration.Apply(db, tx); err != nil {
		return fmt.Errorf("schema migration %d (%s) failed: %w", migration.Version, migration.Name, err)
	}
	if err := RunCommandTx(db, tx, "InsertSchemaMigration", migration.Version, migration.Name); err != nil {
		return fmt.Errorf("failed to record schema migration %d (%s): %w", migration.Version, migration.Name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit schema migration %d (%s): %w", migration.Version, migration.Name, err)
	}
	return nil
}

// UpgradeExistingSchema reapplies idempotent schema definitions without creating
// a schema for a database that has not completed initial setup.
func UpgradeExistingSchema(db DB) error {
	if db == nil || db.GetDB() == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	exists, err := db.TableExists("Configurations")
	if err != nil {
		return fmt.Errorf("failed to inspect existing schema: %w", err)
	}
	if !exists {
		return nil
	}
	if err := runSchemaMigrations(db, existingSchemaMigrations); err != nil {
		return fmt.Errorf("failed to upgrade existing schema: %w", err)
	}
	return nil
}

func LogCommand(db DB, command string, args []string, success bool, errorMessage string) error {
	return RunCommand(db, "InsertCommandLog", command, strings.Join(args, " "), success, errorMessage)
}

func LogWebhook(db DB, receivedAt time.Time, method, uri, headers, body string) error {
	return RunCommand(db, "InsertWebhookLog", receivedAt, method, uri, headers, body)
}

func GetWebhookLog(db DB, id int) (method, uri, headers, body string, err error) {
	sqlText := db.GetSQL("GetWebhookLog")
	if sqlText == "" {
		err = fmt.Errorf("unknown or unavailable SQL command: GetWebhookLog")
		return
	}
	sqlDB := db.GetDB()
	err = sqlDB.QueryRow(sqlText, id).Scan(&method, &uri, &headers, &body)
	return
}

func GetExpectedSchema() map[string][]string {
	return map[string][]string{
		"Accounts": {
			"AccountId", "FirstName", "LastName", "FullName", "PhoneNumber", "Email", "CustomerId", "Notes",
			"OriginalAddress", "CrmId", "AccountOwner", "DaysSinceLastCheckin", "LastCheckinDate",
			"LastModifiedDate", "FollowUpDate", "CustomNumeric", "CustomText", "CustomNumeric2",
			"CustomText2", "CustomNumeric3", "CustomText3", "CustomNumeric4", "CustomText4",
			"CustomNumeric5", "CustomText5", "CustomNumeric6", "CustomText6", "CustomNumeric7",
			"CustomText7", "CustomNumeric8", "CustomText8", "CustomNumeric9", "CustomText9",
			"CustomNumeric10", "CustomText10", "CustomNumeric11", "CustomText11", "CustomNumeric12",
			"CustomText12", "CustomNumeric13", "CustomText13", "CustomNumeric14", "CustomText14",
			"CustomNumeric15", "CustomText15", "CustomNumeric16", "CustomText16", "CustomNumeric17",
			"CustomText17", "CustomNumeric18", "CustomText18", "CustomNumeric19", "CustomText19",
			"CustomNumeric20", "CustomText20", "CustomNumeric21", "CustomText21", "CustomNumeric22",
			"CustomText22", "CustomNumeric23", "CustomText23", "CustomNumeric24", "CustomText24",
			"CustomNumeric25", "CustomText25", "CustomNumeric26", "CustomText26", "CustomNumeric27",
			"CustomText27", "CustomNumeric28", "CustomText28", "CustomNumeric29", "CustomText29",
			"CustomNumeric30", "CustomText30", "CreatedAt", "UpdatedAt",
		},
		"AccountCheckins": {
			"CheckinId", "CrmId", "AccountId", "LogDatetime", "Type", "Comments", "ExtraFields", "EndpointType", "CreatedBy",
			"CreatedAt", "UpdatedAt",
		},
		"AccountLocations": {
			"LocationId", "AccountId", "City", "Name", "Zipcode", "Longitude", "State",
			"Latitude", "AddressLine1", "Location", "IsApproximate", "CreatedAt", "UpdatedAt",
		},
		"AccountsPendingChanges": {
			"ChangeId", "AccountId", "ChangeType", "Changes", "Status", "CreatedAt", "ProcessedAt",
		},
		"AccountCheckinsPendingChanges": {
			"ChangeId", "CheckinId", "AccountId", "CrmId", "LogDatetime", "Type", "Comments", "ExtraFields", "EndpointType", "CreatedBy", "ChangeType", "Status", "CreatedAt", "ProcessedAt",
		},
		"Routes": {
			"RouteId", "Name", "RouteDate", "Duration", "StartAddress", "DestinationAddress", "StartTime",
			"CreatedAt", "UpdatedAt",
		},
		"RouteWaypoints": {
			"WaypointId", "RouteId", "Name", "Address", "Suite", "City", "State", "Zipcode", "Location",
			"Latitude", "Longitude", "LayoverMinutes", "Position", "CompleteAddress", "LocationId",
			"CustomerId", "ApptTime", "Type", "PlaceId", "CreatedAt", "UpdatedAt",
		},
		"JobLog": {
			"HistoryId", "CorrelationId", "ParentCorrelationId", "RootCorrelationId", "RunType", "Direction", "Source", "Initiator",
			"JobKind", "Mode", "StepId", "StepIndex", "TotalSteps", "ActionType", "CommandText", "Status", "ItemsProcessed",
			"ErrorCount", "StartedAt", "StartedAtTimezone", "CompletedAt", "CompletedAtTimezone", "DurationSeconds", "Summary", "Details",
		},
		"UserProfiles": {
			"ProfileId", "Email", "FirstName", "LastName", "IsManager", "IsHideReferralIOSBanner",
			"MarkerIcon", "Manager", "CRMEditableFieldsList", "CRMBaseUrl", "CRMType", "ReferralURL",
			"MapStartZoom", "MapStart", "IsUserCanEdit", "IsUserCanDeleteCheckins",
			"IsUserCanAddNewTextValues", "HasData", "DefaultApptLength", "Completed", "TrialDaysLeft",
			"CompanyId", "CompanyName", "CompanyShortName", "CreatedAt", "UpdatedAt",
		},
		"DataSets": {
			"Name", "ProfileId", "Filterable", "Label", "Position", "Type", "HasData",
			"IsUserCanAddNewTextValues", "RawMin", "Min", "Max", "RawMax", "AccountField", "CreatedAt", "UpdatedAt",
		},
		"DataSetValues": {
			"DataSetName", "ProfileId", "Text", "Value", "DataSetPosition", "CreatedAt", "UpdatedAt",
		},
		"FieldMaps": {
			"FieldName", "ObjectType", "JsonField", "DataSetName", "DataSetLabel",
		},
		"Configurations": {
			"SettingKey", "SettingValue", "LastModified",
		},
		"SchemaMigrations": {
			"Version", "Name", "AppliedAt",
		},
	}
}
