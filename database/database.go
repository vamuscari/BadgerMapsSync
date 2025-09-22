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
	query := db.GetSQL("GetTableColumns")

	rows, err := sqlDB.Query(query, tableName)
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
	if err := db.DropAllTables(); err != nil {
		return err
	}

	sqlDB := db.GetDB()

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
		if _, err := sqlDB.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}

	// Insert initial data for FieldMaps
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Inserting initial data for FieldMaps... ")
	}
	sqlText := db.GetSQL("InsertFieldMaps")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
	sqlText = db.GetSQL("CreateAccountsWithLabelsView")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create view AccountsWithLabels: %w", err)
		}
	}
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}

func (db *SQLiteConfig) TestConnection() error {
	err := db.GetDB().Ping()
	if err != nil {
		db.connected = false
		return err
	}
	db.connected = true
	return nil
}

func (db *SQLiteConfig) ValidateSchema(s *state.State) error {
	if db.db == nil {
		return nil
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

	if s.Verbose && !s.Quiet {
		fmt.Printf("Checking view: AccountsWithLabels... ")
	}
	exists, err := db.ViewExists("AccountsWithLabels")
	if err != nil {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("error checking if view AccountsWithLabels exists: %w", err)
	}
	if !exists {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required view AccountsWithLabels does not exist")
	}
	if s.Verbose && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
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
	for _, tableName := range RequiredTables() {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}
	return nil
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
	rows, err := db.db.Query("SELECT name FROM sqlite_master WHERE type='table'")
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
	return string(data)
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
		if _, err := sqlDB.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}

	// Insert initial data for FieldMaps
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Inserting initial data for FieldMaps... ")
	}
	sqlText := db.GetSQL("InsertFieldMaps")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
	if _, err := db.GetDB().Exec("SELECT AccountsWithLabelsView()"); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to execute AccountsWithLabelsView function: %w", err)
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
	err := db.GetDB().Ping()
	if err != nil {
		db.connected = false
		return err
	}
	db.connected = true
	return nil
}
func (db *PostgreSQLConfig) ValidateSchema(s *state.State) error {
	if db.db == nil {
		return nil
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

	if s.Verbose && !s.Quiet {
		fmt.Printf("Checking view: AccountsWithLabels... ")
	}
	viewExists, err := db.ViewExists("AccountsWithLabels")
	if err != nil {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("error checking if view AccountsWithLabels exists: %w", err)
	}
	if !viewExists {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required view AccountsWithLabels does not exist")
	}
	if s.Verbose && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
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
	for _, tableName := range RequiredTables() {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}
	return nil
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
	rows, err := db.db.Query("SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname != 'pg_catalog' AND schemaname != 'information_schema'")
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
		if _, err := sqlDB.Exec(sqlText); err != nil {
			if (s.Verbose || s.Debug) && !s.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}

	// Insert initial data for FieldMaps
	if (s.Verbose || s.Debug) && !s.Quiet {
		fmt.Printf("Inserting initial data for FieldMaps... ")
	}
	sqlText := db.GetSQL("InsertFieldMaps")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
	if _, err := db.GetDB().Exec("EXEC AccountsWithLabelsView"); err != nil {
		if (s.Verbose || s.Debug) && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to execute AccountsWithLabelsView procedure: %w", err)
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
		if _, err := db.GetDB().Exec(sqlText); err != nil {
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
	err := db.GetDB().Ping()
	if err != nil {
		db.connected = false
		return err
	}
	db.connected = true
	return nil
}
func (db *MSSQLConfig) ValidateSchema(s *state.State) error {
	if db.db == nil {
		return nil
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

	if s.Verbose && !s.Quiet {
		fmt.Printf("Checking view: AccountsWithLabels... ")
	}
	viewExists, err := db.ViewExists("AccountsWithLabels")
	if err != nil {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("error checking if view AccountsWithLabels exists: %w", err)
	}
	if !viewExists {
		if s.Verbose && !s.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required view AccountsWithLabels does not exist")
	}
	if s.Verbose && !s.Quiet {
		fmt.Println(color.GreenString("OK"))
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
	// First, drop all foreign key constraints
	// This is a bit of a heavy-handed approach, but it's reliable
	// A more elegant solution would be to drop tables in the correct order
	// but that requires parsing the schema, which is complex.

	rows, err := sqlDB.Query("SELECT name, object_id FROM sys.foreign_keys")
	if err != nil {
		return fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, objectId string
		if err := rows.Scan(&name, &objectId); err != nil {
			return fmt.Errorf("failed to scan foreign key: %w", err)
		}
		parentTableQuery := fmt.Sprintf("SELECT OBJECT_NAME(parent_object_id) FROM sys.foreign_keys WHERE object_id = %s", objectId)
		var parentTable string
		if err := sqlDB.QueryRow(parentTableQuery).Scan(&parentTable); err != nil {
			return fmt.Errorf("failed to get parent table for foreign key %s: %w", name, err)
		}
		query := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", parentTable, name)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop foreign key %s: %w", name, err)
		}
	}

	for _, tableName := range RequiredTables() {
		query := fmt.Sprintf("IF OBJECT_ID('%s', 'U') IS NOT NULL DROP TABLE %s", tableName, tableName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}
	return nil
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
	rows, err := db.db.Query("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE'")
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
		"SyncHistory",
		"CommandLog",
		"WebhookLog",
	}
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

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
	sqlText := db.GetSQL(command)
	if sqlText == "" {
		return fmt.Errorf("unknown or unavailable SQL command: %s", command)
	}
	sqlDB := db.GetDB()
	_, err := sqlDB.Exec(sqlText, args...)
	return err
}

func UpdateConfiguration(db DB, key string, value string) error {
	return RunCommand(db, "UpdateConfiguration", value, key)
}

func LogCommand(db DB, command string, args []string, success bool, errorMessage string) error {
	sqlText := "INSERT INTO CommandLog (Command, Args, Success, ErrorMessage) VALUES (?, ?, ?, ?)"
	sqlDB := db.GetDB()
	_, err := sqlDB.Exec(sqlText, command, strings.Join(args, " "), success, errorMessage)
	return err
}

func LogWebhook(db DB, receivedAt time.Time, method, uri, headers, body string) error {
	sqlText := "INSERT INTO WebhookLog (ReceivedAt, Method, Uri, Headers, Body) VALUES (?, ?, ?, ?, ?)"
	sqlDB := db.GetDB()
	_, err := sqlDB.Exec(sqlText, receivedAt, method, uri, headers, body)
	return err
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
			"CheckinId", "CrmId", "AccountId", "LogDatetime", "Type", "Comments", "ExtraFields", "CreatedBy",
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
			"ChangeId", "CheckinId", "ChangeType", "Changes", "Status", "CreatedAt", "ProcessedAt",
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
		"SyncHistory": {
			"HistoryId", "CorrelationId", "RunType", "Direction", "Source", "Trigger", "Status", "ItemsProcessed", "ErrorCount",
			"StartedAt", "CompletedAt", "DurationSeconds", "Summary", "Details",
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
	}
}
