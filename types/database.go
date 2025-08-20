package types

// Database is an interface that defines methods for database configuration
type Database interface {
	// GetType returns the database type (e.g., "sqlite3", "postgres", "mssql")
	GetType() string

	// GetDatabaseConnection returns the url connection string for the database
	DatabaseConnection() string

	// GetDatabaseSetting gets the database fields in the config file based on the database type
	GetDatabaseSettings() error

	// SetDatabaseSetting sets the database fields in the config file based on the database type
	SetDatabaseSettings() error

	// PromptDatabaseSettings prompts the user configuration on database
	PromptDatabaseSettings()
}
