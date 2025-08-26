package database

import (
	"badgermapscli/utils"
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"               // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"     // SQLite driver
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
	"github.com/spf13/viper"
)

type DB interface {
	// GetType returns the database driver type (e.g., "sqlite3", "postgres", "mssql")
	GetType() string

	// GetDatabaseConnection returns the url connection string for the database
	DatabaseConnection() string

	// GetDatabaseSettings gets the database fields from the config file
	GetDatabaseSettings() error

	// SetDatabaseSettings sets the database fields in the config file
	SetDatabaseSettings() error

	// PromptDatabaseSettings prompts the user for database configuration
	PromptDatabaseSettings()

	TableExists(tableName string) (bool, error)

	ValidateSchema() error

	EnforceSchema() error

	TestConnection() error

	RunCommand() error
}

// SQLiteConfig represents a SQLite database configuration
type SQLiteConfig struct {
	Path string
}

// RunCommand implements the DB interface for SQLiteConfig.
// Note: Prefer using the top-level RunCommand(db, command, args...) helper to execute a specific SQL command.
func (db *SQLiteConfig) RunCommand() error { return nil }

func (db *SQLiteConfig) EnforceSchema() error {
	// Open a connection to the SQLite database
	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	defer sqlDB.Close()

	// Ensure required tables exist
	for _, tableName := range RequiredTables() {
		exists, err := db.TableExists(tableName)
		if err != nil {
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			createCmd := createCommandForTable(tableName)
			if createCmd == "" {
				return fmt.Errorf("no create command mapped for table %s", tableName)
			}
			sqlText := sqlCommandLoader(db.GetType(), createCmd)
			if sqlText == "" {
				return fmt.Errorf("failed to load SQL for command %s (%s)", createCmd, db.GetType())
			}
			if _, err := sqlDB.Exec(sqlText); err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
		}
	}

	// Ensure RouteWaypoints has columns required by insert statements (idempotent updates)
	// This helps existing databases created with older schemas.
	ensureColumn := func(table, column, colType string) error {
		var cnt int
		q := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name=?", table)
		if err := sqlDB.QueryRow(q, column).Scan(&cnt); err != nil {
			return fmt.Errorf("failed to introspect columns for %s: %w", table, err)
		}
		if cnt == 0 {
			alter := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colType)
			if _, err := sqlDB.Exec(alter); err != nil {
				return fmt.Errorf("failed to add column %s to %s: %w", column, table, err)
			}
		}
		return nil
	}
	if exists, _ := db.TableExists("RouteWaypoints"); exists {
		if err := ensureColumn("RouteWaypoints", "CompleteAddress", "TEXT"); err != nil {
			return err
		}
		if err := ensureColumn("RouteWaypoints", "ApptTime", "DATETIME"); err != nil {
			return err
		}
		if err := ensureColumn("RouteWaypoints", "PlaceId", "TEXT"); err != nil {
			return err
		}
	}

	// Ensure indexes exist (idempotent SQL expected)
	if idxSQL := sqlCommandLoader(db.GetType(), "create_indexes"); idxSQL != "" {
		if _, err := sqlDB.Exec(idxSQL); err != nil {
			return fmt.Errorf("failed to create indexes: %w", err)
		}
	}

	return nil
}

func (db *SQLiteConfig) TestConnection() error {
	// Open a connection to the SQLite database
	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	defer sqlDB.Close()

	// Test the connection with a ping
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	// If we got here, the connection was successful
	return nil
}

func (db *SQLiteConfig) ValidateSchema() error {
	// Open a connection to the SQLite database
	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	defer sqlDB.Close()

	for _, tableName := range RequiredTables() {
		exists, err := db.TableExists(tableName)
		if err != nil {
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			return fmt.Errorf("required table %s does not exist", tableName)
		}
	}

	return nil
}

func (db *SQLiteConfig) TableExists(tableName string) (bool, error) {
	// Open a connection to the SQLite database
	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		return false, fmt.Errorf("failed to open SQLite database: %w", err)
	}
	defer sqlDB.Close()

	// Query to check if table exists in SQLite
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
	var name string
	err = sqlDB.QueryRow(query, tableName).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if table '%s' exists in SQLite: %w", tableName, err)
	}

	return true, nil
}

// GetType returns "sqlite3" for SQLite databases
func (db *SQLiteConfig) GetType() string {
	return "sqlite3"
}

// GetDatabaseSettings gets the database fields from the config file
func (db *SQLiteConfig) GetDatabaseSettings() error {
	// For SQLite, we only need to get the database path
	db.Path = viper.GetString("DATABASE_NAME")

	// If path is empty, use default
	if db.Path == "" {
		db.Path = utils.GetConfigDirFile("badgermaps.db")
	}

	return nil
}

// SetDatabaseSettings sets the database fields in the config file
func (db *SQLiteConfig) SetDatabaseSettings() error {
	viper.Set("DATABASE_NAME", db.Path)
	return nil
}

// GetDatabasePath returns the path to the SQLite database file
func (db *SQLiteConfig) GetDatabasePath() string {
	return db.Path
}

// GetDatabaseConnection returns the url connection string for SQLite
func (db *SQLiteConfig) DatabaseConnection() string {
	return fmt.Sprintf("file:%s?mode=rwc", db.Path)
}

func (db *SQLiteConfig) PromptDatabaseSettings() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(utils.Colors.Cyan("SQLite Database Configuration"))
	db.Path = utils.PromptString(reader, "Database Path", db.Path)
}

// PostgreSQLConfig represents a PostgreSQL database configuration
type PostgreSQLConfig struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string
}

// RunCommand implements the DB interface for PostgreSQLConfig.
// Note: Prefer using the top-level RunCommand(db, command, args...) helper to execute a specific SQL command.
func (db *PostgreSQLConfig) RunCommand() error { return nil }

func (db *PostgreSQLConfig) EnforceSchema() error {
	// Open a connection to the PostgreSQL database
	sqlDB, err := sql.Open("postgres", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	defer sqlDB.Close()

	for _, tableName := range RequiredTables() {
		exists, err := db.TableExists(tableName)
		if err != nil {
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			createCmd := createCommandForTable(tableName)
			if createCmd == "" {
				return fmt.Errorf("no create command mapped for table %s", tableName)
			}
			sqlText := sqlCommandLoader(db.GetType(), createCmd)
			if sqlText == "" {
				return fmt.Errorf("failed to load SQL for command %s (%s)", createCmd, db.GetType())
			}
			if _, err := sqlDB.Exec(sqlText); err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
		}
	}

	if idxSQL := sqlCommandLoader(db.GetType(), "create_indexes"); idxSQL != "" {
		if _, err := sqlDB.Exec(idxSQL); err != nil {
			return fmt.Errorf("failed to create indexes: %w", err)
		}
	}

	return nil
}

func (db *PostgreSQLConfig) TestConnection() error {

	// Open a connection to the PostgreSQL database
	sqlDB, err := sql.Open("postgres", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	defer sqlDB.Close()

	// Test the connection with a ping
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
	}

	// If we got here, the connection was successful
	return nil
}

func (db *PostgreSQLConfig) ValidateSchema() error {
	// Open a connection to the PostgreSQL database
	sqlDB, err := sql.Open("postgres", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	defer sqlDB.Close()

	for _, tableName := range RequiredTables() {
		exists, err := db.TableExists(tableName)
		if err != nil {
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			return fmt.Errorf("required table %s does not exist", tableName)
		}
	}

	return nil
}

func (db *PostgreSQLConfig) TableExists(tableName string) (bool, error) {
	// Open a connection to the PostgreSQL database
	sqlDB, err := sql.Open("postgres", db.DatabaseConnection())
	if err != nil {
		return false, fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	defer sqlDB.Close()

	// Query to check if table exists in PostgreSQL
	query := "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)"
	var exists bool
	err = sqlDB.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if table '%s' exists in PostgreSQL: %w", tableName, err)
	}

	return exists, nil
}

// GetType returns "postgres" for PostgreSQL databases
func (db *PostgreSQLConfig) GetType() string {
	return "postgres"
}

// GetConnectionInfo returns the connection information for PostgreSQL
func (db *PostgreSQLConfig) GetConnectionInfo() map[string]string {
	return map[string]string{
		"host":     db.Host,
		"port":     db.Port,
		"database": db.Database,
		"username": db.Username,
		"password": db.Password,
	}
}

// GetHost returns the host for PostgreSQL
func (db *PostgreSQLConfig) GetHost() string {
	return db.Host
}

// GetPort returns the port for PostgreSQL
func (db *PostgreSQLConfig) GetPort() string {
	return db.Port
}

// GetUsername returns the username for PostgreSQL
func (db *PostgreSQLConfig) GetUsername() string {
	return db.Username
}

// GetPassword returns the password for PostgreSQL
func (db *PostgreSQLConfig) GetPassword() string {
	return db.Password
}

// GetDatabaseConnection returns the url connection string for PostgreSQL
func (db *PostgreSQLConfig) DatabaseConnection() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=enable", db.Username, db.Password, db.Host, db.Port, db.Database)
}

// GetDatabaseSettings gets the database fields from the config file
func (db *PostgreSQLConfig) GetDatabaseSettings() error {
	// Get PostgreSQL configuration from viper
	db.Host = viper.GetString("DATABASE_HOST")
	db.Port = viper.GetString("DATABASE_PORT")
	db.Database = viper.GetString("DATABASE_NAME")
	db.Username = viper.GetString("DATABASE_USER")
	db.Password = viper.GetString("DATABASE_PASSWORD")

	// Set defaults if values are empty
	if db.Host == "" {
		db.Host = "localhost"
	}
	if db.Port == "" {
		db.Port = "5432"
	}
	if db.Database == "" {
		db.Database = "badgermaps"
	}

	return nil
}

// SetDatabaseSettings sets the database fields in the config file
func (db *PostgreSQLConfig) SetDatabaseSettings() error {
	viper.Set("DATABASE_HOST", db.Host)
	viper.Set("DATABASE_PORT", db.Port)
	viper.Set("DATABASE_NAME", db.Database)
	viper.Set("DATABASE_USER", db.Username)
	viper.Set("DATABASE_PASSWORD", db.Password)
	return nil
}

// PromptDatabaseSettings prompts the user for PostgreSQL database configuration
func (db *PostgreSQLConfig) PromptDatabaseSettings() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(utils.Colors.Cyan("PostgreSQL Database Configuration"))
	db.Host = utils.PromptString(reader, "Host", db.Host)
	db.Port = utils.PromptString(reader, "Port", db.Port)
	db.Database = utils.PromptString(reader, "Database Name", db.Database)
	db.Username = utils.PromptString(reader, "Username", db.Username)
	db.Password = utils.PromptPassword(reader, "Password")
}

// MSSQLConfig represents a Microsoft SQL Server database configuration
type MSSQLConfig struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string
}

// RunCommand implements the DB interface for MSSQLConfig.
// Note: Prefer using the top-level RunCommand(db, command, args...) helper to execute a specific SQL command.
func (db *MSSQLConfig) RunCommand() error { return nil }

func (db *MSSQLConfig) EnforceSchema() error {
	// Open a connection to the MSSQL database
	sqlDB, err := sql.Open("mssql", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open MSSQL database: %w", err)
	}
	defer sqlDB.Close()

	for _, tableName := range RequiredTables() {
		exists, err := db.TableExists(tableName)
		if err != nil {
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			createCmd := createCommandForTable(tableName)
			if createCmd == "" {
				return fmt.Errorf("no create command mapped for table %s", tableName)
			}
			sqlText := sqlCommandLoader(db.GetType(), createCmd)
			if sqlText == "" {
				return fmt.Errorf("failed to load SQL for command %s (%s)", createCmd, db.GetType())
			}
			if _, err := sqlDB.Exec(sqlText); err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
		}
	}

	if idxSQL := sqlCommandLoader(db.GetType(), "create_indexes"); idxSQL != "" {
		if _, err := sqlDB.Exec(idxSQL); err != nil {
			return fmt.Errorf("failed to create indexes: %w", err)
		}
	}

	return nil
}

func (db *MSSQLConfig) TestConnection() error {
	// Open a connection to the MSSQL database
	sqlDB, err := sql.Open("mssql", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open MSSQL database: %w", err)
	}
	defer sqlDB.Close()

	// Test the connection with a ping
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to connect to MSSQL database: %w", err)
	}

	// If we got here, the connection was successful
	return nil
}

func (db *MSSQLConfig) ValidateSchema() error {
	// Open a connection to the MSSQL database
	sqlDB, err := sql.Open("mssql", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open MSSQL database: %w", err)
	}
	defer sqlDB.Close()

	// Check if required tables exist

	for _, tableName := range RequiredTables() {
		exists, err := db.TableExists(tableName)
		if err != nil {
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			return fmt.Errorf("required table %s does not exist", tableName)
		}
	}

	return nil
}

func (db *MSSQLConfig) TableExists(tableName string) (bool, error) {
	// Open a connection to the MSSQL database
	sqlDB, err := sql.Open("mssql", db.DatabaseConnection())
	if err != nil {
		return false, fmt.Errorf("failed to open MSSQL database: %w", err)
	}
	defer sqlDB.Close()

	// Query to check if table exists in MSSQL
	query := "SELECT CASE WHEN EXISTS (SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = @p1) THEN 1 ELSE 0 END"
	var exists bool
	err = sqlDB.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if table '%s' exists in MSSQL: %w", tableName, err)
	}

	return exists, nil
}

// GetType returns "mssql" for Microsoft SQL Server databases
func (db *MSSQLConfig) GetType() string {
	return "mssql"
}

// GetConnectionInfo returns the connection information for Microsoft SQL Server
func (db *MSSQLConfig) GetConnectionInfo() map[string]string {
	return map[string]string{
		"host":     db.Host,
		"port":     db.Port,
		"database": db.Database,
		"username": db.Username,
		"password": db.Password,
	}
}

// GetHost returns the host for Microsoft SQL Server
func (db *MSSQLConfig) GetHost() string {
	return db.Host
}

// GetPort returns the port for Microsoft SQL Server
func (db *MSSQLConfig) GetPort() string {
	return db.Port
}

// GetUsername returns the username for Microsoft SQL Server
func (db *MSSQLConfig) GetUsername() string {
	return db.Username
}

// GetPassword returns the password for Microsoft SQL Server
func (db *MSSQLConfig) GetPassword() string {
	return db.Password
}

// GetDatabaseConnection returns the url connection string for Microsoft SQL Server
func (db *MSSQLConfig) DatabaseConnection() string {
	return fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s", db.Username, db.Password, db.Host, db.Port, db.Database)
}

// GetDatabaseSettings gets the database fields from the config file
func (db *MSSQLConfig) GetDatabaseSettings() error {
	// Get MSSQL configuration from viper
	db.Host = viper.GetString("DATABASE_HOST")
	db.Port = viper.GetString("DATABASE_PORT")
	db.Database = viper.GetString("DATABASE_NAME")
	db.Username = viper.GetString("DATABASE_USER")
	db.Password = viper.GetString("DATABASE_PASSWORD")

	// Set defaults if values are empty
	if db.Host == "" {
		db.Host = "localhost"
	}
	if db.Port == "" {
		db.Port = "1433"
	}
	if db.Database == "" {
		db.Database = "badgermaps"
	}

	return nil
}

// SetDatabaseSettings sets the database fields in the config file
func (db *MSSQLConfig) SetDatabaseSettings() error {
	viper.Set("DATABASE_HOST", db.Host)
	viper.Set("DATABASE_PORT", db.Port)
	viper.Set("DATABASE_NAME", db.Database)
	viper.Set("DATABASE_USER", db.Username)
	viper.Set("DATABASE_PASSWORD", db.Password)
	return nil
}

// PromptDatabaseSettings prompts the user for Microsoft SQL Server database configuration
func (db *MSSQLConfig) PromptDatabaseSettings() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(utils.Colors.Cyan("Microsoft SQL Server Database Configuration"))
	db.Host = utils.PromptString(reader, "Host", db.Host)
	db.Port = utils.PromptString(reader, "Port", db.Port)
	db.Database = utils.PromptString(reader, "Database Name", db.Database)
	db.Username = utils.PromptString(reader, "Username", db.Username)
	db.Password = utils.PromptPassword(reader, "Password")
}

// LoadDatabaseSettings loads database settings based on the database type
func LoadDatabaseSettings(dbType string) (DB, error) {
	var db DB
	switch dbType {
	case "sqlite3":
		db = &SQLiteConfig{
			Path: utils.GetConfigDirFile("badgermaps.db"),
		}
	case "postgres":
		db = &PostgreSQLConfig{
			Host:     "localhost",
			Port:     "5432",
			Database: "badgermaps",
			Username: "",
			Password: "",
		}
	case "mssql":
		db = &MSSQLConfig{
			Host:     "localhost",
			Port:     "1433",
			Database: "badgermaps",
			Username: "",
			Password: "",
		}
	default:
		db = &SQLiteConfig{
			Path: utils.GetConfigDirFile("badgermaps.db"),
		}
	}
	err := db.GetDatabaseSettings()
	return db, err
}

// RequiredTables Returns a list of tables required for the app to work
func RequiredTables() []string {

	return []string{
		"Accounts",
		"AccountCheckins",
		"AccountsPendingChanges",
		"Routes",
		"UserProfiles",
		"AccountLocations",
		"RouteWaypoints",
		"DataSets",
		"DataSetValues",
	}

}

// sqlCommandsList returns a list of sql commands that are available through loading.
func sqlCommandsList() []string {
	return []string{
		"check_column_exists",
		"check_index_exists",
		"check_table_exists",
		"create_account_checkins_table",
		"create_account_locations_table",
		"create_accounts_pending_changes_table",
		"create_accounts_table",
		"create_data_set_values_table",
		"create_data_sets_table",
		"create_indexes",
		"create_route_waypoints_table",
		"create_routes_table",
		"create_user_profiles_table",
		"delete_account_locations",
		"delete_data_set_values",
		"delete_data_sets",
		"delete_route_waypoints",
		"insert_account_locations",
		"insert_data_set_values",
		"insert_data_sets",
		"insert_route_waypoints",
		"merge_account_checkins",
		"merge_accounts_basic",
		"merge_accounts_detailed",
		"merge_routes",
		"merge_user_profiles",
		"search_accounts",
		"search_routes",
	}
}

func create_commands_list() []string {
	create_commands := []string{}
	for _, cmd := range sqlCommandsList() {
		if strings.HasPrefix(cmd, "create") {
			create_commands = append(create_commands, cmd)
		}
	}

	return create_commands
}

// createCommandForTable maps logical table names to the corresponding create_* SQL command filename (without extension)
func createCommandForTable(tableName string) string {
	switch tableName {
	case "Accounts":
		return "create_accounts_table"
	case "AccountCheckins":
		return "create_account_checkins_table"
	case "AccountsPendingChanges":
		return "create_accounts_pending_changes_table"
	case "Routes":
		return "create_routes_table"
	case "UserProfiles":
		return "create_user_profiles_table"
	case "AccountLocations":
		return "create_account_locations_table"
	case "RouteWaypoints":
		return "create_route_waypoints_table"
	case "DataSets":
		return "create_data_sets_table"
	case "DataSetValues":
		return "create_data_set_values_table"
	default:
		return ""
	}
}

// sqlCommandLoader Loads a specific command for the corresponding database.
func sqlCommandLoader(dbType string, command string) string {
	// Validate command exists in the known commands list
	valid := false
	for _, c := range sqlCommandsList() {
		if c == command {
			valid = true
			break
		}
	}
	if !valid {
		return ""
	}

	// Normalize/validate dbType; default to sqlite3 if unknown
	subdir := dbType
	switch dbType {
	case "postgres", "mssql", "sqlite3":
		// keep as is
	default:
		subdir = "sqlite3"
	}

	path := fmt.Sprintf("database/%s/%s.sql", subdir, command)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// RunCommand loads a SQL command by name for the provided DB type and executes it with optional args.
// It opens a short-lived connection, executes, and closes it. The SQL must exist under database/<dbtype>/<command>.sql
func RunCommand(db DB, command string, args ...any) error {
	// Load SQL text for the command
	sqlText := sqlCommandLoader(db.GetType(), command)
	if sqlText == "" {
		return fmt.Errorf("unknown or unavailable SQL command: %s for dbType %s", command, db.GetType())
	}

	// Open connection
	sqlDB, err := sql.Open(db.GetType(), db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open %s database: %w", db.GetType(), err)
	}
	defer sqlDB.Close()

	// Execute
	if _, err := sqlDB.Exec(sqlText, args...); err != nil {
		return fmt.Errorf("failed to execute command %s: %w", command, err)
	}
	return nil
}
