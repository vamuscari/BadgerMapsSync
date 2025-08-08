package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

// SQLLoader handles loading and executing SQL files
type SQLLoader struct {
	databaseType string
}

// NewSQLLoader creates a new SQL loader
func NewSQLLoader(databaseType string) *SQLLoader {
	return &SQLLoader{
		databaseType: databaseType,
	}
}

// ValidateSQLFiles checks that all required SQL files exist for the database type
func (loader *SQLLoader) ValidateSQLFiles() error {
	requiredFiles := []string{
		"create_accounts_table.sql",
		"create_routes_table.sql",
		"create_account_checkins_table.sql",
		"create_user_profiles_table.sql",
		"create_account_locations_table.sql",
		"create_route_waypoints_table.sql",
		"create_data_sets_table.sql",
		"create_data_set_values_table.sql",
		"create_indexes.sql",
	}

	// If SQL files are embedded, validate using embedded files
	if IsEmbedded() {
		var missingFiles []string
		dbType := loader.getDatabaseTypeForSQL()

		for _, filename := range requiredFiles {
			_, err := GetEmbeddedSQL(dbType, filename)
			if err != nil {
				missingFiles = append(missingFiles, filename)
			}
		}

		if len(missingFiles) > 0 {
			return fmt.Errorf("missing required embedded SQL files for %s database: %v",
				loader.databaseType, missingFiles)
		}

		return nil
	}

	// Otherwise, validate using filesystem
	var missingFiles []string
	for _, filename := range requiredFiles {
		sqlPath := loader.getSQLPath(filename)
		if _, err := ioutil.ReadFile(sqlPath); err != nil {
			missingFiles = append(missingFiles, filename)
		}
	}

	if len(missingFiles) > 0 {
		dbDir := filepath.Join("database", loader.getDatabaseTypeForSQL())
		return fmt.Errorf("missing required SQL files for %s database: %v (check if these files exist in the %s directory)",
			loader.databaseType, missingFiles, dbDir)
	}

	return nil
}

// LoadCreateTableSQL loads the SQL for creating a specific table
func (loader *SQLLoader) LoadCreateTableSQL(tableName string) (string, error) {
	// Map table names to their corresponding SQL files
	tableFileMap := map[string]string{
		"accounts":          "create_accounts_table.sql",
		"routes":            "create_routes_table.sql",
		"checkins":          "create_account_checkins_table.sql",
		"user_profiles":     "create_user_profiles_table.sql",
		"account_locations": "create_account_locations_table.sql",
		"route_waypoints":   "create_route_waypoints_table.sql",
		"data_sets":         "create_data_sets_table.sql",
		"data_set_values":   "create_data_set_values_table.sql",
	}

	filename, exists := tableFileMap[tableName]
	if !exists {
		// Fallback to the default naming convention
		filename = fmt.Sprintf("create_%s_table.sql", tableName)
	}

	sqlPath := loader.getSQLPath(filename)

	content, err := ioutil.ReadFile(sqlPath)
	if err != nil {
		dbDir := filepath.Join("database", loader.getDatabaseTypeForSQL())
		return "", fmt.Errorf("failed to read SQL file %s: %w (check if file exists and has correct permissions in %s directory)",
			filename, err, dbDir)
	}

	return string(content), nil
}

// LoadCreateIndexesSQL loads the SQL for creating indexes
func (loader *SQLLoader) LoadCreateIndexesSQL() (string, error) {
	filename := "create_indexes.sql"
	sqlPath := loader.getSQLPath(filename)

	content, err := ioutil.ReadFile(sqlPath)
	if err != nil {
		dbDir := filepath.Join("database", loader.getDatabaseTypeForSQL())
		return "", fmt.Errorf("failed to read SQL file %s: %w (check if file exists and has correct permissions in %s directory)",
			filename, err, dbDir)
	}

	return string(content), nil
}

// LoadMergeAccountsBasicSQL loads the SQL for merging accounts using merge_accounts_basic
func (loader *SQLLoader) LoadMergeAccountsBasicSQL() (string, error) {
	filename := "merge_accounts_basic.sql"
	sqlPath := loader.getSQLPath(filename)

	content, err := ioutil.ReadFile(sqlPath)
	if err != nil {
		dbDir := filepath.Join("database", loader.getDatabaseTypeForSQL())
		return "", fmt.Errorf("failed to read SQL file %s: %w (check if file exists and has correct permissions in %s directory)",
			filename, err, dbDir)
	}

	return string(content), nil
}

// LoadMergeUserProfilesSQL loads the SQL for merging user profiles using merge_user_profiles
func (loader *SQLLoader) LoadMergeUserProfilesSQL() (string, error) {
	filename := "merge_user_profiles.sql"
	sqlPath := loader.getSQLPath(filename)

	content, err := ioutil.ReadFile(sqlPath)
	if err != nil {
		dbDir := filepath.Join("database", loader.getDatabaseTypeForSQL())
		return "", fmt.Errorf("failed to read SQL file %s: %w (check if file exists and has correct permissions in %s directory)",
			filename, err, dbDir)
	}

	return string(content), nil
}

// LoadMergeAccountCheckinsSQL loads the SQL for merging account checkins using merge_account_checkins
func (loader *SQLLoader) LoadMergeAccountCheckinsSQL() (string, error) {
	filename := "merge_account_checkins.sql"
	sqlPath := loader.getSQLPath(filename)

	content, err := ioutil.ReadFile(sqlPath)
	if err != nil {
		dbDir := filepath.Join("database", loader.getDatabaseTypeForSQL())
		return "", fmt.Errorf("failed to read SQL file %s: %w (check if file exists and has correct permissions in %s directory)",
			filename, err, dbDir)
	}

	return string(content), nil
}

// LoadMergeRoutesSQL loads the SQL for merging routes using merge_routes
func (loader *SQLLoader) LoadMergeRoutesSQL() (string, error) {
	filename := "merge_routes.sql"
	sqlPath := loader.getSQLPath(filename)

	content, err := ioutil.ReadFile(sqlPath)
	if err != nil {
		dbDir := filepath.Join("database", loader.getDatabaseTypeForSQL())
		return "", fmt.Errorf("failed to read SQL file %s: %w (check if file exists and has correct permissions in %s directory)",
			filename, err, dbDir)
	}

	return string(content), nil
}

// LoadAndExecuteSQL loads and executes a SQL file
func (loader *SQLLoader) LoadAndExecuteSQL(filename string, db *sql.DB) error {
	sqlPath := loader.getSQLPath(filename)

	content, err := ioutil.ReadFile(sqlPath)
	if err != nil {
		dbDir := filepath.Join("database", loader.getDatabaseTypeForSQL())
		return fmt.Errorf("failed to read SQL file %s: %w (check if file exists and has correct permissions in %s directory)",
			filename, err, dbDir)
	}

	sqlContent := string(content)

	// Split by semicolons to handle multiple statements
	statements := strings.Split(sqlContent, ";")

	for i, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		log.Printf("Executing SQL: %s", statement[:min(100, len(statement))]+"...")

		_, err := db.Exec(statement)
		if err != nil {
			// Include statement number to help identify problematic statements
			return fmt.Errorf("failed to execute SQL statement #%d in %s: %w (check SQL syntax and database permissions)",
				i+1, filename, err)
		}
	}

	return nil
}

// getSQLPath returns the appropriate SQL file path based on database type
func (loader *SQLLoader) getSQLPath(filename string) string {
	dbType := loader.getDatabaseTypeForSQL()
	return filepath.Join("database", dbType, filename)
}

// LoadSQL loads a generic SQL file by name
func (loader *SQLLoader) LoadSQL(filename string) (string, error) {
	// If SQL files are embedded, use them
	if IsEmbedded() {
		return GetEmbeddedSQL(loader.getDatabaseTypeForSQL(), filename)
	}

	// Otherwise, load from filesystem
	sqlPath := loader.getSQLPath(filename)
	content, err := ioutil.ReadFile(sqlPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SQL file %s: %w", sqlPath, err)
	}
	return string(content), nil
}

// getDatabaseTypeForSQL returns the database type directory name
func (loader *SQLLoader) getDatabaseTypeForSQL() string {
	switch loader.databaseType {
	case "postgres", "postgresql":
		return "postgres"
	case "mssql", "sqlserver":
		return "mssql"
	case "sqlite3", "sqlite":
		return "sqlite3"
	default:
		return "sqlite3" // fallback
	}
}

// CreateAllTables creates all necessary tables
func (loader *SQLLoader) CreateAllTables(db *sql.DB) error {
	tables := []string{
		"create_accounts_table.sql",
		"create_account_locations_table.sql",
		"create_account_checkins_table.sql",
		"create_routes_table.sql",
		"create_route_waypoints_table.sql",
		"create_data_sets_table.sql",
		"create_data_set_values_table.sql",
		"create_user_profiles_table.sql",
	}

	for _, table := range tables {
		log.Printf("Creating table using: %s", table)
		if err := loader.LoadAndExecuteSQL(table, db); err != nil {
			return fmt.Errorf("failed to create table with %s: %w (check SQL syntax and database permissions)",
				table, err)
		}
	}

	return nil
}

// CreateIndexes creates all necessary indexes
func (loader *SQLLoader) CreateIndexes(db *sql.DB) error {
	log.Println("Creating indexes...")
	indexFile := "create_indexes.sql"
	if err := loader.LoadAndExecuteSQL(indexFile, db); err != nil {
		return fmt.Errorf("failed to create indexes with %s: %w (check SQL syntax and database permissions)",
			indexFile, err)
	}
	return nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
