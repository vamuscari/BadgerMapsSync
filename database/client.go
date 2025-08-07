package database

import (
	"database/sql"
	"fmt"
	"log"

	"badgermaps-cli/api"

	_ "github.com/lib/pq"               // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"     // SQLite driver
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
)

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
	config *Config
	db     *sql.DB
}

// NewClient creates a new database client
func NewClient(config *Config) (*Client, error) {
	client := &Client{
		config: config,
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
	default:
		return nil, fmt.Errorf("unsupported database type: %s", c.config.DatabaseType)
	}

	log.Printf("Connecting to %s database...", c.config.DatabaseType)
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to %s database", c.config.DatabaseType)
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
	log.Println("Initializing database schema...")

	// Validate SQL files exist before proceeding
	sqlLoader := NewSQLLoader(c.config.DatabaseType)
	if err := sqlLoader.ValidateSQLFiles(); err != nil {
		return fmt.Errorf("SQL validation failed: %w", err)
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
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
	}

	// Create indexes
	if err := c.createIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Println("Database schema initialized successfully")
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

	log.Printf("Table %s already exists, skipping creation", tableName)
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
			return false, fmt.Errorf("failed to check table existence: %w", err)
		}
		return exists, nil
	case "mssql", "sqlserver":
		query = "SELECT CASE WHEN EXISTS (SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = @p1) THEN 1 ELSE 0 END"
		var exists bool
		err := c.db.QueryRow(query, tableName).Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("failed to check table existence: %w", err)
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
			return false, fmt.Errorf("failed to check table existence: %w", err)
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported database type: %s", c.config.DatabaseType)
	}
}

// createTable creates a specific table based on the database type
func (c *Client) createTable(tableName string) error {
	log.Printf("Creating table: %s", tableName)

	sqlLoader := NewSQLLoader(c.config.DatabaseType)
	sqlContent, err := sqlLoader.LoadCreateTableSQL(tableName)
	if err != nil {
		return fmt.Errorf("failed to load SQL for table %s: %w", tableName, err)
	}

	_, err = c.db.Exec(sqlContent)
	if err != nil {
		return fmt.Errorf("failed to execute SQL for table %s: %w", tableName, err)
	}

	log.Printf("Successfully created table: %s", tableName)
	return nil
}

// createIndexes creates indexes for better performance
func (c *Client) createIndexes() error {
	log.Println("Creating database indexes...")

	sqlLoader := NewSQLLoader(c.config.DatabaseType)
	sqlContent, err := sqlLoader.LoadCreateIndexesSQL()
	if err != nil {
		return fmt.Errorf("failed to load index creation SQL: %w", err)
	}

	_, err = c.db.Exec(sqlContent)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Println("Database indexes created successfully")
	return nil
}

// DropAllTables drops all tables from the database
func (c *Client) DropAllTables() error {
	log.Println("Dropping all tables...")

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

	log.Println("All tables dropped successfully")
	return nil
}

// dropTable drops a specific table
func (c *Client) dropTable(tableName string) error {
	exists, err := c.tableExists(tableName)
	if err != nil {
		return fmt.Errorf("failed to check if table %s exists: %w", tableName, err)
	}

	if !exists {
		log.Printf("Table %s does not exist, skipping drop", tableName)
		return nil
	}

	query := fmt.Sprintf("DROP TABLE %s", tableName)
	_, err = c.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}

	log.Printf("Successfully dropped table: %s", tableName)
	return nil
}

// StoreAccounts stores accounts in the database using merge_accounts_basic
func (c *Client) StoreAccounts(accounts []api.Account) error {
	log.Printf("Storing %d accounts using merge_accounts_basic", len(accounts))

	sqlLoader := NewSQLLoader(c.config.DatabaseType)
	sqlContent, err := sqlLoader.LoadMergeAccountsBasicSQL()
	if err != nil {
		return fmt.Errorf("failed to load merge_accounts_basic SQL: %w", err)
	}

	// Begin transaction
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare statement
	stmt, err := tx.Prepare(sqlContent)
	if err != nil {
		return fmt.Errorf("failed to prepare merge_accounts_basic statement: %w", err)
	}
	defer stmt.Close()

	// Execute for each account
	for _, account := range accounts {
		firstName := ""
		if account.FirstName != nil {
			firstName = *account.FirstName
		}

		_, err := stmt.Exec(account.ID, firstName, account.LastName)
		if err != nil {
			return fmt.Errorf("failed to merge account %d: %w", account.ID, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully stored %d accounts using merge_accounts_basic", len(accounts))
	return nil
}

// GetAccountIDs retrieves all account IDs from the database
func (c *Client) GetAccountIDs() ([]int, error) {
	log.Println("Retrieving all account IDs from database")

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

	log.Printf("Retrieved %d account IDs from database", len(accountIDs))
	return accountIDs, nil
}
