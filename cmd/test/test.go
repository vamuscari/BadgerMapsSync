package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"badgermapscli/api"
	"badgermapscli/database"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TestCmd creates a new test command
func TestCmd() *cobra.Command {
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
	testCmd.AddCommand(newAPICmd())
	testCmd.AddCommand(testDatabaseCmd())
	testCmd.AddCommand(testEndpointsCmd())

	return testCmd
}

// newAPICmd creates a command to test API connectivity
func newAPICmd() *cobra.Command {
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

// testDatabaseCmd creates a command to test database functionality
func testDatabaseCmd() *cobra.Command {
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

	maxNameLen := 0
	for ep := range results {
		if len(ep) > maxNameLen {
			maxNameLen = len(ep)
		}
	}

	// Check results
	allPassed := true
	for endpoint, err := range results {
		spacer := strings.Repeat(" ", maxNameLen-len(endpoint))
		if err != nil {
			fmt.Printf("%s:%s %s\n", endpoint, spacer, color.RedString("FAILED"))
			fmt.Printf("  Error: %v\n", err)
			allPassed = false
		} else {
			fmt.Printf("%s:%s %s\n", endpoint, spacer, color.GreenString("PASSED"))
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

		// Get and save details for the first account if available
		if len(accounts) > 0 {
			fmt.Println(color.CyanString("Getting details for the first account (ID: %d)...", accounts[0].ID))
			accountDetails, err := apiClient.GetAccount(accounts[0].ID)
			if err == nil {
				// Display account details
				fmt.Println(color.GreenString("First account details:"))
				fmt.Printf("ID: %d\n", accountDetails.ID)

				// Handle FirstName which is a pointer
				firstName := ""
				if accountDetails.FirstName != nil {
					firstName = *accountDetails.FirstName
				}
				fmt.Printf("Name: %s %s\n", firstName, accountDetails.LastName)

				if accountDetails.FullName != "" {
					fmt.Printf("Full Name: %s\n", accountDetails.FullName)
				}
				if accountDetails.Email != "" {
					fmt.Printf("Email: %s\n", accountDetails.Email)
				}
				if accountDetails.PhoneNumber != "" {
					fmt.Printf("Phone: %s\n", accountDetails.PhoneNumber)
				}

				// Save detailed account to file
				saveResponseToFile(accountDetails, filepath.Join(testDir, fmt.Sprintf("account_details_%d_%s.json", accountDetails.ID, timestamp)))
			} else {
				fmt.Printf("Error getting details for account %d: %v\n", accounts[0].ID, err)
			}
		}
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

// saveResponseToFile saves a response to a file as properly formatted JSON
func saveResponseToFile(response interface{}, filename string) {
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		fmt.Printf("Error serializing response to JSON: %v\n", err)
		return
	}

	err = ioutil.WriteFile(filename, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error saving response to %s: %v\n", filename, err)
	} else {
		fmt.Printf("Response saved to %s\n", filename)
	}
}

// saveRawResponseToFile saves raw response bytes to a file
func saveRawResponseToFile(response []byte, filename string) {
	// Try to format the JSON for better readability
	var jsonObj interface{}
	err := json.Unmarshal(response, &jsonObj)
	if err == nil {
		// If it's valid JSON, pretty print it
		formattedJSON, err := json.MarshalIndent(jsonObj, "", "  ")
		if err == nil {
			response = formattedJSON
		}
	}

	err = ioutil.WriteFile(filename, response, 0644)
	if err != nil {
		fmt.Printf("Error saving response to %s: %v\n", filename, err)
	} else {
		fmt.Printf("Response saved to %s\n", filename)
	}
}

// makeDirectRequest makes a direct HTTP request to the given URL with the API key
func makeDirectRequest(url string, apiKey string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// testEndpointsCmd creates a command to test API endpoints directly
func testEndpointsCmd() *cobra.Command {
	var (
		saveResponses bool
		apiKey        string
	)

	cmd := &cobra.Command{
		Use:   "endpoints",
		Short: "Test API endpoints directly",
		Long:  `Test API endpoints directly without using the API service layer, outputting the exact response.`,
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

			testEndpoints(apiKey, saveResponses)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&saveResponses, "save-responses", "s", false, "Save API response bodies to text files")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "BadgerMaps API key (default is from env)")

	return cmd
}

// Store results in a list
type EndpointResult struct {
	Name     string
	Status   string
	Response *[]byte
	Error    error
	Duration time.Duration
}

// testEndpoints tests API endpoints directly
func testEndpoints(apiKey string, saveResponses bool) {
	fmt.Println(color.CyanString("Testing API endpoints..."))

	// Create test directory if it doesn't exist
	testDir := "badgermaps__test"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.Mkdir(testDir, 0755)
	}

	var results []EndpointResult

	endpoints := api.DefaultEndpoints()
	// Test profile
	results = append(results, testEndpoint("Profile", endpoints.Profiles(), apiKey, saveResponses))
	results = append(results, testEndpoint("Accounts", endpoints.Customers(), apiKey, saveResponses))
	results = append(results, testEndpoint("Routes", endpoints.Routes(), apiKey, saveResponses))

	if results[1].Error == nil {

		var accounts_json []map[string]interface{}

		err := json.Unmarshal(*results[1].Response, &accounts_json)

		if err != nil {
			fmt.Printf("Error unmarshalling accounts response: %v\n", err)
		} else {
			if len(accounts_json) > 0 {
				customerID := int(accounts_json[1]["id"].(float64))
				results = append(results, testEndpoint(
					fmt.Sprintf("AccountDetails_%d", customerID),
					endpoints.Customer(customerID),
					apiKey,
					saveResponses,
				))
				results = append(results, testEndpoint(
					fmt.Sprintf("Checkins_%d", customerID),
					endpoints.AppointmentsForCustomer(customerID),
					apiKey,
					saveResponses,
				))
			}
		}
	}

	if results[2].Error == nil {
		var routes []map[string]interface{}
		err := json.Unmarshal(*results[2].Response, &routes)
		if err != nil {
			fmt.Printf("Error unmarshalling routes response: %v\n", err)
		} else {
			if len(routes) > 0 {
				routeID := int(routes[0]["id"].(float64))
				results = append(results, testEndpoint(
					"Route Detail",
					endpoints.Route(routeID),
					apiKey,
					saveResponses,
				))
			}
		}
	}

	maxNameLen := 0
	for _, ep := range results {
		if len(ep.Name) > maxNameLen {
			maxNameLen = len(ep.Name)
		}
	}

	// Check results
	for _, ep := range results {
		spacer := strings.Repeat(" ", maxNameLen-len(ep.Name))
		if ep.Error != nil {
			fmt.Printf("%s:%s %s (%.2fs)\n", ep.Name, spacer, color.RedString(ep.Status), ep.Duration.Seconds())
			fmt.Printf("  Error: %v\n", ep.Error)
		} else {
			fmt.Printf("%s:%s %s (%.2fs)\n", ep.Name, spacer, color.GreenString(ep.Status), ep.Duration.Seconds())
		}
	}
}

// testEndpoint tests a single API endpoint directly
func testEndpoint(endpoint string, url string, apiKey string, saveResponses bool) EndpointResult {

	// Create test directory if it doesn't exist
	testDir := "badgermaps__test"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.Mkdir(testDir, 0755)
	}

	// Get timestamp for filenames
	timestamp := time.Now().Format("20060102_150405")

	// Test the endpoint
	startTime := time.Now()
	resp, err := makeDirectRequest(url, apiKey)
	duration := time.Since(startTime)

	status := "PASSED"
	if err != nil {
		status = fmt.Sprintf("FAILED: %v", err)
	} else if saveResponses {
		filename := filepath.Join(testDir, fmt.Sprintf("direct_%s_%s.json", endpoint, timestamp))
		saveRawResponseToFile(resp, filename)
	}

	return EndpointResult{
		Name:     endpoint,
		Status:   status,
		Error:    err,
		Response: &resp,
		Duration: duration,
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

	// Get database configuration from viper
	dbType := viper.GetString("DATABASE_TYPE")
	if dbType == "" {
		dbType = "sqlite3" // Default to SQLite
	}

	// Create database config
	dbConfig := &database.Config{
		DatabaseType: dbType,
		Host:         viper.GetString("DATABASE_HOST"),
		Port:         viper.GetString("DATABASE_PORT"),
		Database:     viper.GetString("DATABASE_NAME"),
		Username:     viper.GetString("DATABASE_USER"),
		Password:     viper.GetString("DATABASE_PASSWORD"),
	}

	// For SQLite, use a test database if no path is specified
	if dbType == "sqlite3" && dbConfig.Database == "" {
		dbConfig.Database = "test.db"
	}

	// Test database connection
	fmt.Println(color.CyanString("Connecting to %s database...", dbType))
	client, err := database.NewClient(dbConfig, true)
	if err != nil {
		fmt.Println(color.RedString("FAILED: Could not connect to database"))
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println(color.GreenString("PASSED: Database connection successful"))

	// Validate database schema
	fmt.Println(color.CyanString("\nValidating database tables:"))

	// Get required tables
	requiredTables := client.GetRequiredTables()

	// Calculate the longest table name for alignment
	maxNameLen := 0
	for tableName := range requiredTables {
		if len(tableName) > maxNameLen {
			maxNameLen = len(tableName)
		}
	}

	// Check each table individually
	allTablesValid := true

	for tableName, _ := range requiredTables {
		spacer := strings.Repeat(" ", maxNameLen-len(tableName))

		// Try to query the table to see if it exists and is accessible
		_, err := client.GetDB().Query(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", tableName))
		if err != nil {
			fmt.Printf("%s:%s %s\n", tableName, spacer, color.RedString("FAILED"))
			fmt.Printf("  Error: %v\n", err)
			allTablesValid = false
		} else {
			fmt.Printf("%s:%s %s\n", tableName, spacer, color.GreenString("PASSED"))
		}
	}

	if allTablesValid {
		fmt.Println(color.GreenString("\nAll database tables validated successfully"))
	} else {
		fmt.Println(color.RedString("\nSome database tables failed validation"))
		fmt.Println("Run 'badgermaps utils create-tables' to create missing tables")
	}
}
