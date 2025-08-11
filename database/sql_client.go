package database

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"badgermapscli/api"

	_ "github.com/lib/pq"               // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"     // SQLite driver
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
)

// logf logs a message if verbose mode is enabled
func (c *Client) logf(format string, args ...interface{}) {
	if c.verbose {
		log.Printf(format, args...)
	}
}

// Config holds the database configuration
type Config struct {
	DatabaseType string
	Host         string
	Port         string
	Database     string
	Username     string
	Password     string
}

// Client represents the database client
type Client struct {
	config          *Config
	db              *sql.DB
	verbose         bool
	needsSchemaInit bool
}

// NewClient creates a new database client
func NewClient(config *Config, verbose bool) (*Client, error) {
	client := &Client{
		config:  config,
		verbose: verbose,
	}

	db, err := client.connectDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	client.db = db
	return client, nil
}

// connectDatabase establishes a database connection based on the configuration
func (c *Client) connectDatabase() (*sql.DB, error) {
	var dsn string
	var driverName string

	switch c.config.DatabaseType {
	case "postgres", "postgresql":
		driverName = "postgres"
		dsn = c.buildPostgresDSN()
	case "mssql", "sqlserver":
		driverName = "mssql"
		dsn = c.buildMSSQLDSN()
	case "sqlite3", "sqlite":
		driverName = "sqlite3"
		dsn = c.config.Database

		// Check if SQLite database file exists
		_, err := os.Stat(dsn)
		if os.IsNotExist(err) {
			fmt.Printf("SQLite database file '%s' does not exist. Would you like to create it? (y/n): ", dsn)
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("error reading input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return nil, fmt.Errorf("database creation cancelled by user")
			}

			// Create the database directory if it doesn't exist
			dbDir := filepath.Dir(dsn)
			if dbDir != "." {
				if err := os.MkdirAll(dbDir, 0755); err != nil {
					return nil, fmt.Errorf("failed to create database directory: %w", err)
				}
			}

			fmt.Printf("Creating new SQLite database at '%s'...\n", dsn)
			c.needsSchemaInit = true
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s (supported types: postgres, mssql, sqlite3)", c.config.DatabaseType)
	}

	c.logf("Connecting to %s database...", c.config.DatabaseType)
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s database connection: %w (check connection parameters)", c.config.DatabaseType, err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping %s database: %w (check if database server is running and accessible)", c.config.DatabaseType, err)
	}

	c.logf("Successfully connected to %s database", c.config.DatabaseType)

	// Initialize schema if needed (for new SQLite databases)
	if c.needsSchemaInit {
		c.db = db // Temporarily set db to initialize schema
		if err := c.InitializeSchema(); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to initialize database schema: %w", err)
		}
		c.db = nil // Reset it as it will be set by the caller
		fmt.Println("Database schema initialized successfully")
	}

	return db, nil
}

// buildPostgresDSN builds a PostgreSQL connection string
func (c *Client) buildPostgresDSN() string {
	if c.config.Host == "" {
		c.config.Host = "localhost"
	}
	if c.config.Port == "" {
		c.config.Port = "5432"
	}

	dsn := fmt.Sprintf("host=%s port=%s dbname=%s sslmode=disable",
		c.config.Host, c.config.Port, c.config.Database)

	if c.config.Username != "" {
		dsn += fmt.Sprintf(" user=%s", c.config.Username)
	}
	if c.config.Password != "" {
		dsn += fmt.Sprintf(" password=%s", c.config.Password)
	}

	return dsn
}

// buildMSSQLDSN builds a SQL Server connection string
func (c *Client) buildMSSQLDSN() string {
	if c.config.Host == "" {
		c.config.Host = "localhost"
	}
	if c.config.Port == "" {
		c.config.Port = "1433"
	}

	dsn := fmt.Sprintf("server=%s;port=%s;database=%s;encrypt=disable",
		c.config.Host, c.config.Port, c.config.Database)

	if c.config.Username != "" {
		dsn += fmt.Sprintf(";user id=%s", c.config.Username)
	}
	if c.config.Password != "" {
		dsn += fmt.Sprintf(";password=%s", c.config.Password)
	}

	return dsn
}

// Close closes the database connection
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// GetDB returns the underlying database connection
func (c *Client) GetDB() *sql.DB {
	return c.db
}

// InitializeSchema creates all necessary tables and indexes
func (c *Client) InitializeSchema() error {
	c.logf("Initializing database schema...")

	// Validate SQL files exist before proceeding
	sqlLoader := NewSQLLoader(c.config.DatabaseType, c.verbose)
	if err := sqlLoader.ValidateSQLFiles(); err != nil {
		return fmt.Errorf("SQL validation failed: %w (check if all required SQL files exist in database/%s directory)", err, c.config.DatabaseType)
	}

	// Create tables
	tables := []string{
		"accounts",
		"routes",
		"checkins",
		"user_profiles",
		"account_locations",
		"route_waypoints",
		"data_sets",
		"data_set_values",
	}

	for _, tableName := range tables {
		if err := c.createTableIfNotExists(tableName); err != nil {
			return fmt.Errorf("failed to create table %s: %w (check SQL syntax in create_%s_table.sql)", tableName, err, tableName)
		}
	}

	// Create indexes
	if err := c.createIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w (check SQL syntax in create_indexes.sql)", err)
	}

	c.logf("Database schema initialized successfully")
	return nil
}

// createTableIfNotExists creates a table if it doesn't exist
func (c *Client) createTableIfNotExists(tableName string) error {
	exists, err := c.tableExists(tableName)
	if err != nil {
		return fmt.Errorf("failed to check if table %s exists: %w", tableName, err)
	}

	if !exists {
		return c.createTable(tableName)
	}

	c.logf("Table %s already exists, skipping creation", tableName)
	return nil
}

// tableExists checks if a table exists in the database
func (c *Client) tableExists(tableName string) (bool, error) {
	var query string
	switch c.config.DatabaseType {
	case "postgres", "postgresql":
		query = "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)"
		var exists bool
		err := c.db.QueryRow(query, tableName).Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("failed to check if table '%s' exists in PostgreSQL: %w (check database connection and permissions)", tableName, err)
		}
		return exists, nil
	case "mssql", "sqlserver":
		query = "SELECT CASE WHEN EXISTS (SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = @p1) THEN 1 ELSE 0 END"
		var exists bool
		err := c.db.QueryRow(query, tableName).Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("failed to check if table '%s' exists in MSSQL: %w (check database connection and permissions)", tableName, err)
		}
		return exists, nil
	case "sqlite3", "sqlite":
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
		var name string
		err := c.db.QueryRow(query, tableName).Scan(&name)
		if err == sql.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to check if table '%s' exists in SQLite: %w (check database file permissions)", tableName, err)
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported database type: %s (supported types: postgres, mssql, sqlite3)", c.config.DatabaseType)
	}
}

// createTable creates a specific table based on the database type
func (c *Client) createTable(tableName string) error {
	c.logf("Creating table: %s", tableName)

	sqlLoader := NewSQLLoader(c.config.DatabaseType, c.verbose)
	sqlContent, err := sqlLoader.LoadCreateTableSQL(tableName)
	if err != nil {
		return fmt.Errorf("failed to load SQL for table %s: %w (check if create_%s_table.sql exists in database/%s directory)",
			tableName, err, tableName, c.config.DatabaseType)
	}

	_, err = c.db.Exec(sqlContent)
	if err != nil {
		return fmt.Errorf("failed to execute SQL for table %s: %w (check SQL syntax and database permissions)",
			tableName, err)
	}

	log.Printf("Successfully created table: %s", tableName)
	return nil
}

// ValidateDatabaseSchema checks if all required tables exist with the correct fields
func (c *Client) ValidateDatabaseSchema() error {
	c.logf("Validating database schema...")

	// Get required tables and their essential columns
	requiredTables := c.GetRequiredTables()

	// Check each table and its columns
	for tableName, columns := range requiredTables {
		// Check if table exists
		exists, err := c.tableExists(tableName)
		if err != nil {
			return fmt.Errorf("error checking if table %s exists: %w", tableName, err)
		}
		if !exists {
			return fmt.Errorf("required table %s does not exist. Run 'badgermaps utils create-tables' to create the necessary tables", tableName)
		}

		// Check if all required columns exist
		for _, column := range columns {
			exists, err := c.columnExists(tableName, column)
			if err != nil {
				return fmt.Errorf("error checking if column %s exists in table %s: %w", column, tableName, err)
			}
			if !exists {
				return fmt.Errorf("required column %s does not exist in table %s. Run 'badgermaps utils create-tables' to recreate the tables with the correct schema", column, tableName)
			}
		}
	}

	c.logf("Database schema validation successful")
	return nil
}

// columnExists checks if a column exists in a table
func (c *Client) columnExists(tableName string, columnName string) (bool, error) {
	sqlLoader := NewSQLLoader(c.config.DatabaseType, c.verbose)
	sqlContent, err := sqlLoader.LoadSQL("check_column_exists.sql")
	if err != nil {
		return false, fmt.Errorf("failed to load check_column_exists.sql: %w", err)
	}

	var count int
	err = c.db.QueryRow(sqlContent, tableName, columnName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if column %s exists in table %s: %w", columnName, tableName, err)
	}

	return count > 0, nil
}

// GetRequiredTables returns a map of required tables and their essential columns
func (c *Client) GetRequiredTables() map[string][]string {
	return map[string][]string{
		"Accounts":         {"Id", "FirstName", "LastName"},
		"Routes":           {"Id", "Name", "RouteDate"},
		"AccountCheckins":  {"Id", "CrmId", "AccountId", "Comments"},
		"UserProfiles":     {"ProfileId", "Email", "FirstName", "LastName"},
		"AccountLocations": {"Id", "AccountId", "Latitude", "Longitude"},
		"RouteWaypoints":   {"Id", "RouteId"},
		"DataSets":         {"Name", "ProfileId"},
		"DataSetValues":    {"DataSetName", "Value"},
	}
}

// createIndexes creates indexes for better performance
func (c *Client) createIndexes() error {
	c.logf("Creating database indexes...")

	// Use the SQLLoader's CreateIndexes method which properly handles multiple statements
	sqlLoader := NewSQLLoader(c.config.DatabaseType, c.verbose)
	if err := sqlLoader.CreateIndexes(c.db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	c.logf("Database indexes created successfully")
	return nil
}

// DropAllTables drops all tables from the database
func (c *Client) DropAllTables() error {
	c.logf("Dropping all tables...")

	tables := []string{
		"data_set_values",
		"data_sets",
		"route_waypoints",
		"account_locations",
		"checkins",
		"routes",
		"user_profiles",
		"accounts",
	}

	for _, tableName := range tables {
		if err := c.dropTable(tableName); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}

	c.logf("All tables dropped successfully")
	return nil
}

// dropTable drops a specific table
func (c *Client) dropTable(tableName string) error {
	exists, err := c.tableExists(tableName)
	if err != nil {
		return fmt.Errorf("failed to check if table %s exists: %w", tableName, err)
	}

	if !exists {
		c.logf("Table %s does not exist, skipping drop", tableName)
		return nil
	}

	query := fmt.Sprintf("DROP TABLE %s", tableName)
	_, err = c.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}

	c.logf("Successfully dropped table: %s", tableName)
	return nil
}

// StoreAccountLocations stores account locations in the database
func (c *Client) StoreAccountLocations(accountID int, locations []api.Location) error {
	if len(locations) == 0 {
		return nil
	}

	c.logf("Storing %d locations for account %d", len(locations), accountID)

	sqlLoader := NewSQLLoader(c.config.DatabaseType, c.verbose)

	// First, delete existing locations for this account
	deleteSQL, err := sqlLoader.LoadSQL("delete_account_locations.sql")
	if err != nil {
		return fmt.Errorf("failed to load delete_account_locations SQL: %w (check if delete_account_locations.sql exists in database/%s directory)",
			err, c.config.DatabaseType)
	}

	// Begin transaction
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w (check database connection and transaction support)", err)
	}
	defer tx.Rollback()

	// Delete existing locations
	_, err = tx.Exec(deleteSQL, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete existing locations for account %d: %w", accountID, err)
	}

	// Load insert SQL
	insertSQL, err := sqlLoader.LoadSQL("insert_account_locations.sql")
	if err != nil {
		return fmt.Errorf("failed to load insert_account_locations SQL: %w (check if insert_account_locations.sql exists in database/%s directory)",
			err, c.config.DatabaseType)
	}

	// Prepare statement for inserts
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert_account_locations statement: %w (check SQL syntax in insert_account_locations.sql)", err)
	}
	defer stmt.Close()

	// Execute for each location
	for _, location := range locations {
		// Handle nullable fields
		name := ""
		if location.Name != nil {
			name = *location.Name
		}

		_, err := stmt.Exec(
			location.ID,
			accountID,
			location.City,
			name,
			location.Zipcode,
			location.Long, // Maps to Longitude column in database
			location.State,
			location.Lat, // Maps to Latitude column in database
			location.AddressLine1,
			location.Location,
		)
		if err != nil {
			return fmt.Errorf("failed to insert location ID %d for account %d: %w (check location data and table schema compatibility)",
				location.ID, accountID, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w (check database connection and transaction state)", err)
	}

	return nil
}

// StoreAccounts stores accounts in the database using merge_accounts_detailed
func (c *Client) StoreAccounts(accounts []api.Account) error {
	c.logf("Storing %d accounts using merge_accounts_detailed", len(accounts))

	sqlLoader := NewSQLLoader(c.config.DatabaseType, c.verbose)
	sqlContent, err := sqlLoader.LoadMergeAccountsDetailedSQL()
	if err != nil {
		return fmt.Errorf("failed to load merge_accounts_detailed SQL: %w (check if merge_accounts_detailed.sql exists in database/%s directory)",
			err, c.config.DatabaseType)
	}

	// Begin transaction
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w (check database connection and transaction support)", err)
	}
	defer tx.Rollback()

	// Prepare statement
	stmt, err := tx.Prepare(sqlContent)
	if err != nil {
		return fmt.Errorf("failed to prepare merge_accounts_detailed statement: %w (check SQL syntax in merge_accounts_detailed.sql)", err)
	}
	defer stmt.Close()

	// Execute for each account
	for _, account := range accounts {
		// Handle nullable fields
		firstName := ""
		if account.FirstName != nil {
			firstName = *account.FirstName
		}

		customerID := ""
		if account.CustomerID != nil {
			customerID = *account.CustomerID
		}

		notes := ""
		if account.Notes != nil {
			notes = *account.Notes
		}

		crmID := ""
		if account.CRMID != nil {
			crmID = *account.CRMID
		}

		accountOwner := ""
		if account.AccountOwner != nil {
			accountOwner = *account.AccountOwner
		}

		lastCheckinDate := ""
		if account.LastCheckinDate != nil {
			lastCheckinDate = *account.LastCheckinDate
		}

		lastModifiedDate := ""
		if account.LastModifiedDate != nil {
			lastModifiedDate = *account.LastModifiedDate
		}

		followUpDate := ""
		if account.FollowUpDate != nil {
			followUpDate = *account.FollowUpDate
		}

		// Handle nullable custom fields
		var customNumeric, customNumeric2, customNumeric3, customNumeric4, customNumeric5 float64
		var customNumeric6, customNumeric7, customNumeric8, customNumeric9, customNumeric10 float64
		var customNumeric11, customNumeric12, customNumeric13, customNumeric14, customNumeric15 float64
		var customNumeric16, customNumeric17, customNumeric18, customNumeric19, customNumeric20 float64
		var customNumeric21, customNumeric22, customNumeric23, customNumeric24, customNumeric25 float64
		var customNumeric26, customNumeric27, customNumeric28, customNumeric29, customNumeric30 float64

		var customText, customText2, customText3, customText4, customText5 string
		var customText6, customText7, customText8, customText9, customText10 string
		var customText11, customText12, customText13, customText14, customText15 string
		var customText16, customText17, customText18, customText19, customText20 string
		var customText21, customText22, customText23, customText24, customText25 string
		var customText26, customText27, customText28, customText29, customText30 string

		if account.CustomNumeric != nil {
			customNumeric = *account.CustomNumeric
		}
		if account.CustomText != nil {
			customText = *account.CustomText
		}
		if account.CustomNumeric2 != nil {
			customNumeric2 = *account.CustomNumeric2
		}
		if account.CustomText2 != nil {
			customText2 = *account.CustomText2
		}
		if account.CustomNumeric3 != nil {
			customNumeric3 = *account.CustomNumeric3
		}
		if account.CustomText3 != nil {
			customText3 = *account.CustomText3
		}
		if account.CustomNumeric4 != nil {
			customNumeric4 = *account.CustomNumeric4
		}
		if account.CustomText4 != nil {
			customText4 = *account.CustomText4
		}
		if account.CustomNumeric5 != nil {
			customNumeric5 = *account.CustomNumeric5
		}
		if account.CustomText5 != nil {
			customText5 = *account.CustomText5
		}
		if account.CustomNumeric6 != nil {
			customNumeric6 = *account.CustomNumeric6
		}
		if account.CustomText6 != nil {
			customText6 = *account.CustomText6
		}
		if account.CustomNumeric7 != nil {
			customNumeric7 = *account.CustomNumeric7
		}
		if account.CustomText7 != nil {
			customText7 = *account.CustomText7
		}
		if account.CustomNumeric8 != nil {
			customNumeric8 = *account.CustomNumeric8
		}
		if account.CustomText8 != nil {
			customText8 = *account.CustomText8
		}
		if account.CustomNumeric9 != nil {
			customNumeric9 = *account.CustomNumeric9
		}
		if account.CustomText9 != nil {
			customText9 = *account.CustomText9
		}
		if account.CustomNumeric10 != nil {
			customNumeric10 = *account.CustomNumeric10
		}
		if account.CustomText10 != nil {
			customText10 = *account.CustomText10
		}
		if account.CustomNumeric11 != nil {
			customNumeric11 = *account.CustomNumeric11
		}
		if account.CustomText11 != nil {
			customText11 = *account.CustomText11
		}
		if account.CustomNumeric12 != nil {
			customNumeric12 = *account.CustomNumeric12
		}
		if account.CustomText12 != nil {
			customText12 = *account.CustomText12
		}
		if account.CustomNumeric13 != nil {
			customNumeric13 = *account.CustomNumeric13
		}
		if account.CustomText13 != nil {
			customText13 = *account.CustomText13
		}
		if account.CustomNumeric14 != nil {
			customNumeric14 = *account.CustomNumeric14
		}
		if account.CustomText14 != nil {
			customText14 = *account.CustomText14
		}
		if account.CustomNumeric15 != nil {
			customNumeric15 = *account.CustomNumeric15
		}
		if account.CustomText15 != nil {
			customText15 = *account.CustomText15
		}
		if account.CustomNumeric16 != nil {
			customNumeric16 = *account.CustomNumeric16
		}
		if account.CustomText16 != nil {
			customText16 = *account.CustomText16
		}
		if account.CustomNumeric17 != nil {
			customNumeric17 = *account.CustomNumeric17
		}
		if account.CustomText17 != nil {
			customText17 = *account.CustomText17
		}
		if account.CustomNumeric18 != nil {
			customNumeric18 = *account.CustomNumeric18
		}
		if account.CustomText18 != nil {
			customText18 = *account.CustomText18
		}
		if account.CustomNumeric19 != nil {
			customNumeric19 = *account.CustomNumeric19
		}
		if account.CustomText19 != nil {
			customText19 = *account.CustomText19
		}
		if account.CustomNumeric20 != nil {
			customNumeric20 = *account.CustomNumeric20
		}
		if account.CustomText20 != nil {
			customText20 = *account.CustomText20
		}
		if account.CustomNumeric21 != nil {
			customNumeric21 = *account.CustomNumeric21
		}
		if account.CustomText21 != nil {
			customText21 = *account.CustomText21
		}
		if account.CustomNumeric22 != nil {
			customNumeric22 = *account.CustomNumeric22
		}
		if account.CustomText22 != nil {
			customText22 = *account.CustomText22
		}
		if account.CustomNumeric23 != nil {
			customNumeric23 = *account.CustomNumeric23
		}
		if account.CustomText23 != nil {
			customText23 = *account.CustomText23
		}
		if account.CustomNumeric24 != nil {
			customNumeric24 = *account.CustomNumeric24
		}
		if account.CustomText24 != nil {
			customText24 = *account.CustomText24
		}
		if account.CustomNumeric25 != nil {
			customNumeric25 = *account.CustomNumeric25
		}
		if account.CustomText25 != nil {
			customText25 = *account.CustomText25
		}
		if account.CustomNumeric26 != nil {
			customNumeric26 = *account.CustomNumeric26
		}
		if account.CustomText26 != nil {
			customText26 = *account.CustomText26
		}
		if account.CustomNumeric27 != nil {
			customNumeric27 = *account.CustomNumeric27
		}
		if account.CustomText27 != nil {
			customText27 = *account.CustomText27
		}
		if account.CustomNumeric28 != nil {
			customNumeric28 = *account.CustomNumeric28
		}
		if account.CustomText28 != nil {
			customText28 = *account.CustomText28
		}
		if account.CustomNumeric29 != nil {
			customNumeric29 = *account.CustomNumeric29
		}
		if account.CustomText29 != nil {
			customText29 = *account.CustomText29
		}
		if account.CustomNumeric30 != nil {
			customNumeric30 = *account.CustomNumeric30
		}
		if account.CustomText30 != nil {
			customText30 = *account.CustomText30
		}

		_, err := stmt.Exec(
			account.ID,
			firstName,
			account.LastName,
			account.FullName,
			account.PhoneNumber,
			account.Email,
			accountOwner,
			customerID,
			notes,
			account.OriginalAddress,
			crmID,
			account.DaysSinceLastCheckin,
			followUpDate,
			lastCheckinDate,
			lastModifiedDate,
			customNumeric,
			customText,
			customNumeric2,
			customText2,
			customNumeric3,
			customText3,
			customNumeric4,
			customText4,
			customNumeric5,
			customText5,
			customNumeric6,
			customText6,
			customNumeric7,
			customText7,
			customNumeric8,
			customText8,
			customNumeric9,
			customText9,
			customNumeric10,
			customText10,
			customNumeric11,
			customText11,
			customNumeric12,
			customText12,
			customNumeric13,
			customText13,
			customNumeric14,
			customText14,
			customNumeric15,
			customText15,
			customNumeric16,
			customText16,
			customNumeric17,
			customText17,
			customNumeric18,
			customText18,
			customNumeric19,
			customText19,
			customNumeric20,
			customText20,
			customNumeric21,
			customText21,
			customNumeric22,
			customText22,
			customNumeric23,
			customText23,
			customNumeric24,
			customText24,
			customNumeric25,
			customText25,
			customNumeric26,
			customText26,
			customNumeric27,
			customText27,
			customNumeric28,
			customText28,
			customNumeric29,
			customText29,
			customNumeric30,
			customText30,
		)
		if err != nil {
			return fmt.Errorf("failed to merge account ID %d: %w (check account data and table schema compatibility)",
				account.ID, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w (check database connection and transaction state)", err)
	}

	// Store account locations for each account
	for _, account := range accounts {
		if len(account.Locations) > 0 {
			err := c.StoreAccountLocations(account.ID, account.Locations)
			if err != nil {
				return fmt.Errorf("failed to store locations for account %d: %w", account.ID, err)
			}
		}
	}

	// log.Printf("Successfully stored %d accounts using merge_accounts_detailed", len(accounts))
	return nil
}

// GetAccountIDs retrieves all account IDs from the database
func (c *Client) GetAccountIDs() ([]int, error) {
	c.logf("Retrieving all account IDs from database")

	query := "SELECT Id FROM Accounts ORDER BY Id"
	rows, err := c.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query account IDs: %w", err)
	}
	defer rows.Close()

	var accountIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan account ID: %w", err)
		}
		accountIDs = append(accountIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account IDs: %w", err)
	}

	c.logf("Retrieved %d account IDs from database", len(accountIDs))
	return accountIDs, nil
}

// StoreProfiles stores user profiles in the database using merge_user_profiles
func (c *Client) StoreProfiles(profile *api.UserProfile) error {
	c.logf("Storing user profile using merge_user_profiles")

	sqlLoader := NewSQLLoader(c.config.DatabaseType, c.verbose)
	sqlContent, err := sqlLoader.LoadMergeUserProfilesSQL()
	if err != nil {
		return fmt.Errorf("failed to load merge_user_profiles SQL: %w (check if merge_user_profiles.sql exists in database/%s directory)",
			err, c.config.DatabaseType)
	}

	// Begin transaction
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w (check database connection and transaction support)", err)
	}
	defer tx.Rollback()

	// Prepare statement
	stmt, err := tx.Prepare(sqlContent)
	if err != nil {
		return fmt.Errorf("failed to prepare merge_user_profiles statement: %w (check SQL syntax in merge_user_profiles.sql)", err)
	}
	defer stmt.Close()

	// Handle nullable manager field
	var manager string
	if profile.Manager != nil {
		manager = *profile.Manager
	}

	// Execute for the profile
	_, err = stmt.Exec(
		profile.ID,
		profile.FirstName,
		profile.LastName,
		profile.Email,
		profile.IsManager,
		manager,
		profile.Company.ID,
		profile.Company.Name,
		profile.Company.ShortName,
		profile.Completed,
		profile.TrialDaysLeft,
		profile.HasData,
		profile.DefaultApptLength,
	)
	if err != nil {
		return fmt.Errorf("failed to merge user profile ID %d: %w (check profile data and table schema compatibility)",
			profile.ID, err)
	}

	// Store datasets and dataset values
	if len(profile.Datafields) > 0 {
		c.logf("Storing %d datasets for profile ID %d", len(profile.Datafields), profile.ID)
		err = c.storeDataSets(tx, profile.ID, profile.Datafields)
		if err != nil {
			return fmt.Errorf("failed to store datasets: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w (check database connection and transaction state)", err)
	}

	c.logf("Successfully stored user profile using merge_user_profiles")
	return nil
}

// storeDataSets stores datasets and their values in the database
func (c *Client) storeDataSets(tx *sql.Tx, profileID int, datafields []api.DataField) error {
	// Load SQL for inserting datasets
	sqlLoader := NewSQLLoader(c.config.DatabaseType, c.verbose)

	deleteDataSetValuesSQL, err := sqlLoader.LoadDeleteDataSetValuesSQL()
	if err != nil {
		return fmt.Errorf("failed to load delete_data_set_values SQL: %w", err)
	}

	// First, delete existing datasets and values for this profile
	deleteDataSetsSQL, err := sqlLoader.LoadDeleteDataSetsSQL()
	if err != nil {
		return fmt.Errorf("failed to load delete_data_sets SQL: %w", err)
	}

	// Delete existing dataset values
	_, err = tx.Exec(deleteDataSetValuesSQL, profileID)
	if err != nil {
		return fmt.Errorf("failed to delete existing dataset values: %w", err)
	}

	// Delete existing datasets
	_, err = tx.Exec(deleteDataSetsSQL, profileID)
	if err != nil {
		return fmt.Errorf("failed to delete existing datasets: %w", err)
	}

	// Load SQL for inserting datasets and values
	insertDataSetsSQL, err := sqlLoader.LoadInsertDataSetsSQL()
	if err != nil {
		return fmt.Errorf("failed to load insert_data_sets SQL: %w", err)
	}

	insertDataSetValuesSQL, err := sqlLoader.LoadInsertDataSetValuesSQL()
	if err != nil {
		return fmt.Errorf("failed to load insert_data_set_values SQL: %w", err)
	}

	// Prepare statements
	insertDataSetsStmt, err := tx.Prepare(insertDataSetsSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert_data_sets statement: %w", err)
	}
	defer insertDataSetsStmt.Close()

	insertDataSetValuesStmt, err := tx.Prepare(insertDataSetValuesSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert_data_set_values statement: %w", err)
	}
	defer insertDataSetValuesStmt.Close()

	// Collect errors in an array
	var errors []error

	// Insert datasets and their values
	for _, datafield := range datafields {
		// Insert dataset
		_, err = insertDataSetsStmt.Exec(
			datafield.Name,
			profileID,
			datafield.Label,
			datafield.Type,
			datafield.Filterable,
			datafield.Position,
			datafield.HasData,
		)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to insert dataset %s: %w", datafield.Name, err))
			continue
		}

		// Insert dataset values
		for _, value := range datafield.Values {
			// Convert value to string
			var valueStr string
			if value.Value != nil {
				valueStr = fmt.Sprintf("%v", value.Value)
			}

			_, err = insertDataSetValuesStmt.Exec(
				profileID,
				datafield.Name,
				datafield.Position,
				valueStr,
				value.Text,
			)
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to insert dataset value for %s: %w", datafield.Name, err))
				continue
			}
		}
	}

	// Print all errors after processing
	if len(errors) > 0 {
		c.logf("Errors occurred while storing datasets:")
		for _, err := range errors {
			log.Println(err)
		}
		return fmt.Errorf("failed to store %d datasets or values", len(errors))
	}

	return nil
}

// StoreCheckins stores checkins in the database using merge_account_checkins
func (c *Client) StoreCheckins(checkins []api.Checkin) error {
	c.logf("Storing %d checkins using merge_account_checkins", len(checkins))

	sqlLoader := NewSQLLoader(c.config.DatabaseType)
	sqlContent, err := sqlLoader.LoadMergeAccountCheckinsSQL()
	if err != nil {
		return fmt.Errorf("failed to load merge_account_checkins SQL: %w (check if merge_account_checkins.sql exists in database/%s directory)",
			err, c.config.DatabaseType)
	}

	// Begin transaction
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w (check database connection and transaction support)", err)
	}
	defer tx.Rollback()

	// Prepare statement
	stmt, err := tx.Prepare(sqlContent)
	if err != nil {
		return fmt.Errorf("failed to prepare merge_account_checkins statement: %w (check SQL syntax in merge_account_checkins.sql)", err)
	}
	defer stmt.Close()

	// Execute for each checkin
	for _, checkin := range checkins {
		// Handle nullable CRMID field
		var crmID string
		if checkin.CRMID != nil {
			crmID = *checkin.CRMID
		}

		// Handle nullable ExtraFields field
		var extraFields string
		if checkin.ExtraFields != nil {
			extraFields = *checkin.ExtraFields
		}

		_, err := stmt.Exec(
			checkin.ID,
			crmID,
			checkin.Customer,
			checkin.LogDatetime,
			checkin.Type,
			checkin.Comments,
			extraFields,
			checkin.CreatedBy,
		)
		if err != nil {
			return fmt.Errorf("failed to merge checkin ID %d: %w (check checkin data and table schema compatibility)",
				checkin.ID, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w (check database connection and transaction state)", err)
	}

	c.logf("Successfully stored %d checkins using merge_account_checkins", len(checkins))
	return nil
}

// StoreRoutes stores routes in the database using merge_routes
func (c *Client) StoreRoutes(routes []api.Route) error {
	c.logf("Storing %d routes using merge_routes", len(routes))

	sqlLoader := NewSQLLoader(c.config.DatabaseType)
	sqlContent, err := sqlLoader.LoadMergeRoutesSQL()
	if err != nil {
		return fmt.Errorf("failed to load merge_routes SQL: %w (check if merge_routes.sql exists in database/%s directory)",
			err, c.config.DatabaseType)
	}

	// Begin transaction
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w (check database connection and transaction support)", err)
	}
	defer tx.Rollback()

	// Prepare statement
	stmt, err := tx.Prepare(sqlContent)
	if err != nil {
		return fmt.Errorf("failed to prepare merge_routes statement: %w (check SQL syntax in merge_routes.sql)", err)
	}
	defer stmt.Close()

	// Execute for each route
	for _, route := range routes {
		// Handle nullable Duration field
		var duration int
		if route.Duration != nil {
			duration = *route.Duration
		}

		_, err := stmt.Exec(
			route.ID,
			route.Name,
			route.RouteDate,
			duration,
			route.StartAddress,
			route.DestinationAddress,
			route.StartTime,
		)
		if err != nil {
			return fmt.Errorf("failed to merge route ID %d: %w (check route data and table schema compatibility)",
				route.ID, err)
		}

		// Store waypoints if any
		if len(route.Waypoints) > 0 {
			if err := c.storeWaypoints(tx, route.ID, route.Waypoints); err != nil {
				return fmt.Errorf("failed to store waypoints for route ID %d: %w", route.ID, err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w (check database connection and transaction state)", err)
	}

	c.logf("Successfully stored %d routes using merge_routes", len(routes))
	return nil
}

// storeWaypoints stores waypoints for a route
func (c *Client) storeWaypoints(tx *sql.Tx, routeID int, waypoints []api.Waypoint) error {
	// First, delete existing waypoints for this route
	sqlLoader := NewSQLLoader(c.config.DatabaseType)
	deleteSQL, err := sqlLoader.LoadDeleteRouteWaypointsSQL()
	if err != nil {
		return fmt.Errorf("failed to load delete_route_waypoints SQL: %w (check if delete_route_waypoints.sql exists in database/%s directory)",
			err, c.config.DatabaseType)
	}
	_, err = tx.Exec(deleteSQL, routeID)
	if err != nil {
		return fmt.Errorf("failed to delete existing waypoints for route ID %d: %w", routeID, err)
	}

	// Prepare insert statement
	insertSQL, err := sqlLoader.LoadInsertRouteWaypointsSQL()
	if err != nil {
		return fmt.Errorf("failed to load insert_route_waypoints SQL: %w (check if insert_route_waypoints.sql exists in database/%s directory)",
			err, c.config.DatabaseType)
	}
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare waypoint insert statement: %w", err)
	}
	defer stmt.Close()

	// Insert each waypoint
	for _, waypoint := range waypoints {
		// Handle nullable fields
		var suite, city, state, zipcode, completeAddress, apptTime, placeID string

		if waypoint.Suite != nil {
			suite = *waypoint.Suite
		}
		if waypoint.City != nil {
			city = *waypoint.City
		}
		if waypoint.State != nil {
			state = *waypoint.State
		}
		if waypoint.Zipcode != nil {
			zipcode = *waypoint.Zipcode
		}
		if waypoint.CompleteAddress != nil {
			completeAddress = *waypoint.CompleteAddress
		}
		if waypoint.ApptTime != nil {
			apptTime = *waypoint.ApptTime
		}
		if waypoint.PlaceID != nil {
			placeID = *waypoint.PlaceID
		}

		_, err := stmt.Exec(
			waypoint.ID,
			routeID,
			waypoint.Name,
			waypoint.Address,
			suite,
			city,
			state,
			zipcode,
			waypoint.Location,
			waypoint.Lat,
			waypoint.Long,
			waypoint.LayoverMinutes,
			waypoint.Position,
			completeAddress,
			waypoint.LocationID,
			waypoint.CustomerID,
			apptTime,
			waypoint.Type,
			placeID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert waypoint ID %d: %w", waypoint.ID, err)
		}
	}

	return nil
}
