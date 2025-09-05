package test

import (
	"badgermaps/app"
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"badgermaps/api"
	"badgermaps/database"
	"badgermaps/utils"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TestCmd creates a new test command
func TestCmd(app *app.State) *cobra.Command {
	app.VerifySetupOrExit()

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

			fmt.Println("")
			runTests(app, apiKey, saveResponses)
		},
	}

	// Add flags
	testCmd.Flags().BoolVarP(&saveResponses, "save-responses", "s", false, "Save API response bodies to text files")

	testCmd.AddCommand(testDatabaseCmd(app))
	testCmd.AddCommand(testEndpointsCmd(app))

	return testCmd
}

// runTests runs all tests
func runTests(app *app.State, apiKey string, saveResponses bool) {
	fmt.Println(color.CyanString("Running all tests..."))

	// Test Endpoint
	testEndpoints(apiKey, saveResponses)

	// Test database functionality
	testDatabase(app)

	fmt.Println(color.GreenString("All tests completed successfully"))
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

// makeDirectPostForm makes a direct POST request with application/x-www-form-urlencoded body
func makeDirectPostForm(endpoint string, apiKey string, form map[string]string) ([]byte, int, error) {
	values := url.Values{}
	for k, v := range form {
		values.Set(k, v)
	}

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", apiKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to connect to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp.StatusCode, nil
}

// testEndpointsCmd creates a command to test API endpoints directly
func testEndpointsCmd(app *app.State) *cobra.Command {
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
	results = append(results, testGetEndpoint("Profile", endpoints.Profiles(), apiKey, saveResponses))
	results = append(results, testGetEndpoint("Accounts", endpoints.Customers(), apiKey, saveResponses))
	results = append(results, testGetEndpoint("Routes", endpoints.Routes(), apiKey, saveResponses))

	// If profile fetch succeeded, attempt to create a new account via POST
	if results[0].Error == nil {
		var profile map[string]interface{}
		if err := json.Unmarshal(*results[0].Response, &profile); err == nil {
			// Expect profile has an "id" we can use as account_owner
			ownerID := 0
			if v, ok := profile["id"]; ok {
				if f, ok2 := v.(float64); ok2 {
					ownerID = int(f)
				}
			}
			if ownerID > 0 {
				form := map[string]string{
					"last_name":     fmt.Sprintf("CLI Test %s", time.Now().Format("20060102_150405")),
					"address":       "1 Test Street, Test City, FL",
					"account_owner": fmt.Sprintf("%d", ownerID),
					"email":         "example@example.com",
					"phone_number":  "555-555-5555",
				}
				res := testPostEndpoint("CreateAccount", endpoints.Customers(), apiKey, form, http.StatusCreated, saveResponses)
				results = append(results, res)

				// After creating the account, update the phone number via PATCH
				if res.Error == nil && res.Response != nil {
					var created map[string]interface{}
					if err := json.Unmarshal(*res.Response, &created); err == nil {
						newID := 0
						// Response may be {"customer": {...}} or directly the account object
						if v, ok := created["customer"]; ok {
							if m, ok2 := v.(map[string]interface{}); ok2 {
								if idv, ok3 := m["id"]; ok3 {
									if f, ok4 := idv.(float64); ok4 {
										newID = int(f)
									}
								}
							}
						} else if idv, ok := created["id"]; ok {
							if f, ok2 := idv.(float64); ok2 {
								newID = int(f)
							}
						}
						if newID > 0 {
							// 1) Update phone number via PATCH
							newPhone := fmt.Sprintf("555-000-%04d", time.Now().Unix()%10000)
							patchForm := map[string]string{"phone_number": newPhone}
							updName := fmt.Sprintf("UpdateAccountPhone_%d", newID)
							resPatch := testPatchEndpoint(updName, endpoints.Customer(newID), apiKey, patchForm, http.StatusOK, saveResponses)
							results = append(results, resPatch)

							// Only proceed if PATCH succeeded
							if resPatch.Error == nil {
								// 2) Create a check-in for this account via POST /appointments/
								checkinForm := map[string]string{
									"customer": fmt.Sprintf("%d", newID),
									"comments": fmt.Sprintf("CLI test check-in at %s", time.Now().Format(time.RFC3339)),
									"type":     "Drop-in",
								}
								resChk := testPostEndpoint(fmt.Sprintf("CreateCheckin_%d", newID), endpoints.Appointments(), apiKey, checkinForm, http.StatusCreated, saveResponses)
								results = append(results, resChk)

								// 3) Delete the test account
								resDel := testDeleteEndpoint(fmt.Sprintf("DeleteAccount_%d", newID), endpoints.Customer(newID), apiKey, http.StatusOK, saveResponses)
								results = append(results, resDel)

								// 4) Validate deletion: GET the account expecting 404
								resVal := testGetExpectStatus(fmt.Sprintf("ValidateDeletedAccount_%d", newID), endpoints.Customer(newID), apiKey, http.StatusNotFound, saveResponses)
								results = append(results, resVal)
							}
						}
					}
				}
			}
		}
	}

	if results[1].Error == nil {

		var accounts_json []map[string]interface{}

		err := json.Unmarshal(*results[1].Response, &accounts_json)

		if err != nil {
			fmt.Printf("Error unmarshalling accounts response: %v\n", err)
		} else {
			if len(accounts_json) > 0 {
				customerID := int(accounts_json[1]["id"].(float64))
				results = append(results, testGetEndpoint(
					fmt.Sprintf("AccountDetails_%d", customerID),
					endpoints.Customer(customerID),
					apiKey,
					saveResponses,
				))
				results = append(results, testGetEndpoint(
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
				results = append(results, testGetEndpoint(
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

// testGetEndpoint tests a single API endpoint directly (GET)
func testGetEndpoint(endpoint string, url string, apiKey string, saveResponses bool) EndpointResult {

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

// testPostEndpoint tests a POST request to an API endpoint
func testPostEndpoint(endpoint string, url string, apiKey string, form map[string]string, expectedStatus int, saveResponses bool) EndpointResult {
	// Ensure test directory
	testDir := "badgermaps__test"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.Mkdir(testDir, 0755)
	}

	// Timestamp for filenames
	timestamp := time.Now().Format("20060102_150405")

	start := time.Now()
	body, status, err := makeDirectPostForm(url, apiKey, form)
	duration := time.Since(start)

	statusText := "PASSED"
	if err != nil || status != expectedStatus {
		if err == nil {
			err = fmt.Errorf("unexpected status %d (expected %d)", status, expectedStatus)
		}
		statusText = fmt.Sprintf("FAILED: %v", err)
	} else if saveResponses && body != nil {
		filename := filepath.Join(testDir, fmt.Sprintf("direct_%s_%s.json", endpoint, timestamp))
		saveRawResponseToFile(body, filename)
	}

	return EndpointResult{
		Name:     endpoint,
		Status:   statusText,
		Error:    err,
		Response: &body,
		Duration: duration,
	}
}

// testPatchEndpoint tests a PATCH request to an API endpoint
func testPatchEndpoint(endpoint string, urlStr string, apiKey string, form map[string]string, expectedStatus int, saveResponses bool) EndpointResult {
	// Ensure test directory
	testDir := "badgermaps__test"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.Mkdir(testDir, 0755)
	}
	// Timestamp
	timestamp := time.Now().Format("20060102_150405")

	// Build form body
	values := url.Values{}
	for k, v := range form {
		values.Set(k, v)
	}

	req, err := http.NewRequest("PATCH", urlStr, strings.NewReader(values.Encode()))
	if err != nil {
		return EndpointResult{Name: endpoint, Status: fmt.Sprintf("FAILED: %v", err), Error: err}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", apiKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)
	if err != nil {
		return EndpointResult{Name: endpoint, Status: fmt.Sprintf("FAILED: %v", err), Error: err, Duration: duration}
	}
	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return EndpointResult{Name: endpoint, Status: fmt.Sprintf("FAILED: %v", readErr), Error: readErr, Duration: duration}
	}

	statusText := "PASSED"
	var finalErr error
	if resp.StatusCode != expectedStatus {
		finalErr = fmt.Errorf("unexpected status %d (expected %d)", resp.StatusCode, expectedStatus)
		statusText = fmt.Sprintf("FAILED: %v", finalErr)
	} else if saveResponses && body != nil {
		filename := filepath.Join(testDir, fmt.Sprintf("direct_%s_%s.json", endpoint, timestamp))
		saveRawResponseToFile(body, filename)
	}

	return EndpointResult{
		Name:     endpoint,
		Status:   statusText,
		Error:    finalErr,
		Response: &body,
		Duration: duration,
	}
}

// testDeleteEndpoint tests a DELETE request to an API endpoint
func testDeleteEndpoint(endpoint string, urlStr string, apiKey string, expectedStatus int, saveResponses bool) EndpointResult {
	// Ensure test directory
	testDir := "badgermaps__test"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.Mkdir(testDir, 0755)
	}
	// Timestamp
	timestamp := time.Now().Format("20060102_150405")

	req, err := http.NewRequest("DELETE", urlStr, nil)
	if err != nil {
		return EndpointResult{Name: endpoint, Status: fmt.Sprintf("FAILED: %v", err), Error: err}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", apiKey))

	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)
	if err != nil {
		return EndpointResult{Name: endpoint, Status: fmt.Sprintf("FAILED: %v", err), Error: err, Duration: duration}
	}
	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return EndpointResult{Name: endpoint, Status: fmt.Sprintf("FAILED: %v", readErr), Error: readErr, Duration: duration}
	}

	statusText := "PASSED"
	var finalErr error
	if resp.StatusCode != expectedStatus {
		finalErr = fmt.Errorf("unexpected status %d (expected %d)", resp.StatusCode, expectedStatus)
		statusText = fmt.Sprintf("FAILED: %v", finalErr)
	} else if saveResponses && len(body) > 0 {
		filename := filepath.Join(testDir, fmt.Sprintf("direct_%s_%s.json", endpoint, timestamp))
		saveRawResponseToFile(body, filename)
	}

	return EndpointResult{
		Name:     endpoint,
		Status:   statusText,
		Error:    finalErr,
		Response: &body,
		Duration: duration,
	}
}

// testGetExpectStatus performs a GET expecting a specific HTTP status code
func testGetExpectStatus(endpoint string, urlStr string, apiKey string, expectedStatus int, saveResponses bool) EndpointResult {
	// Ensure test directory
	testDir := "badgermaps__test"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.Mkdir(testDir, 0755)
	}
	// Timestamp
	timestamp := time.Now().Format("20060102_150405")

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return EndpointResult{Name: endpoint, Status: fmt.Sprintf("FAILED: %v", err), Error: err}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)
	if err != nil {
		return EndpointResult{Name: endpoint, Status: fmt.Sprintf("FAILED: %v", err), Error: err, Duration: duration}
	}
	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return EndpointResult{Name: endpoint, Status: fmt.Sprintf("FAILED: %v", readErr), Error: readErr, Duration: duration}
	}

	statusText := "PASSED"
	var finalErr error
	if resp.StatusCode != expectedStatus {
		finalErr = fmt.Errorf("unexpected status %d (expected %d)", resp.StatusCode, expectedStatus)
		statusText = fmt.Sprintf("FAILED: %v", finalErr)
	} else if saveResponses && len(body) > 0 {
		filename := filepath.Join(testDir, fmt.Sprintf("direct_%s_%s.json", endpoint, timestamp))
		saveRawResponseToFile(body, filename)
	}

	return EndpointResult{
		Name:     endpoint,
		Status:   statusText,
		Error:    finalErr,
		Response: &body,
		Duration: duration,
	}
}

// testDatabaseCmd creates a command to test database functionality
func testDatabaseCmd(app *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Test database functionality",
		Long:  `Test database connectivity and verify that all required tables exist with the correct schema.`,
		Run: func(cmd *cobra.Command, args []string) {
			testDatabase(app)
		},
	}

	return cmd
}

// testDatabase tests database functionality
func testDatabase(app *app.State) {
	fmt.Println(color.CyanString("Testing database..."))

	// Create test directory if it doesn't exist
	testDir := "badgermaps__test"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		os.Mkdir(testDir, 0755)
	}

	// Determine database type
	dbType := viper.GetString("DATABASE_TYPE")
	if dbType == "" {
		dbType = "sqlite3"
	}

	// Load database settings using the new interface
	db, err := database.LoadDatabaseSettings(dbType)
	if err != nil {
		fmt.Println(color.RedString("FAILED: Could not load database settings"))
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Test database connection
	fmt.Println(color.CyanString("Connecting to %s database...", db.GetType()))
	if err := db.TestConnection(); err != nil {
		fmt.Println(color.RedString("FAILED: Could not connect to database"))
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(color.GreenString("PASSED: Database connection successful"))

	// Validate database schema first
	fmt.Println(color.CyanString("\nValidating database schema..."))
	if err := db.ValidateSchema(); err != nil {
		fmt.Println(color.RedString("FAILED: Schema validation failed"))
		fmt.Printf("Error: %v\n", err)

		// Prompt user whether to enforce schema in test mode
		reader := bufio.NewReader(os.Stdin)
		shouldEnforce := utils.PromptBool(reader, "One or more required tables are missing. Enforce schema now? (y/n)", false)
		if shouldEnforce {
			fmt.Println(color.CyanString("\nEnforcing database schema (creating missing tables/indexes)..."))
			if err := db.EnforceSchema(); err != nil {
				fmt.Println(color.RedString("FAILED: Could not enforce schema"))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(color.GreenString("PASSED: Schema enforcement completed"))

			// Re-validate after enforcement
			fmt.Println(color.CyanString("\nRe-validating database schema..."))
			if err := db.ValidateSchema(); err != nil {
				fmt.Println(color.RedString("FAILED: Schema validation still failing after enforcement"))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(color.GreenString("PASSED: All required tables exist"))
		} else {
			fmt.Println(color.YellowString("User chose not to enforce schema. Exiting test."))
			os.Exit(1)
		}
	} else {
		fmt.Println(color.GreenString("PASSED: All required tables exist"))
	}

	// Show per-table existence for visibility
	maxNameLen := 0
	for _, tableName := range database.RequiredTables() {
		if len(tableName) > maxNameLen {
			maxNameLen = len(tableName)
		}
	}
	for _, tableName := range database.RequiredTables() {
		exists, err := db.TableExists(tableName)
		spacer := strings.Repeat(" ", maxNameLen-len(tableName))
		if err != nil {
			fmt.Printf("%s:%s %s\n", tableName, spacer, color.RedString("ERROR"))
			fmt.Printf("  Error: %v\n", err)
			continue
		}
		if exists {
			fmt.Printf("%s:%s %s\n", tableName, spacer, color.GreenString("EXISTS"))
		} else {
			fmt.Printf("%s:%s %s\n", tableName, spacer, color.RedString("MISSING"))
		}
	}
}
