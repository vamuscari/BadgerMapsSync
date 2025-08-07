package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"badgermaps-cli/api"
	"badgermaps-cli/database"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	configFile string
)

// ANSI color codes
const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
	colorReset  = "\033[0m"
)

// colorText returns colored text using ANSI codes
func colorText(text string, color string) string {
	return color + text + colorReset
}

// boldText returns bold text using ANSI codes
func boldText(text string) string {
	return colorBold + text + colorReset
}

// Config holds the complete configuration for the BadgerMaps client
type Config struct {
	DatabaseType string
	Host         string
	Port         string
	Database     string
	Username     string
	Password     string
	APIKey       string
	APIURL       string
	LogLevel     string
	LogFile      string
}

// Client represents the BadgerMaps sync client
type Client struct {
	config *Config
	db     *database.Client
	api    *api.APIClient
}

// loadConfigFromEnv loads database and API configuration from environment variables
func loadConfigFromEnv() *Config {
	config := &Config{
		DatabaseType: getEnvOrDefault("DB_TYPE", "sqlite3"),
		Host:         getEnvOrDefault("DB_HOST", "localhost"),
		Port:         getEnvOrDefault("DB_PORT", "5432"),
		Database:     getEnvOrDefault("DB_NAME", "badgersync.db"),
		Username:     getEnvOrDefault("DB_USER", ""),
		Password:     getEnvOrDefault("DB_PASSWORD", ""),
		APIKey:       getEnvOrDefault("BADGERMAPS_API_KEY", ""),
		APIURL:       getEnvOrDefault("BADGERMAPS_API_URL", "https://api.badgermapping.com/v1"),
		LogLevel:     getEnvOrDefault("LOG_LEVEL", "info"),
		LogFile:      getEnvOrDefault("LOG_FILE", ""),
	}

	// Set appropriate default port based on database type
	if config.DatabaseType == "mssql" || config.DatabaseType == "sqlserver" {
		if config.Port == "5432" { // If still using default PostgreSQL port
			config.Port = "1433"
		}
	}

	return config
}

// SetupLogging configures logging based on the configuration
func SetupLogging(config *Config) error {
	// Set log flags for better formatting
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// If log file is specified, set up file logging
	if config.LogFile != "" {
		// Create directory if it doesn't exist
		logDir := filepath.Dir(config.LogFile)
		if logDir != "." && logDir != "" {
			if err := os.MkdirAll(logDir, 0755); err != nil {
				return fmt.Errorf("failed to create log directory: %w", err)
			}
		}

		// Open log file
		logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		// Create a multi-writer to log to both file and console
		multiWriter := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multiWriter)
		log.Printf("Logging initialized - file: %s, level: %s", config.LogFile, config.LogLevel)
	} else {
		log.Printf("Logging initialized - console only, level: %s", config.LogLevel)
	}

	return nil
}

// NewClient creates a new BadgerMaps sync client
func NewClient(config *Config) (*Client, error) {
	client := &Client{
		config: config,
	}

	// Initialize database client
	dbConfig := &database.Config{
		DatabaseType: config.DatabaseType,
		Host:         config.Host,
		Port:         config.Port,
		Database:     config.Database,
		Username:     config.Username,
		Password:     config.Password,
	}

	dbClient, err := database.NewClient(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database client: %w", err)
	}
	client.db = dbClient

	// Initialize API client
	if config.APIKey != "" {
		client.api = api.NewAPIClientWithURL(config.APIKey, config.APIURL)
	}

	return client, nil
}

// Close closes the database connection
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Sync performs a full sync operation
func (c *Client) Sync() error {
	log.Println("Starting sync operation...")

	// Initialize database schema
	if err := c.InitializeSchema(); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	// TODO: Implement actual data sync
	log.Println("Sync operation completed")
	return nil
}

// SyncAccounts pulls accounts from API and stores them in the database using merge_accounts_basic
func (c *Client) SyncAccounts() error {
	log.Println("Syncing accounts from API using merge_accounts_basic...")

	// Get accounts from API
	accounts, err := c.api.GetAccounts()
	if err != nil {
		return fmt.Errorf("failed to get accounts from API: %w", err)
	}

	log.Printf("Retrieved %d accounts from API", len(accounts))

	// Store accounts in database using merge_accounts_basic
	if err := c.db.StoreAccounts(accounts); err != nil {
		return fmt.Errorf("failed to store accounts in database: %w", err)
	}

	log.Printf("Successfully stored %d accounts in database using merge_accounts_basic", len(accounts))

	// Get complete list of account IDs from database
	accountIDs, err := c.db.GetAccountIDs()
	if err != nil {
		return fmt.Errorf("failed to get account IDs from database: %w", err)
	}

	log.Printf("Retrieved %d account IDs from database", len(accountIDs))

	// Fetch account details for each account ID
	log.Println("Fetching detailed account information for each account...")
	for i, accountID := range accountIDs {
		log.Printf("Fetching details for account %d (%d/%d)", accountID, i+1, len(accountIDs))

		accountDetail, err := c.api.GetAccount(accountID)
		if err != nil {
			log.Printf("Warning: failed to get details for account %d: %v", accountID, err)
			continue
		}

		firstName := ""
		if accountDetail.FirstName != nil {
			firstName = *accountDetail.FirstName
		}
		log.Printf("Successfully retrieved details for account %d: %s %s",
			accountDetail.ID, firstName, accountDetail.LastName)
	}

	log.Printf("Completed fetching account details for %d accounts", len(accountIDs))
	return nil
}

// SyncProfile pulls user profile from API and stores it in the database using merge_user_profiles
func (c *Client) SyncProfile() error {
	log.Println("Syncing user profile from API using merge_user_profiles...")

	// Get user profile from API
	profile, err := c.api.GetUserProfile()
	if err != nil {
		return fmt.Errorf("failed to get user profile from API: %w", err)
	}

	log.Printf("Retrieved user profile for %s %s (ID: %d) from API",
		profile.FirstName, profile.LastName, profile.ID)

	// Store profile in database using merge_user_profiles
	if err := c.db.StoreProfiles(profile); err != nil {
		return fmt.Errorf("failed to store user profile in database: %w", err)
	}

	log.Printf("Successfully stored user profile in database using merge_user_profiles")
	return nil
}

// InitializeSchema creates all necessary tables and indexes
func (c *Client) InitializeSchema() error {
	return c.db.InitializeSchema()
}

// DropAllTables drops all tables from the database
func (c *Client) DropAllTables() error {
	return c.db.DropAllTables()
}

// TestAPIConnectivity tests the API connectivity
func (c *Client) TestAPIConnectivity() error {
	if c.api == nil {
		return fmt.Errorf("no API key configured. Set BADGERMAPS_API_KEY environment variable")
	}
	return c.api.TestAPIConnection()
}

// TestAPIEndpoints tests all API endpoints
func (c *Client) TestAPIEndpoints() map[string]error {
	if c.api == nil {
		return map[string]error{
			"all": fmt.Errorf("no API key configured. Set BADGERMAPS_API_KEY environment variable"),
		}
	}
	return c.api.TestAllEndpoints()
}

// loadEnvFile loads environment variables from .env file if it exists
func loadEnvFile() error {
	// Try to load .env file, but don't fail if it doesn't exist
	err := godotenv.Load()
	if err != nil {
		// .env file doesn't exist, which is fine
		log.Printf("No .env file found, using environment variables only")
	} else {
		log.Printf("Loaded environment variables from .env file")
	}
	return nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// validateDataType validates if the provided data type is supported
func validateDataType(dataType string) bool {
	validTypes := []string{"accounts", "routes", "checkins", "profiles", "all"}
	for _, validType := range validTypes {
		if strings.ToLower(dataType) == validType {
			return true
		}
	}
	return false
}

// getDataTypes returns the list of data types to process
func getDataTypes(args []string) []string {
	if len(args) == 0 {
		// No arguments provided, pull/push all data types
		return []string{"accounts", "routes", "checkins", "profiles"}
	}

	var dataTypes []string
	for _, arg := range args {
		dataType := strings.ToLower(arg)
		if dataType == "all" {
			return []string{"accounts", "routes", "checkins", "profiles"}
		}
		if validateDataType(dataType) {
			dataTypes = append(dataTypes, dataType)
		} else {
			log.Printf("Warning: Unknown data type '%s', skipping", arg)
		}
	}

	return dataTypes
}

var rootCmd = &cobra.Command{
	Use:   "badgersync",
	Short: "BadgerMaps CLI for syncing data",
	Long:  `A CLI tool for syncing BadgerMaps data with various databases including PostgreSQL, SQL Server, and SQLite.`,
}

var pullCmd = &cobra.Command{
	Use:   "pull [data-types...]",
	Short: "Pull data from BadgerMaps API",
	Long: `Pull data from BadgerMaps API and store it in the configured database.

Available data types:
  accounts    - Customer account information
  routes      - Route planning data
  checkins    - Sales visit tracking
  profiles    - User profile data
  all         - Pull all data types (default if no arguments provided)

Examples:
  ./badgersync pull                    # Pull all data types
  ./badgersync pull all               # Pull all data types
  ./badgersync pull accounts          # Pull only accounts
  ./badgersync pull profiles routes   # Pull profiles and routes`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		config := loadConfigFromEnv()

		client, err := NewClient(config)
		if err != nil {
			log.Fatal("Failed to create client:", err)
		}

		dataTypes := getDataTypes(args)
		if len(dataTypes) == 0 {
			log.Fatal("No valid data types specified")
		}

		log.Printf("Pulling data types: %s", strings.Join(dataTypes, ", "))

		// Initialize database schema first
		if err := client.InitializeSchema(); err != nil {
			log.Fatal("Failed to initialize schema:", err)
		}

		// Process each data type
		for _, dataType := range dataTypes {
			switch dataType {
			case "accounts":
				log.Printf("Pulling accounts...")
				if err := client.SyncAccounts(); err != nil {
					log.Fatal("Failed to pull accounts:", err)
				}
				fmt.Printf("Successfully pulled accounts\n")
			case "routes":
				log.Printf("Pulling routes...")
				// TODO: Implement routes pulling
				fmt.Printf("Routes pulling not yet implemented\n")
			case "checkins":
				log.Printf("Pulling checkins...")
				// TODO: Implement checkins pulling
				fmt.Printf("Checkins pulling not yet implemented\n")
			case "profiles":
				log.Printf("Pulling profile...")
				if err := client.SyncProfile(); err != nil {
					log.Fatal("Failed to pull profile:", err)
				}
				fmt.Printf("Successfully pulled profile\n")
			}
		}

		fmt.Printf("Pull completed successfully for: %s\n", strings.Join(dataTypes, ", "))
	},
}

var pushCmd = &cobra.Command{
	Use:   "push [data-types...]",
	Short: "Push data to BadgerMaps API",
	Long: `Push data from the configured database to BadgerMaps API.

Available data types:
  accounts    - Customer account information
  routes      - Route planning data
  checkins    - Sales visit tracking
  profiles    - User profile data
  all         - Push all data types (default if no arguments provided)

Examples:
  ./badgersync push                    # Push all data types
  ./badgersync push all               # Push all data types
  ./badgersync push accounts          # Push only accounts
  ./badgersync push profiles routes   # Push profiles and routes`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		config := loadConfigFromEnv()

		client, err := NewClient(config)
		if err != nil {
			log.Fatal("Failed to create client:", err)
		}
		defer client.Close()

		dataTypes := getDataTypes(args)
		if len(dataTypes) == 0 {
			log.Fatal("No valid data types specified")
		}

		log.Printf("Pushing data types: %s", strings.Join(dataTypes, ", "))

		// TODO: Implement specific data type pushing
		fmt.Printf("Push functionality not yet implemented for: %s\n", strings.Join(dataTypes, ", "))
	},
}

var utilsCmd = &cobra.Command{
	Use:   "utils",
	Short: "Utility commands",
	Long:  `Various utility commands for database management and data operations.`,
}

var createTablesCmd = &cobra.Command{
	Use:   "create-tables",
	Short: "Create all tables in the database",
	Long:  `Create all necessary tables in the configured database without syncing data.`,
	Run: func(cmd *cobra.Command, args []string) {
		config := loadConfigFromEnv()

		client, err := NewClient(config)
		if err != nil {
			log.Fatal("Failed to create client:", err)
		}
		defer client.Close()

		// Initialize database schema only
		if err := client.InitializeSchema(); err != nil {
			log.Fatal("Failed to create tables:", err)
		}

		fmt.Println("All tables created successfully")
	},
}

var dropTablesCmd = &cobra.Command{
	Use:   "drop-tables",
	Short: "Drop all tables from the database",
	Long:  `Drop all tables from the configured database. WARNING: This will delete all data!`,
	Run: func(cmd *cobra.Command, args []string) {
		config := loadConfigFromEnv()

		client, err := NewClient(config)
		if err != nil {
			log.Fatal("Failed to create client:", err)
		}
		defer client.Close()

		// Drop all tables
		if err := client.DropAllTables(); err != nil {
			log.Fatal("Failed to drop tables:", err)
		}

		fmt.Println("All tables dropped successfully")
	},
}

var testCmd = &cobra.Command{
	Use:   "test [command] [args...]",
	Short: "Run tests",
	Long: `Run various tests to verify functionality including API connectivity.

Examples:
  ./badgersync test                    # Run basic connectivity tests
  ./badgersync test pull profiles      # Test pull profiles command
  ./badgersync test pull all           # Test pull all command
  ./badgersync test push accounts      # Test push accounts command`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		config := loadConfigFromEnv()

		// If specific command to test is provided
		if len(args) > 0 {
			testSpecificCommand(args, config)
			return
		}

		// Run basic connectivity tests
		fmt.Println(boldText("Running BadgerMaps CLI tests..."))
		fmt.Println("==================================")

		// Test database connectivity
		fmt.Println("\n" + colorText("Testing database connectivity...", colorBlue))
		client, err := NewClient(config)
		if err != nil {
			fmt.Printf("%s Database connection failed: %v\n", colorText("ERROR", colorRed), err)
			os.Exit(1)
		}
		defer client.Close()
		fmt.Printf("%s Database connection successful\n", colorText("SUCCESS", colorGreen))

		// Test API connectivity
		fmt.Println("\n" + colorText("Testing API connectivity...", colorBlue))
		if err := client.TestAPIConnectivity(); err != nil {
			fmt.Printf("%s API connection failed: %v\n", colorText("ERROR", colorRed), err)
			fmt.Println(colorText("Make sure BADGERMAPS_API_KEY is set correctly", colorYellow))
		} else {
			fmt.Printf("%s API connection successful\n", colorText("SUCCESS", colorGreen))
		}

		// Test individual API endpoints
		fmt.Println("\n" + colorText("Testing API endpoints...", colorBlue))
		endpointResults := client.TestAPIEndpoints()

		allPassed := true
		for endpoint, err := range endpointResults {
			if err != nil {
				fmt.Printf("%s %s endpoint failed: %v\n", colorText("ERROR", colorRed), endpoint, err)
				allPassed = false
			} else {
				fmt.Printf("%s %s endpoint successful\n", colorText("SUCCESS", colorGreen), endpoint)
			}
		}

		fmt.Println("\n" + boldText("Test Summary:"))
		fmt.Println("==================")
		fmt.Printf("%s Database connectivity: PASS\n", colorText("SUCCESS", colorGreen))

		if config.APIKey != "" {
			if allPassed {
				fmt.Printf("%s API connectivity: PASS\n", colorText("SUCCESS", colorGreen))
				fmt.Printf("%s All API endpoints: PASS\n", colorText("SUCCESS", colorGreen))
			} else {
				fmt.Printf("%s API connectivity: PARTIAL\n", colorText("WARNING", colorYellow))
				fmt.Printf("%s Some API endpoints: FAIL\n", colorText("ERROR", colorRed))
			}
		} else {
			fmt.Printf("%s API connectivity: SKIPPED (no API key)\n", colorText("WARNING", colorYellow))
		}

		fmt.Println("\n" + colorText("Tests completed!", colorGreen))
	},
}

// testSpecificCommand tests a specific command and saves formatted API calls
func testSpecificCommand(args []string, config *Config) {
	command := args[0]
	commandArgs := args[1:]

	fmt.Printf("%s Testing command: %s %s\n", boldText("TESTING"), command, strings.Join(commandArgs, " "))
	fmt.Println("==================================")

	// Create client
	client, err := NewClient(config)
	if err != nil {
		fmt.Printf("%s Failed to create client: %v\n", colorText("ERROR", colorRed), err)
		os.Exit(1)
	}
	defer client.Close()

	// Test database connectivity first
	fmt.Println("\n" + colorText("Testing database connectivity...", colorBlue))
	if err := client.InitializeSchema(); err != nil {
		fmt.Printf("%s Database connection failed: %v\n", colorText("ERROR", colorRed), err)
		os.Exit(1)
	}
	fmt.Printf("%s Database connection successful\n", colorText("SUCCESS", colorGreen))

	// Test API connectivity
	fmt.Println("\n" + colorText("Testing API connectivity...", colorBlue))
	if err := client.TestAPIConnectivity(); err != nil {
		fmt.Printf("%s API connection failed: %v\n", colorText("WARNING", colorYellow), err)
	} else {
		fmt.Printf("%s API connection successful\n", colorText("SUCCESS", colorGreen))
	}

	// Make actual API calls and save responses
	fmt.Println("\n" + colorText("Making API calls and saving responses...", colorBlue))
	saveAPIResponses(command, commandArgs, config, client)

	fmt.Println("\n" + colorText("Command test completed!", colorGreen))
}

// saveAPIExamples generates and saves formatted API call examples
func saveAPIExamples(command string, args []string, config *Config) {
	examples := make(map[string]interface{})

	switch command {
	case "pull":
		examples = generatePullExamples(args, config)
	case "push":
		examples = generatePushExamples(args, config)
	default:
		fmt.Printf("%s No API examples available for command: %s\n", colorText("WARNING", colorYellow), command)
		return
	}

	// Save examples to file
	filename := fmt.Sprintf("api_examples_%s_%s.json", command, strings.Join(args, "_"))
	if len(args) == 0 {
		filename = fmt.Sprintf("api_examples_%s_all.json", command)
	}

	// Create examples directory if it doesn't exist
	if err := os.MkdirAll("api_examples", 0755); err != nil {
		fmt.Printf("%s Failed to create examples directory: %v\n", colorText("ERROR", colorRed), err)
		return
	}

	filepath := fmt.Sprintf("api_examples/%s", filename)

	// Convert to JSON
	jsonData, err := json.MarshalIndent(examples, "", "  ")
	if err != nil {
		fmt.Printf("%s Failed to marshal examples: %v\n", colorText("ERROR", colorRed), err)
		return
	}

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		fmt.Printf("%s Failed to write examples file: %v\n", colorText("ERROR", colorRed), err)
		return
	}

	fmt.Printf("%s API examples saved to: %s\n", colorText("SUCCESS", colorGreen), filepath)
}

// saveAPIResponses makes actual API calls and saves the responses
func saveAPIResponses(command string, args []string, config *Config, client *Client) {
	responses := make(map[string]interface{})

	switch command {
	case "pull":
		responses = makePullAPICalls(args, config, client)
	case "push":
		responses = makePushAPICalls(args, config, client)
	default:
		fmt.Printf("%s No API calls available for command: %s\n", colorText("WARNING", colorYellow), command)
		return
	}

	// Save responses to file with datetime
	currentTime := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("api_responses_%s_%s_%s.json", command, strings.Join(args, "_"), currentTime)
	if len(args) == 0 {
		filename = fmt.Sprintf("api_responses_%s_all_%s.json", command, currentTime)
	}

	// Create responses directory if it doesn't exist
	if err := os.MkdirAll("test_responses", 0755); err != nil {
		fmt.Printf("%s Failed to create responses directory: %v\n", colorText("ERROR", colorRed), err)
		return
	}

	filepath := fmt.Sprintf("test_responses/%s", filename)

	// Convert to JSON
	jsonData, err := json.MarshalIndent(responses, "", "  ")
	if err != nil {
		fmt.Printf("%s Failed to marshal responses: %v\n", colorText("ERROR", colorRed), err)
		return
	}

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		fmt.Printf("%s Failed to write responses file: %v\n", colorText("ERROR", colorRed), err)
		return
	}

	fmt.Printf("%s Test responses saved to: %s\n", colorText("SUCCESS", colorGreen), filepath)
}

// makePullAPICalls makes actual API calls for pull command and returns responses
func makePullAPICalls(args []string, config *Config, client *Client) map[string]interface{} {
	responses := map[string]interface{}{
		"command":   "pull",
		"args":      args,
		"responses": map[string]interface{}{},
	}

	dataTypes := getDataTypes(args)
	if len(dataTypes) == 0 {
		dataTypes = []string{"all"}
	}

	for _, dataType := range dataTypes {
		switch dataType {
		case "accounts":
			fmt.Printf("  Making API call: GET /customers\n")
			accounts, err := client.api.GetAccounts()
			errorMsg := ""
			if err != nil {
				errorMsg = err.Error()
			}
			responses["responses"].(map[string]interface{})["customers"] = map[string]interface{}{
				"endpoint": "/customers",
				"method":   "GET",
				"success":  err == nil,
				"error":    errorMsg,
				"data":     accounts,
			}
		case "routes":
			fmt.Printf("  Making API call: GET /routes\n")
			routes, err := client.api.GetRoutes()
			errorMsg := ""
			if err != nil {
				errorMsg = err.Error()
			}
			responses["responses"].(map[string]interface{})["routes"] = map[string]interface{}{
				"endpoint": "/routes",
				"method":   "GET",
				"success":  err == nil,
				"error":    errorMsg,
				"data":     routes,
			}
		case "checkins":
			fmt.Printf("  Making API call: GET /appointments\n")
			checkins, err := client.api.GetCheckins()
			errorMsg := ""
			if err != nil {
				errorMsg = err.Error()
			}
			responses["responses"].(map[string]interface{})["appointments"] = map[string]interface{}{
				"endpoint": "/appointments",
				"method":   "GET",
				"success":  err == nil,
				"error":    errorMsg,
				"data":     checkins,
			}
		case "profiles":
			fmt.Printf("  Making API call: GET /profiles\n")
			profile, err := client.api.GetUserProfile()
			errorMsg := ""
			if err != nil {
				errorMsg = err.Error()
			}
			responses["responses"].(map[string]interface{})["profiles"] = map[string]interface{}{
				"endpoint": "/profiles",
				"method":   "GET",
				"success":  err == nil,
				"error":    errorMsg,
				"data":     profile,
			}
		case "all":
			// Make all API calls
			fmt.Printf("  Making API call: GET /customers\n")
			accounts, err := client.api.GetAccounts()
			responses["responses"].(map[string]interface{})["customers"] = map[string]interface{}{
				"endpoint": "/customers",
				"method":   "GET",
				"success":  err == nil,
				"error":    err,
				"data":     accounts,
			}

			fmt.Printf("  Making API call: GET /routes\n")
			routes, err := client.api.GetRoutes()
			responses["responses"].(map[string]interface{})["routes"] = map[string]interface{}{
				"endpoint": "/routes",
				"method":   "GET",
				"success":  err == nil,
				"error":    err,
				"data":     routes,
			}

			fmt.Printf("  Making API call: GET /appointments\n")
			checkins, err := client.api.GetCheckins()
			responses["responses"].(map[string]interface{})["appointments"] = map[string]interface{}{
				"endpoint": "/appointments",
				"method":   "GET",
				"success":  err == nil,
				"error":    err,
				"data":     checkins,
			}

			fmt.Printf("  Making API call: GET /profiles\n")
			profile, err := client.api.GetUserProfile()
			responses["responses"].(map[string]interface{})["profiles"] = map[string]interface{}{
				"endpoint": "/profiles",
				"method":   "GET",
				"success":  err == nil,
				"error":    err,
				"data":     profile,
			}
		}
	}

	return responses
}

// makePushAPICalls makes actual API calls for push command and returns responses
func makePushAPICalls(args []string, config *Config, client *Client) map[string]interface{} {
	responses := map[string]interface{}{
		"command":   "push",
		"args":      args,
		"responses": map[string]interface{}{},
	}

	dataTypes := getDataTypes(args)
	if len(dataTypes) == 0 {
		dataTypes = []string{"all"}
	}

	for _, dataType := range dataTypes {
		switch dataType {
		case "accounts":
			// Create a sample account for testing
			accountData := map[string]string{
				"first_name": "Test",
				"last_name":  "Account",
				"email":      "test@example.com",
				"phone":      "555-1234",
			}
			fmt.Printf("  Making API call: POST /customers\n")
			_, err := client.api.CreateAccount(accountData)
			responses["responses"].(map[string]interface{})["create_account"] = map[string]interface{}{
				"endpoint": "/customers",
				"method":   "POST",
				"success":  err == nil,
				"error":    err,
				"data":     accountData,
			}
		case "checkins":
			// Create a sample checkin for testing
			checkinData := map[string]string{
				"customer_id": "1",
				"type":        "arrival",
				"comments":    "Test checkin",
			}
			fmt.Printf("  Making API call: POST /appointments\n")
			_, err := client.api.CreateCheckin(checkinData)
			responses["responses"].(map[string]interface{})["create_checkin"] = map[string]interface{}{
				"endpoint": "/appointments",
				"method":   "POST",
				"success":  err == nil,
				"error":    err,
				"data":     checkinData,
			}
		case "all":
			// Create a sample account for testing
			accountData := map[string]string{
				"first_name": "Test",
				"last_name":  "Account",
				"email":      "test@example.com",
				"phone":      "555-1234",
			}
			fmt.Printf("  Making API call: POST /customers\n")
			_, err := client.api.CreateAccount(accountData)
			responses["responses"].(map[string]interface{})["create_account"] = map[string]interface{}{
				"endpoint": "/customers",
				"method":   "POST",
				"success":  err == nil,
				"error":    err,
				"data":     accountData,
			}

			// Create a sample checkin for testing
			checkinData := map[string]string{
				"customer_id": "1",
				"type":        "arrival",
				"comments":    "Test checkin",
			}
			fmt.Printf("  Making API call: POST /appointments\n")
			_, err = client.api.CreateCheckin(checkinData)
			responses["responses"].(map[string]interface{})["create_checkin"] = map[string]interface{}{
				"endpoint": "/appointments",
				"method":   "POST",
				"success":  err == nil,
				"error":    err,
				"data":     checkinData,
			}
		}
	}

	return responses
}

// generatePullExamples generates API call examples for pull command
func generatePullExamples(args []string, config *Config) map[string]interface{} {
	examples := map[string]interface{}{
		"command":   "pull",
		"args":      args,
		"api_calls": map[string]interface{}{},
	}

	dataTypes := getDataTypes(args)
	if len(dataTypes) == 0 {
		dataTypes = []string{"all"}
	}

	for _, dataType := range dataTypes {
		switch dataType {
		case "accounts":
			examples["api_calls"].(map[string]interface{})["get_accounts"] = map[string]interface{}{
				"method": "GET",
				"url":    fmt.Sprintf("%s/customers", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/json",
				},
				"description": "Retrieve all customer accounts",
			}
		case "routes":
			examples["api_calls"].(map[string]interface{})["get_routes"] = map[string]interface{}{
				"method": "GET",
				"url":    fmt.Sprintf("%s/routes", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/json",
				},
				"description": "Retrieve all routes",
			}
		case "checkins":
			examples["api_calls"].(map[string]interface{})["get_checkins"] = map[string]interface{}{
				"method": "GET",
				"url":    fmt.Sprintf("%s/appointments", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/json",
				},
				"description": "Retrieve all checkins/appointments",
			}
		case "profiles":
			examples["api_calls"].(map[string]interface{})["get_profile"] = map[string]interface{}{
				"method": "GET",
				"url":    fmt.Sprintf("%s/profiles", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/json",
				},
				"description": "Retrieve current user profile",
			}
		case "all":
			// Add all API calls for "all" case
			examples["api_calls"].(map[string]interface{})["get_accounts"] = map[string]interface{}{
				"method": "GET",
				"url":    fmt.Sprintf("%s/customers", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/json",
				},
				"description": "Retrieve all customer accounts",
			}
			examples["api_calls"].(map[string]interface{})["get_routes"] = map[string]interface{}{
				"method": "GET",
				"url":    fmt.Sprintf("%s/routes", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/json",
				},
				"description": "Retrieve all routes",
			}
			examples["api_calls"].(map[string]interface{})["get_checkins"] = map[string]interface{}{
				"method": "GET",
				"url":    fmt.Sprintf("%s/appointments", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/json",
				},
				"description": "Retrieve all checkins/appointments",
			}
			examples["api_calls"].(map[string]interface{})["get_profile"] = map[string]interface{}{
				"method": "GET",
				"url":    fmt.Sprintf("%s/profiles", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/json",
				},
				"description": "Retrieve current user profile",
			}
		}
	}

	return examples
}

// generatePushExamples generates API call examples for push command
func generatePushExamples(args []string, config *Config) map[string]interface{} {
	examples := map[string]interface{}{
		"command":   "push",
		"args":      args,
		"api_calls": map[string]interface{}{},
	}

	dataTypes := getDataTypes(args)
	if len(dataTypes) == 0 {
		dataTypes = []string{"all"}
	}

	for _, dataType := range dataTypes {
		switch dataType {
		case "accounts":
			examples["api_calls"].(map[string]interface{})["create_account"] = map[string]interface{}{
				"method": "POST",
				"url":    fmt.Sprintf("%s/customers", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/x-www-form-urlencoded",
				},
				"body": map[string]string{
					"last_name":     "Example Customer",
					"address":       "123 Main St, City, State, ZIP",
					"phone_number":  "555-123-4567",
					"email":         "customer@example.com",
					"account_owner": "441",
				},
				"description": "Create a new customer account",
			}
			examples["api_calls"].(map[string]interface{})["update_account"] = map[string]interface{}{
				"method": "PATCH",
				"url":    fmt.Sprintf("%s/customers/{account_id}", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/x-www-form-urlencoded",
				},
				"body": map[string]string{
					"custom_text": "Updated value",
				},
				"description": "Update an existing customer account",
			}
		case "checkins":
			examples["api_calls"].(map[string]interface{})["create_checkin"] = map[string]interface{}{
				"method": "POST",
				"url":    fmt.Sprintf("%s/appointments", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/x-www-form-urlencoded",
				},
				"body": map[string]string{
					"customer": "1981937",
					"comments": "Sales visit completed",
					"type":     "Drop-in",
				},
				"description": "Create a new checkin/appointment",
			}
		case "all":
			// Add all API calls for "all" case
			examples["api_calls"].(map[string]interface{})["create_account"] = map[string]interface{}{
				"method": "POST",
				"url":    fmt.Sprintf("%s/customers", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/x-www-form-urlencoded",
				},
				"body": map[string]string{
					"last_name":     "Example Customer",
					"address":       "123 Main St, City, State, ZIP",
					"phone_number":  "555-123-4567",
					"email":         "customer@example.com",
					"account_owner": "441",
				},
				"description": "Create a new customer account",
			}
			examples["api_calls"].(map[string]interface{})["update_account"] = map[string]interface{}{
				"method": "PATCH",
				"url":    fmt.Sprintf("%s/customers/{account_id}", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/x-www-form-urlencoded",
				},
				"body": map[string]string{
					"custom_text": "Updated value",
				},
				"description": "Update an existing customer account",
			}
			examples["api_calls"].(map[string]interface{})["create_checkin"] = map[string]interface{}{
				"method": "POST",
				"url":    fmt.Sprintf("%s/appointments", config.APIURL),
				"headers": map[string]string{
					"Authorization": "Token YOUR_API_KEY",
					"Content-Type":  "application/x-www-form-urlencoded",
				},
				"body": map[string]string{
					"customer": "1981937",
					"comments": "Sales visit completed",
					"type":     "Drop-in",
				},
				"description": "Create a new checkin/appointment",
			}
		}
	}

	return examples
}

func init() {
	// Add commands to root
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(utilsCmd)
	rootCmd.AddCommand(testCmd)

	// Add subcommands to utils
	utilsCmd.AddCommand(createTablesCmd)
	utilsCmd.AddCommand(dropTablesCmd)

	// Global flags (only config file)
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is ./config)")
}

func main() {
	// Load .env file first
	if err := loadEnvFile(); err != nil {
		fmt.Printf("Failed to load .env file: %v\n", err)
		os.Exit(1)
	}

	// Initialize logging early
	config := loadConfigFromEnv()
	if err := SetupLogging(config); err != nil {
		fmt.Printf("Failed to setup logging: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		log.Printf("Command execution failed: %v", err)
		fmt.Println(err)
		os.Exit(1)
	}
}
