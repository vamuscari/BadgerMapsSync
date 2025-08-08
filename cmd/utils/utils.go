package utils

import (
	"fmt"
	"os"

	"badgermapscli/common"
	"badgermapscli/database"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewUtilsCmd creates a new utils command
func NewUtilsCmd() *cobra.Command {
	utilsCmd := &cobra.Command{
		Use:   "utils",
		Short: "Utility commands for maintenance",
		Long:  `Utility commands for database maintenance, schema validation, and other operations.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a utility command")
			cmd.Help()
		},
	}

	// Add subcommands
	utilsCmd.AddCommand(newCreateTablesCmd())
	utilsCmd.AddCommand(newValidateSchemaCmd())
	utilsCmd.AddCommand(newDropTablesCmd())
	utilsCmd.AddCommand(newInitDatabaseCmd())
	utilsCmd.AddCommand(newCreateEnvCmd())

	return utilsCmd
}

// newCreateTablesCmd creates a command to create database tables
func newCreateTablesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-tables",
		Short: "Create database tables",
		Long:  `Create all necessary database tables if they don't exist.`,
		Run: func(cmd *cobra.Command, args []string) {
			createTables()
		},
	}

	return cmd
}

// newValidateSchemaCmd creates a command to validate database schema
func newValidateSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-schema",
		Short: "Validate database schema",
		Long:  `Validate that all required database tables exist with the correct fields.`,
		Run: func(cmd *cobra.Command, args []string) {
			validateSchema()
		},
	}

	return cmd
}

// newDropTablesCmd creates a command to drop database tables
func newDropTablesCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "drop-tables",
		Short: "Drop database tables",
		Long:  `Drop all database tables. This will delete all data in the database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if !force {
				fmt.Println(color.YellowString("Warning: This will delete all data in the database."))
				fmt.Println("Use --force to confirm.")
				return
			}
			dropTables()
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force dropping tables without confirmation")

	return cmd
}

// newInitDatabaseCmd creates a command to initialize the database
func newInitDatabaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init-database",
		Short: "Initialize database",
		Long:  `Initialize the database with all necessary tables and indexes.`,
		Run: func(cmd *cobra.Command, args []string) {
			initDatabase()
		},
	}

	return cmd
}

// createTables creates all necessary database tables
func createTables() {
	fmt.Println(color.CyanString("Creating database tables..."))

	// Get database configuration from viper
	dbConfig := getDatabaseConfig()

	// Create database client
	client, err := database.NewClient(dbConfig, false)
	if err != nil {
		fmt.Println(color.RedString("Error creating database client: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Create tables
	err = client.InitializeSchema()
	if err != nil {
		fmt.Println(color.RedString("Error creating tables: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Database tables created successfully"))
}

// validateSchema validates the database schema
func validateSchema() {
	fmt.Println(color.CyanString("Validating database schema..."))

	// Get database configuration from viper
	dbConfig := getDatabaseConfig()

	// Create database client
	client, err := database.NewClient(dbConfig, false)
	if err != nil {
		fmt.Println(color.RedString("Error creating database client: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Validate schema
	err = client.ValidateDatabaseSchema()
	if err != nil {
		fmt.Println(color.RedString("Schema validation failed: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Database schema validation successful"))
}

// dropTables drops all database tables
func dropTables() {
	fmt.Println(color.CyanString("Dropping database tables..."))

	// Get database configuration from viper
	dbConfig := getDatabaseConfig()

	// Create database client
	client, err := database.NewClient(dbConfig, false)
	if err != nil {
		fmt.Println(color.RedString("Error creating database client: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Drop tables
	err = client.DropAllTables()
	if err != nil {
		fmt.Println(color.RedString("Error dropping tables: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Database tables dropped successfully"))
}

// initDatabase initializes the database
func initDatabase() {
	fmt.Println(color.CyanString("Initializing database..."))

	// Get database configuration from viper
	dbConfig := getDatabaseConfig()

	// Create database client
	client, err := database.NewClient(dbConfig, false)
	if err != nil {
		fmt.Println(color.RedString("Error creating database client: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Initialize schema
	err = client.InitializeSchema()
	if err != nil {
		fmt.Println(color.RedString("Error initializing database: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Database initialized successfully"))
}

// newCreateEnvCmd creates a command to create a new .env file
func newCreateEnvCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "create-env",
		Short: "Create a new environment file",
		Long:  `Create a new .env file with default configuration values.`,
		Run: func(cmd *cobra.Command, args []string) {
			createEnvFile(force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force creation of .env file even if it already exists")

	return cmd
}

// createEnvFile creates a new .env file with default configuration values
func createEnvFile(force bool) {
	fmt.Println(color.CyanString("Creating .env file..."))

	err := common.CreateEnvFile(force)
	if err != nil {
		fmt.Println(color.RedString("Error creating .env file: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Environment file created successfully"))
	fmt.Println("You can now edit the .env file to customize your configuration")
}

// getDatabaseConfig gets the database configuration from viper
func getDatabaseConfig() *database.Config {
	dbType := viper.GetString("DATABASE_TYPE")
	if dbType == "" {
		dbType = "sqlite3" // Default to SQLite
	}

	dbConfig := &database.Config{
		DatabaseType: dbType,
		Host:         viper.GetString("DATABASE_HOST"),
		Port:         viper.GetString("DATABASE_PORT"),
		Database:     viper.GetString("DATABASE_NAME"),
		Username:     viper.GetString("DATABASE_USERNAME"),
		Password:     viper.GetString("DATABASE_PASSWORD"),
	}

	// Set default database name for SQLite if not provided
	if dbType == "sqlite3" && dbConfig.Database == "" {
		dbConfig.Database = "badgermaps.db"
	}

	return dbConfig
}
