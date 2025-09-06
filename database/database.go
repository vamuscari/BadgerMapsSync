package database

import (
	"badgermaps/utils"
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/lib/pq"               // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"     // SQLite driver
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
	"github.com/spf13/viper"
)

type DB interface {
	GetType() string
	DatabaseConnection() string
	GetDatabaseSettings() error
	SetDatabaseSettings() error
	PromptDatabaseSettings()
	TableExists(tableName string) (bool, error)
	ValidateSchema(verbose bool) error
	EnforceSchema(verbose bool) error
	TestConnection() error
	DropAllTables() error
}

// SQLiteConfig represents a SQLite database configuration
type SQLiteConfig struct {
	Path string `mapstructure:"DB_PATH"`
}

func (db *SQLiteConfig) EnforceSchema(verbose bool) error {
	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	defer sqlDB.Close()

	for _, tableName := range RequiredTables() {
		if verbose {
			fmt.Printf("Creating table: %s... ", tableName)
		}
		createCmd := createCommandForTable(tableName)
		sqlText := sqlCommandLoader(db.GetType(), createCmd)
		if _, err := sqlDB.Exec(sqlText); err != nil {
			if verbose {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if verbose {
			fmt.Println(color.GreenString("OK"))
		}
	}
	return nil
}

func (db *SQLiteConfig) TestConnection() error {
	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	defer sqlDB.Close()
	return sqlDB.Ping()
}

func (db *SQLiteConfig) ValidateSchema(verbose bool) error {
	for _, tableName := range RequiredTables() {
		if verbose {
			fmt.Printf("Checking table: %s... ", tableName)
		}
		exists, err := db.TableExists(tableName)
		if err != nil {
			if verbose {
				fmt.Println(color.RedString("ERROR"))
			}
			if viper.GetBool("debug") {
				fmt.Printf("ValidateSchema error: %v\n", err)
			}
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			if verbose {
				fmt.Println(color.RedString("MISSING"))
			}
			return fmt.Errorf("required table %s does not exist", tableName)
		}
		if verbose {
			fmt.Println(color.GreenString("OK"))
		}
	}
	return nil
}

func (db *SQLiteConfig) TableExists(tableName string) (bool, error) {
	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		return false, fmt.Errorf("failed to open SQLite database: %w", err)
	}
	defer sqlDB.Close()

	query := sqlCommandLoader(db.GetType(), "check_table_exists")
	var count int
	err = sqlDB.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		if viper.GetBool("debug") {
			fmt.Printf("TableExists error: %v\n", err)
		}
		return false, err
	}
	return count > 0, nil
}

func (db *SQLiteConfig) GetType() string {
	return "sqlite3"
}

func (db *SQLiteConfig) GetDatabaseSettings() error {
	db.Path = viper.GetString("DB_PATH")
	if db.Path == "" {
		db.Path = utils.GetConfigDirFile("badgermaps.db")
	}
	return nil
}

func (db *SQLiteConfig) SetDatabaseSettings() error {
	viper.Set("DB_PATH", db.Path)
	return nil
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
	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	defer sqlDB.Close()

	for _, tableName := range RequiredTables() {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}
	return nil
}

// PostgreSQLConfig represents a PostgreSQL database configuration
type PostgreSQLConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     int    `mapstructure:"DB_PORT"`
	Database string `mapstructure:"DB_NAME"`
	Username string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	SSLMode  string `mapstructure:"DB_SSL_MODE"`
}

func (db *PostgreSQLConfig) EnforceSchema(verbose bool) error {
	sqlDB, err := sql.Open("postgres", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	defer sqlDB.Close()

	for _, tableName := range RequiredTables() {
		if verbose {
			fmt.Printf("Creating table: %s... ", tableName)
		}
		createCmd := createCommandForTable(tableName)
		sqlText := sqlCommandLoader(db.GetType(), createCmd)
		if _, err := sqlDB.Exec(sqlText); err != nil {
			if verbose {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if verbose {
			fmt.Println(color.GreenString("OK"))
		}
	}
	return nil
}
func (db *PostgreSQLConfig) TestConnection() error {
	sqlDB, err := sql.Open("postgres", db.DatabaseConnection())
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return sqlDB.Ping()
}
func (db *PostgreSQLConfig) ValidateSchema(verbose bool) error {
	for _, tableName := range RequiredTables() {
		if verbose {
			fmt.Printf("Checking table: %s... ", tableName)
		}
		exists, err := db.TableExists(tableName)
		if err != nil {
			if verbose {
				fmt.Println(color.RedString("ERROR"))
			}
			if viper.GetBool("debug") {
				fmt.Printf("ValidateSchema error: %v\n", err)
			}
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			if verbose {
				fmt.Println(color.RedString("MISSING"))
			}
			return fmt.Errorf("required table %s does not exist", tableName)
		}
		if verbose {
			fmt.Println(color.GreenString("OK"))
		}
	}
	return nil
}
func (db *PostgreSQLConfig) TableExists(tableName string) (bool, error) {
	sqlDB, err := sql.Open("postgres", db.DatabaseConnection())
	if err != nil {
		return false, fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	defer sqlDB.Close()

	query := sqlCommandLoader(db.GetType(), "check_table_exists")
	var count int
	err = sqlDB.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		if viper.GetBool("debug") {
			fmt.Printf("TableExists error: %v\n", err)
		}
		return false, err
	}
	return count > 0, nil
}
func (db *PostgreSQLConfig) GetType() string {
	return "postgres"
}
func (db *PostgreSQLConfig) GetDatabaseSettings() error {
	db.Host = viper.GetString("DB_HOST")
	db.Port = viper.GetInt("DB_PORT")
	db.Database = viper.GetString("DB_NAME")
	db.Username = viper.GetString("DB_USER")
	db.Password = viper.GetString("DB_PASSWORD")
	db.SSLMode = viper.GetString("DB_SSL_MODE")
	return nil
}
func (db *PostgreSQLConfig) SetDatabaseSettings() error {
	viper.Set("DB_HOST", db.Host)
	viper.Set("DB_PORT", db.Port)
	viper.Set("DB_NAME", db.Database)
	viper.Set("DB_USER", db.Username)
	viper.Set("DB_PASSWORD", db.Password)
	viper.Set("DB_SSL_MODE", db.SSLMode)
	return nil
}
func (db *PostgreSQLConfig) DatabaseConnection() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", db.Username, db.Password, db.Host, db.Port, db.Database, db.SSLMode)
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
	sqlDB, err := sql.Open("postgres", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	defer sqlDB.Close()

	for _, tableName := range RequiredTables() {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)
		if _, err := sqlDB.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}
	return nil
}

// MSSQLConfig represents a Microsoft SQL Server database configuration
type MSSQLConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     int    `mapstructure:"DB_PORT"`
	Database string `mapstructure:"DB_NAME"`
	Username string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
}

func (db *MSSQLConfig) EnforceSchema(verbose bool) error {
	sqlDB, err := sql.Open("mssql", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open MSSQL database: %w", err)
	}
	defer sqlDB.Close()

	for _, tableName := range RequiredTables() {
		if verbose {
			fmt.Printf("Creating table: %s... ", tableName)
		}
		createCmd := createCommandForTable(tableName)
		sqlText := sqlCommandLoader(db.GetType(), createCmd)
		if _, err := sqlDB.Exec(sqlText); err != nil {
			if verbose {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if verbose {
			fmt.Println(color.GreenString("OK"))
		}
	}
	return nil
}
func (db *MSSQLConfig) TestConnection() error {
	sqlDB, err := sql.Open("mssql", db.DatabaseConnection())
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return sqlDB.Ping()
}
func (db *MSSQLConfig) ValidateSchema(verbose bool) error {
	for _, tableName := range RequiredTables() {
		if verbose {
			fmt.Printf("Checking table: %s... ", tableName)
		}
		exists, err := db.TableExists(tableName)
		if err != nil {
			if verbose {
				fmt.Println(color.RedString("ERROR"))
			}
			if viper.GetBool("debug") {
				fmt.Printf("ValidateSchema error: %v\n", err)
			}
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			if verbose {
				fmt.Println(color.RedString("MISSING"))
			}
			return fmt.Errorf("required table %s does not exist", tableName)
		}
		if verbose {
			fmt.Println(color.GreenString("OK"))
		}
	}
	return nil
}
func (db *MSSQLConfig) TableExists(tableName string) (bool, error) {
	sqlDB, err := sql.Open("mssql", db.DatabaseConnection())
	if err != nil {
		return false, fmt.Errorf("failed to open MSSQL database: %w", err)
	}
	defer sqlDB.Close()

	query := "SELECT COUNT(*) FROM sys.tables WHERE name = @p1"
	if viper.GetBool("debug") {
		fmt.Printf("DEBUG: Executing query: %s with arg: %s\n", query, tableName)
	}
	var count int
	err = sqlDB.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		if viper.GetBool("debug") {
			fmt.Printf("DEBUG: TableExists SQL ERROR: %v\n", err)
		}
		return false, err
	}
	if viper.GetBool("debug") {
		fmt.Printf("DEBUG: TableExists count for %s: %d\n", tableName, count)
	}
	return count > 0, nil
}
func (db *MSSQLConfig) GetType() string {
	return "mssql"
}
func (db *MSSQLConfig) GetDatabaseSettings() error {
	db.Host = viper.GetString("DB_HOST")
	db.Port = viper.GetInt("DB_PORT")
	db.Database = viper.GetString("DB_NAME")
	db.Username = viper.GetString("DB_USER")
	db.Password = viper.GetString("DB_PASSWORD")
	return nil
}
func (db *MSSQLConfig) SetDatabaseSettings() error {
	viper.Set("DB_HOST", db.Host)
	viper.Set("DB_PORT", db.Port)
	viper.Set("DB_NAME", db.Database)
	viper.Set("DB_USER", db.Username)
	viper.Set("DB_PASSWORD", db.Password)
	return nil
}
func (db *MSSQLConfig) DatabaseConnection() string {
	return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", db.Username, db.Password, db.Host, db.Port, db.Database)
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
	sqlDB, err := sql.Open("mssql", db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open MSSQL database: %w", err)
	}
	defer sqlDB.Close()

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

// LoadDatabaseSettings loads database settings based on the database type
func LoadDatabaseSettings() (DB, error) {
	dbType := viper.GetString("DB_TYPE")
	if dbType == "" {
		dbType = "sqlite3" // Default
	}

	var db DB
	switch dbType {
	case "sqlite3":
		db = &SQLiteConfig{}
	case "postgres":
		db = &PostgreSQLConfig{}
	case "mssql":
		db = &MSSQLConfig{}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	if err := viper.Unmarshal(db); err != nil {
		return nil, fmt.Errorf("error unmarshalling database config: %w", err)
	}

	// Run GetDatabaseSettings to set defaults if values are missing
	if err := db.GetDatabaseSettings(); err != nil {
		return nil, err
	}

	return db, nil
}

func RequiredTables() []string {
	return []string{
		"Accounts",
		"AccountCheckins",
		"AccountLocations",
		"AccountsPendingChanges",
		"Routes",
		"RouteWaypoints",
		"UserProfiles",
		"DataSets",
		"DataSetValues",
	}
}

func createCommandForTable(tableName string) string {
	return "create_" + strings.ToLower(tableName) + "_table"
}

func sqlCommandLoader(dbType, command string) string {
	path := fmt.Sprintf("database/%s/%s.sql", dbType, command)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func RunCommand(db DB, command string, args ...any) error {
	sqlText := sqlCommandLoader(db.GetType(), command)
	if sqlText == "" {
		return fmt.Errorf("unknown or unavailable SQL command: %s", command)
	}
	sqlDB, err := sql.Open(db.GetType(), db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()
	_, err = sqlDB.Exec(sqlText, args...)
	return err
}