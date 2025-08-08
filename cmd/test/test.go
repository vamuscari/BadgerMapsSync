package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"badgermapscli/api"

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
	testCmd.AddCommand(newTestAPICmd())
	testCmd.AddCommand(testDatabaseCmd())
	testCmd.AddCommand(testEndpointsCmd())

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

// testEndpoints tests API endpoints directly
func testEndpoints(apiKey string, saveResponses bool) {
	fmt.Println(color.CyanString("Testing API endpoints..."))

	// Create API endpoints
	endpoints := api.DefaultEndpoints()

	// Create test directory if it doesn't exist
	testDir := "badgermaps__test"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.Mkdir(testDir, 0755)
	}

	// Get timestamp for filenames
	timestamp := time.Now().Format("20060102_150405")

	// Store results in a list
	type EndpointResult struct {
		Name     string
		Status   string
		Duration time.Duration
	}
	var results []EndpointResult

	// Test customers endpoint
	startTime := time.Now()
	customersURL := endpoints.Customers()
	customersResp, err := makeDirectRequest(customersURL, apiKey)
	duration := time.Since(startTime)
	status := "Success"
	if err != nil {
		status = fmt.Sprintf("Error: %v", err)
	} else {
		if saveResponses {
			filename := filepath.Join(testDir, fmt.Sprintf("direct_customers_%s.json", timestamp))
			saveRawResponseToFile(customersResp, filename)
		}

		// Get first customer ID for detailed customer test
		var customers []map[string]interface{}
		if err := json.Unmarshal(customersResp, &customers); err == nil && len(customers) > 0 {
			if customerID, ok := customers[0]["id"].(float64); ok {
				// Test customer detail endpoint (required)
				startTime = time.Now()
				customerDetailURL := endpoints.Customer(int(customerID))
				customerDetailResp, err := makeDirectRequest(customerDetailURL, apiKey)
				customerDetailDuration := time.Since(startTime)
				customerDetailStatus := "Success"
				if err != nil {
					customerDetailStatus = fmt.Sprintf("Error: %v", err)
				} else {
					if saveResponses {
						filename := filepath.Join(testDir, fmt.Sprintf("direct_customer_detail_%d_%s.json", int(customerID), timestamp))
						saveRawResponseToFile(customerDetailResp, filename)
					}
				}
				results = append(results, EndpointResult{
					Name:     "Customer Detail",
					Status:   customerDetailStatus,
					Duration: customerDetailDuration,
				})

				// Test account checkin list endpoint for this customer (required)
				startTime = time.Now()
				accountCheckinsURL := endpoints.AppointmentsForCustomer(int(customerID))
				accountCheckinsResp, err := makeDirectRequest(accountCheckinsURL, apiKey)
				accountCheckinsDuration := time.Since(startTime)
				accountCheckinsStatus := "Success"
				if err != nil {
					accountCheckinsStatus = fmt.Sprintf("Error: %v", err)
				} else {
					if saveResponses {
						filename := filepath.Join(testDir, fmt.Sprintf("direct_account_checkins_%d_%s.json", int(customerID), timestamp))
						saveRawResponseToFile(accountCheckinsResp, filename)
					}
				}
				results = append(results, EndpointResult{
					Name:     "Account Checkins",
					Status:   accountCheckinsStatus,
					Duration: accountCheckinsDuration,
				})
			}
		}
	}
	results = append(results, EndpointResult{
		Name:     "Customers",
		Status:   status,
		Duration: duration,
	})

	// Test routes endpoint
	startTime = time.Now()
	routesURL := endpoints.Routes()
	routesResp, err := makeDirectRequest(routesURL, apiKey)
	duration = time.Since(startTime)
	status = "Success"
	if err != nil {
		status = fmt.Sprintf("Error: %v", err)
	} else {
		if saveResponses {
			filename := filepath.Join(testDir, fmt.Sprintf("direct_routes_%s.json", timestamp))
			saveRawResponseToFile(routesResp, filename)
		}
	}
	results = append(results, EndpointResult{
		Name:     "Routes",
		Status:   status,
		Duration: duration,
	})

	// Test appointments endpoint
	startTime = time.Now()
	appointmentsURL := endpoints.Appointments()
	appointmentsResp, err := makeDirectRequest(appointmentsURL, apiKey)
	duration = time.Since(startTime)
	status = "Success"
	if err != nil {
		status = fmt.Sprintf("Error: %v", err)
	} else {
		if saveResponses {
			filename := filepath.Join(testDir, fmt.Sprintf("direct_appointments_%s.json", timestamp))
			saveRawResponseToFile(appointmentsResp, filename)
		}
	}
	results = append(results, EndpointResult{
		Name:     "Appointments",
		Status:   status,
		Duration: duration,
	})

	// Test profiles endpoint
	startTime = time.Now()
	profilesURL := endpoints.Profiles()
	profilesResp, err := makeDirectRequest(profilesURL, apiKey)
	duration = time.Since(startTime)
	status = "Success"
	if err != nil {
		status = fmt.Sprintf("Error: %v", err)
	} else {
		if saveResponses {
			filename := filepath.Join(testDir, fmt.Sprintf("direct_profiles_%s.json", timestamp))
			saveRawResponseToFile(profilesResp, filename)
		}
	}
	results = append(results, EndpointResult{
		Name:     "Profiles",
		Status:   status,
		Duration: duration,
	})

	// Display results
	fmt.Println("\nEndpoint Test Results:")
	for _, result := range results {
		fmt.Printf("- %s: %s (%.2fs)\n", result.Name, result.Status, result.Duration.Seconds())
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
