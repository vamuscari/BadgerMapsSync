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

	"github.com/fatih/color"
	_ "github.com/lib/pq"               // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"     // SQLite driver
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
	"github.com/spf13/viper"
)

//go:embed mssql/*.sql
var mssqlFS embed.FS

//go:embed postgres/*.sql
var postgresFS embed.FS

//go:embed sqlite3/*.sql
var sqlite3FS embed.FS

type DB interface {
	GetType() string
	DatabaseConnection() string
	LoadConfig() error
	GetUsername() string
	SaveConfig() error
	PromptDatabaseSettings()
	TableExists(tableName string) (bool, error)
	ViewExists(viewName string) (bool, error)
	ProcedureExists(procedureName string) (bool, error)
	TriggerExists(triggerName string) (bool, error)
	GetTableColumns(tableName string) ([]string, error)
	ValidateSchema() error
	EnforceSchema() error
	TestConnection() error
	DropAllTables() error
	Connect() error
	Close() error
	GetDB() *sql.DB
	GetSQL(command string) string
}

// SQLiteConfig represents a SQLite database configuration
type SQLiteConfig struct {
	state *state.State
	db    *sql.DB
	Path  string `mapstructure:"DB_PATH"`
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
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	return nil
}

func (db *SQLiteConfig) Close() error {
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
	query := db.GetSQL("get_table_columns")

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

func (db *SQLiteConfig) EnforceSchema() error {
	sqlDB := db.GetDB()

	for _, tableName := range RequiredTables() {
		if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
			fmt.Printf("Creating table: %s... ", tableName)
		}
		createCmd := createCommandForTable(tableName)
		sqlText := db.GetSQL(createCmd)
		if sqlText == "" {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			err := fmt.Errorf("failed to load SQL command '%s' for database type '%s'", createCmd, db.GetType())
			if db.state.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: %v\n", err)
			}
			return err
		}
		if _, err := sqlDB.Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			if db.state.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: SQL execution error for table %s: %v\n", tableName, err)
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}

	// Insert initial data for FieldMaps
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Inserting initial data for FieldMaps... ")
	}
	sqlText := db.GetSQL("insert_field_maps")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for FieldMaps: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Insert initial data for Configurations
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Inserting initial data for Configurations... ")
	}
	sqlText = db.GetSQL("insert_configurations")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for Configurations: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create view
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating view: AccountsWithLabels... ")
	}
	sqlText = db.GetSQL("create_accounts_with_labels_view")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create view AccountsWithLabels: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}

func (db *SQLiteConfig) TestConnection() error {
	return db.GetDB().Ping()
}

func (db *SQLiteConfig) ValidateSchema() error {
	expectedSchema := GetExpectedSchema()
	for _, tableName := range RequiredTables() {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Printf("Checking table: %s... ", tableName)
		}
		exists, err := db.TableExists(tableName)
		if err != nil {
			if db.state.Verbose && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			if db.state.Debug {
				fmt.Printf("ValidateSchema error: %v\n", err)
			}
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			if db.state.Verbose && !db.state.Quiet {
				fmt.Println(color.RedString("MISSING"))
			}
			return fmt.Errorf("required table %s does not exist", tableName)
		}
		if db.state.Verbose && !db.state.Quiet {
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

	if db.state.Verbose && !db.state.Quiet {
		fmt.Printf("Checking view: AccountsWithLabels... ")
	}
	exists, err := db.ViewExists("AccountsWithLabels")
	if err != nil {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		if db.state.Debug {
			fmt.Printf("ValidateSchema error: %v\n", err)
		}
		return fmt.Errorf("error checking if view AccountsWithLabels exists: %w", err)
	}
	if !exists {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required view AccountsWithLabels does not exist")
	}
	if db.state.Verbose && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}

func (db *SQLiteConfig) TableExists(tableName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("check_table_exists")
	var count int
	err := sqlDB.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		if db.state.Debug {
			fmt.Printf("TableExists error: %v\n", err)
		}
		return false, err
	}
	return count > 0, nil
}

func (db *SQLiteConfig) ViewExists(viewName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("check_view_exists")
	var count int
	err := sqlDB.QueryRow(query, viewName).Scan(&count)
	if err != nil {
		if db.state.Debug {
			fmt.Printf("ViewExists error: %v\n", err)
		}
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

func (db *SQLiteConfig) LoadConfig() error {
	if db.Path == "" {
		db.Path = utils.GetConfigDirFile("badgermaps.db")
	}
	return nil
}

func (db *SQLiteConfig) GetUsername() string {
	return ""
}

func (db *SQLiteConfig) SaveConfig() error {
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
	sqlDB := db.GetDB()
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
	state    *state.State
	db       *sql.DB
	Host     string `mapstructure:"DB_HOST"`
	Port     int    `mapstructure:"DB_PORT"`
	Database string `mapstructure:"DB_NAME"`
	Username string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	SSLMode  string `mapstructure:"DB_SSL_MODE"`
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
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	return nil
}

func (db *PostgreSQLConfig) Close() error {
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
	query := db.GetSQL("get_table_columns")

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

func (db *PostgreSQLConfig) EnforceSchema() error {
	sqlDB := db.GetDB()

	for _, tableName := range RequiredTables() {
		if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
			fmt.Printf("Creating table: %s... ", tableName)
		}
		createCmd := createCommandForTable(tableName)
		sqlText := db.GetSQL(createCmd)
		if sqlText == "" {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			err := fmt.Errorf("failed to load SQL command '%s' for database type '%s'", createCmd, db.GetType())
			if db.state.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: %v\n", err)
			}
			return err
		}
		if _, err := sqlDB.Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			if db.state.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: SQL execution error for table %s: %v\n", tableName, err)
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}

	// Insert initial data for FieldMaps
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Inserting initial data for FieldMaps... ")
	}
	sqlText := db.GetSQL("insert_field_maps")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for FieldMaps: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Insert initial data for Configurations
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Inserting initial data for Configurations... ")
	}
	sqlText = db.GetSQL("insert_configurations")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for Configurations: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create function
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating function: AccountsWithLabelsView... ")
	}
	sqlText = db.GetSQL("create_accounts_with_labels_view")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create function AccountsWithLabelsView: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Call function to create view
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating view: AccountsWithLabels... ")
	}
	if _, err := db.GetDB().Exec("SELECT AccountsWithLabelsView()"); err != nil {
		if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to execute AccountsWithLabelsView function: %w", err)
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create trigger
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating trigger: datasets_update_trigger... ")
	}
	sqlText = db.GetSQL("create_datasets_update_trigger")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create trigger datasets_update_trigger: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create function to update field mappings
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating function: update_field_mappings_from_datasets... ")
	}
	sqlText = db.GetSQL("update_field_mappings_from_datasets")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create function update_field_mappings_from_datasets: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create trigger to update field mappings
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating trigger: datasets_field_mappings_update_trigger... ")
	}
	sqlText = db.GetSQL("create_field_mappings_update_trigger")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create trigger datasets_field_mappings_update_trigger: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}
func (db *PostgreSQLConfig) TestConnection() error {
	return db.GetDB().Ping()
}
func (db *PostgreSQLConfig) ValidateSchema() error {
	expectedSchema := GetExpectedSchema()
	for _, tableName := range RequiredTables() {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Printf("Checking table: %s... ", tableName)
		}
		exists, err := db.TableExists(tableName)
		if err != nil {
			if db.state.Verbose && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			if db.state.Debug {
				fmt.Printf("ValidateSchema error: %v\n", err)
			}
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			if db.state.Verbose && !db.state.Quiet {
				fmt.Println(color.RedString("MISSING"))
			}
			return fmt.Errorf("required table %s does not exist", tableName)
		}
		if db.state.Verbose && !db.state.Quiet {
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

	if db.state.Verbose && !db.state.Quiet {
		fmt.Printf("Checking view: AccountsWithLabels... ")
	}
	viewExists, err := db.ViewExists("AccountsWithLabels")
	if err != nil {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		if db.state.Debug {
			fmt.Printf("ValidateSchema error: %v\n", err)
		}
		return fmt.Errorf("error checking if view AccountsWithLabels exists: %w", err)
	}
	if !viewExists {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required view AccountsWithLabels does not exist")
	}
	if db.state.Verbose && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	if db.state.Verbose && !db.state.Quiet {
		fmt.Printf("Checking function: update_field_maps_from_datasets... ")
	}
	procExists, err := db.ProcedureExists("update_field_maps_from_datasets")
	if err != nil {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		if db.state.Debug {
			fmt.Printf("ValidateSchema error: %v\n", err)
		}
		return fmt.Errorf("error checking if function update_field_maps_from_datasets exists: %w", err)
	}
	if !procExists {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required function update_field_maps_from_datasets does not exist")
	}
	if db.state.Verbose && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	if db.state.Verbose && !db.state.Quiet {
		fmt.Printf("Checking trigger: datasets_update_trigger... ")
	}
	triggerExists, err := db.TriggerExists("datasets_update_trigger")
	if err != nil {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		if db.state.Debug {
			fmt.Printf("ValidateSchema error: %v\n", err)
		}
		return fmt.Errorf("error checking if trigger datasets_update_trigger exists: %w", err)
	}
	if !triggerExists {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required trigger datasets_update_trigger does not exist")
	}
	if db.state.Verbose && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}
func (db *PostgreSQLConfig) TableExists(tableName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("check_table_exists")
	var count int
	err := sqlDB.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		if db.state.Debug {
			fmt.Printf("TableExists error: %v\n", err)
		}
		return false, err
	}
	return count > 0, nil
}

func (db *PostgreSQLConfig) ViewExists(viewName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("check_view_exists")
	var count int
	err := sqlDB.QueryRow(query, viewName).Scan(&count)
	if err != nil {
		if db.state.Debug {
			fmt.Printf("ViewExists error: %v\n", err)
		}
		return false, err
	}
	return count > 0, nil
}

func (db *PostgreSQLConfig) ProcedureExists(procedureName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("check_procedure_exists")
	var count int
	err := sqlDB.QueryRow(query, procedureName).Scan(&count)
	if err != nil {
		if db.state.Debug {
			fmt.Printf("ProcedureExists error: %v\n", err)
		}
		return false, err
	}
	return count > 0, nil
}

func (db *PostgreSQLConfig) TriggerExists(triggerName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("check_trigger_exists")
	var count int
	err := sqlDB.QueryRow(query, triggerName).Scan(&count)
	if err != nil {
		if db.state.Debug {
			fmt.Printf("TriggerExists error: %v\n", err)
		}
		return false, err
	}
	return count > 0, nil
}
func (db *PostgreSQLConfig) GetType() string {
	return "postgres"
}
func (db *PostgreSQLConfig) LoadConfig() error {
	db.Host = viper.GetString("DB_HOST")
	db.Port = viper.GetInt("DB_PORT")
	db.Database = viper.GetString("DB_NAME")
	db.Username = viper.GetString("DB_USER")
	db.Password = viper.GetString("DB_PASSWORD")
	db.SSLMode = viper.GetString("DB_SSL_MODE")
	return nil
}

func (db *PostgreSQLConfig) GetUsername() string {
	return db.Username
}
func (db *PostgreSQLConfig) SaveConfig() error {
	viper.Set("DB_HOST", db.Host)
	viper.Set("DB_PORT", db.Port)
	viper.Set("DB_NAME", db.Database)
	viper.Set("DB_USER", db.Username)
	viper.Set("DB_PASSWORD", db.Password)
	viper.Set("DB_SSL_MODE", db.SSLMode)
	return nil
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

// MSSQLConfig represents a Microsoft SQL Server database configuration
type MSSQLConfig struct {
	state    *state.State
	db       *sql.DB
	Host     string `mapstructure:"DB_HOST"`
	Port     int    `mapstructure:"DB_PORT"`
	Database string `mapstructure:"DB_NAME"`
	Username string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
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
		return fmt.Errorf("failed to open MSSQL database: %w", err)
	}
	return nil
}

func (db *MSSQLConfig) Close() error {
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
	query := db.GetSQL("get_table_columns")

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

func (db *MSSQLConfig) EnforceSchema() error {
	sqlDB := db.GetDB()

	for _, tableName := range RequiredTables() {
		if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
			fmt.Printf("Creating table: %s... ", tableName)
		}
		createCmd := createCommandForTable(tableName)
		sqlText := db.GetSQL(createCmd)
		if sqlText == "" {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			err := fmt.Errorf("failed to load SQL command '%s' for database type '%s'", createCmd, db.GetType())
			if db.state.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: %v\n", err)
			}
			return err
		}
		if _, err := sqlDB.Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			if db.state.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: SQL execution error for table %s: %v\n", tableName, err)
			}
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
			fmt.Println(color.GreenString("OK"))
		}
	}

	// Insert initial data for FieldMaps
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Inserting initial data for FieldMaps... ")
	}
	sqlText := db.GetSQL("insert_field_maps")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for FieldMaps: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Insert initial data for Configurations
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Inserting initial data for Configurations... ")
	}
	sqlText = db.GetSQL("insert_configurations")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to insert initial data for Configurations: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create procedure
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating procedure: AccountsWithLabelsView... ")
	}
	sqlText = db.GetSQL("create_accounts_with_labels_view")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create procedure AccountsWithLabelsView: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Call procedure to create view
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating view: AccountsWithLabels... ")
	}
	if _, err := db.GetDB().Exec("EXEC AccountsWithLabelsView"); err != nil {
		if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		return fmt.Errorf("failed to execute AccountsWithLabelsView procedure: %w", err)
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create trigger
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating trigger: datasets_update_trigger... ")
	}
	sqlText = db.GetSQL("create_datasets_update_trigger")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create trigger datasets_update_trigger: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create procedure to update field maps
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating procedure: update_field_maps_from_datasets... ")
	}
	sqlText = db.GetSQL("update_field_maps_from_datasets")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create procedure update_field_maps_from_datasets: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	// Create trigger to update field maps
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Printf("Creating trigger: datasets_field_maps_update_trigger... ")
	}
	sqlText = db.GetSQL("create_field_maps_update_trigger")
	if sqlText != "" {
		if _, err := db.GetDB().Exec(sqlText); err != nil {
			if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			return fmt.Errorf("failed to create trigger datasets_field_maps_update_trigger: %w", err)
		}
	}
	if (db.state.Verbose || db.state.Debug) && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}
func (db *MSSQLConfig) TestConnection() error {
	return db.GetDB().Ping()
}
func (db *MSSQLConfig) ValidateSchema() error {
	expectedSchema := GetExpectedSchema()
	for _, tableName := range RequiredTables() {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Printf("Checking table: %s... ", tableName)
		}
		exists, err := db.TableExists(tableName)
		if err != nil {
			if db.state.Verbose && !db.state.Quiet {
				fmt.Println(color.RedString("ERROR"))
			}
			if db.state.Debug {
				fmt.Printf("ValidateSchema error: %v\n", err)
			}
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			if db.state.Verbose && !db.state.Quiet {
				fmt.Println(color.RedString("MISSING"))
			}
			return fmt.Errorf("required table %s does not exist", tableName)
		}
		if db.state.Verbose && !db.state.Quiet {
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

	if db.state.Verbose && !db.state.Quiet {
		fmt.Printf("Checking view: AccountsWithLabels... ")
	}
	viewExists, err := db.ViewExists("AccountsWithLabels")
	if err != nil {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		if db.state.Debug {
			fmt.Printf("ValidateSchema error: %v\n", err)
		}
		return fmt.Errorf("error checking if view AccountsWithLabels exists: %w", err)
	}
	if !viewExists {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required view AccountsWithLabels does not exist")
	}
	if db.state.Verbose && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	if db.state.Verbose && !db.state.Quiet {
		fmt.Printf("Checking procedure: update_field_maps_from_datasets... ")
	}
	procExists, err := db.ProcedureExists("update_field_maps_from_datasets")
	if err != nil {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		if db.state.Debug {
			fmt.Printf("ValidateSchema error: %v\n", err)
		}
		return fmt.Errorf("error checking if procedure update_field_maps_from_datasets exists: %w", err)
	}
	if !procExists {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required procedure update_field_maps_from_datasets does not exist")
	}
	if db.state.Verbose && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	if db.state.Verbose && !db.state.Quiet {
		fmt.Printf("Checking trigger: datasets_update_trigger... ")
	}
	triggerExists, err := db.TriggerExists("datasets_update_trigger")
	if err != nil {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("ERROR"))
		}
		if db.state.Debug {
			fmt.Printf("ValidateSchema error: %v\n", err)
		}
		return fmt.Errorf("error checking if trigger datasets_update_trigger exists: %w", err)
	}
	if !triggerExists {
		if db.state.Verbose && !db.state.Quiet {
			fmt.Println(color.RedString("MISSING"))
		}
		return fmt.Errorf("required trigger datasets_update_trigger does not exist")
	}
	if db.state.Verbose && !db.state.Quiet {
		fmt.Println(color.GreenString("OK"))
	}

	return nil
}
func (db *MSSQLConfig) TableExists(tableName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("check_table_exists")
	if db.state.Debug {
		fmt.Printf("DEBUG: Executing query: %s with arg: %s\n", query, tableName)
	}
	var count int
	err := sqlDB.QueryRow(query, tableName).Scan(&count)

	if err != nil {
		if db.state.Debug {
			fmt.Printf("DEBUG: TableExists SQL ERROR: %v\n", err)
		}
		return false, err
	}
	if db.state.Debug {
		fmt.Printf("DEBUG: TableExists count for %s: %d\n", tableName, count)
	}
	return count > 0, nil
}

func (db *MSSQLConfig) ViewExists(viewName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("check_view_exists")
	var count int
	err := sqlDB.QueryRow(query, viewName).Scan(&count)
	if err != nil {
		if db.state.Debug {
			fmt.Printf("ViewExists error: %v\n", err)
		}
		return false, err
	}
	return count > 0, nil
}

func (db *MSSQLConfig) ProcedureExists(procedureName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("check_procedure_exists")
	var count int
	err := sqlDB.QueryRow(query, procedureName).Scan(&count)
	if err != nil {
		if db.state.Debug {
			fmt.Printf("ProcedureExists error: %v\n", err)
		}
		return false, err
	}
	return count > 0, nil
}

func (db *MSSQLConfig) TriggerExists(triggerName string) (bool, error) {
	sqlDB := db.GetDB()
	query := db.GetSQL("check_trigger_exists")
	var count int
	err := sqlDB.QueryRow(query, triggerName).Scan(&count)
	if err != nil {
		if db.state.Debug {
			fmt.Printf("TriggerExists error: %v\n", err)
		}
		return false, err
	}
	return count > 0, nil
}

func (db *MSSQLConfig) GetType() string {
	return "mssql"
}
func (db *MSSQLConfig) LoadConfig() error {
	db.Host = viper.GetString("DB_HOST")
	db.Port = viper.GetInt("DB_PORT")
	db.Database = viper.GetString("DB_NAME")
	db.Username = viper.GetString("DB_USER")
	db.Password = viper.GetString("DB_PASSWORD")
	return nil
}

func (db *MSSQLConfig) GetUsername() string {
	return db.Username
}
func (db *MSSQLConfig) SaveConfig() error {
	viper.Set("DB_HOST", db.Host)
	viper.Set("DB_PORT", db.Port)
	viper.Set("DB_NAME", db.Database)
	viper.Set("DB_USER", db.Username)
	viper.Set("DB_PASSWORD", db.Password)
	return nil
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

// NewDBFromConfig creates a new DB instance from a config struct.
func NewDB(dbType string, s *state.State) (DB, error) {
	var db DB
	switch dbType {
	case "sqlite3":
		db = &SQLiteConfig{
			state: s,
		}
	case "postgres":
		db = &PostgreSQLConfig{
			state: s,
		}
	case "mssql":
		db = &MSSQLConfig{
			state: s,
		}
	default:
		db = &SQLiteConfig{
			state: s,
		}
	}

	db.LoadConfig()

	if err := db.Connect(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return db, nil
}

// LoadDatabaseSettings loads database settings based on the database type
func LoadDatabaseSettings(s *state.State) (DB, error) {
	dbType := viper.Get("DB_Type")

	if dbType == "" {
		dbType = "sqlite3" // Default
	}

	var db DB
	switch dbType {
	case "sqlite3":
		db = &SQLiteConfig{
			state: s,
		}
	case "postgres":
		db = &PostgreSQLConfig{
			state: s,
		}
	case "mssql":
		db = &MSSQLConfig{
			state: s,
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Run GetDatabaseSettings to set defaults if values are missing
	if err := db.LoadConfig(); err != nil {
		return nil, err
	}

	if err := db.Connect(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return db, nil
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
	}
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func createCommandForTable(tableName string) string {
	return "create_" + toSnakeCase(tableName) + "_table"
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
	return RunCommand(db, "update_configuration", value, key)
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