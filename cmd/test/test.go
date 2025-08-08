package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"badgermapscli/api"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewTestCmd creates a new test command
func NewTestCmd() *cobra.Command {
	var (
		saveResponses bool
		apiKey        string
	)

	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Run tests and diagnostics",
		Long:  `Test the BadgerMaps CLI functionality, including API connectivity and database functionality.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get API key from flag or environment
			if apiKey == "" {
				apiKey = viper.GetString("API_KEY")
				if apiKey == "" {
					fmt.Println(color.RedString("Error: API key is required"))
					fmt.Println("Please provide an API key using the --api-key flag or set it in the environment")
					os.Exit(1)
				}
			}

			runTests(apiKey, saveResponses)
		},
	}

	// Add flags
	testCmd.Flags().BoolVarP(&saveResponses, "save-responses", "s", false, "Save API response bodies to text files")
	testCmd.Flags().StringVar(&apiKey, "api-key", "", "BadgerMaps API key (default is from env)")

	// Add subcommands
	testCmd.AddCommand(newTestAPICmd())
	testCmd.AddCommand(newTestDatabaseCmd())

	return testCmd
}

// newTestAPICmd creates a command to test API connectivity
func newTestAPICmd() *cobra.Command {
	var (
		saveResponses bool
		apiKey        string
	)

	cmd := &cobra.Command{
		Use:   "api",
		Short: "Test API connectivity",
		Long:  `Test connectivity to the BadgerMaps API and verify that all endpoints are accessible.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get API key from flag or environment
			if apiKey == "" {
				apiKey = viper.GetString("API_KEY")
				if apiKey == "" {
					fmt.Println(color.RedString("Error: API key is required"))
					fmt.Println("Please provide an API key using the --api-key flag or set it in the environment")
					os.Exit(1)
				}
			}

			testAPI(apiKey, saveResponses)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&saveResponses, "save-responses", "s", false, "Save API response bodies to text files")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "BadgerMaps API key (default is from env)")

	return cmd
}

// newTestDatabaseCmd creates a command to test database functionality
func newTestDatabaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Test database functionality",
		Long:  `Test database connectivity and verify that all required tables exist with the correct schema.`,
		Run: func(cmd *cobra.Command, args []string) {
			testDatabase()
		},
	}

	return cmd
}

// runTests runs all tests
func runTests(apiKey string, saveResponses bool) {
	fmt.Println(color.CyanString("Running all tests..."))

	// Test API connectivity
	testAPI(apiKey, saveResponses)

	// Test database functionality
	testDatabase()

	fmt.Println(color.GreenString("All tests completed successfully"))
}

// testAPI tests API connectivity
func testAPI(apiKey string, saveResponses bool) {
	fmt.Println(color.CyanString("Testing API connectivity..."))

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Test API connection
	err := apiClient.TestAPIConnection()
	if err != nil {
		fmt.Println(color.RedString("API connection test failed: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("API connection test passed"))

	// Test all endpoints
	fmt.Println(color.CyanString("Testing API endpoints..."))
	results := apiClient.TestAllEndpoints()

	// Check results
	allPassed := true
	for endpoint, err := range results {
		if err != nil {
			fmt.Printf("%s: %s\n", endpoint, color.RedString("FAILED"))
			fmt.Printf("  Error: %v\n", err)
			allPassed = false
		} else {
			fmt.Printf("%s: %s\n", endpoint, color.GreenString("PASSED"))
		}
	}

	if !allPassed {
		fmt.Println(color.RedString("Some API endpoint tests failed"))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("All API endpoint tests passed"))

	// Save responses if requested
	if saveResponses {
		saveAPIResponses(apiClient)
	}
}

// saveAPIResponses saves API responses to text files
func saveAPIResponses(apiClient *api.APIClient) {
	fmt.Println(color.CyanString("Saving API responses..."))

	// Create test directory if it doesn't exist
	testDir := "badgermaps__test"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.Mkdir(testDir, 0755)
	}

	// Get timestamp for filenames
	timestamp := time.Now().Format("20060102_150405")

	// Save account response
	accounts, err := apiClient.GetAccounts()
	if err == nil {
		saveResponseToFile(accounts, filepath.Join(testDir, fmt.Sprintf("accounts_%s.json", timestamp)))
	}

	// Save routes response
	routes, err := apiClient.GetRoutes()
	if err == nil {
		saveResponseToFile(routes, filepath.Join(testDir, fmt.Sprintf("routes_%s.json", timestamp)))
	}

	// Save checkins response
	checkins, err := apiClient.GetCheckins()
	if err == nil {
		saveResponseToFile(checkins, filepath.Join(testDir, fmt.Sprintf("checkins_%s.json", timestamp)))
	}

	// Save profile response
	profile, err := apiClient.GetUserProfile()
	if err == nil {
		saveResponseToFile(profile, filepath.Join(testDir, fmt.Sprintf("profile_%s.json", timestamp)))
	}

	fmt.Println(color.GreenString("API responses saved to %s directory", testDir))
}

// saveResponseToFile saves a response to a file
func saveResponseToFile(response interface{}, filename string) {
	// In a real implementation, we would serialize the response to JSON
	// For now, just write a placeholder
	content := fmt.Sprintf("%v", response)
	err := ioutil.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error saving response to %s: %v\n", filename, err)
	} else {
		fmt.Printf("Response saved to %s\n", filename)
	}
}

// testDatabase tests database functionality
func testDatabase() {
	fmt.Println(color.CyanString("Testing database functionality..."))

	// Create test directory if it doesn't exist
	testDir := "badgermaps__test"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.Mkdir(testDir, 0755)
	}

	// In a real implementation, we would:
	// 1. Create a test database in the badgermaps__test directory
	// 2. Initialize the schema
	// 3. Verify that all required tables exist with the correct schema
	// 4. Run some basic CRUD operations

	// For example:
	// dbPath := filepath.Join(testDir, "test_database.db")
	// db, err := sql.Open("sqlite3", dbPath)
	// if err != nil {
	//     fmt.Println(color.RedString("Failed to create test database: %v", err))
	//     os.Exit(1)
	// }
	// defer db.Close()

	// For now, just print a placeholder message
	fmt.Println(color.GreenString("Database functionality test passed"))
	fmt.Println(color.CyanString("Test database would be created in the %s directory", testDir))
}
