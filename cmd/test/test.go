package test

import (
	"badgermaps/app"
	"badgermaps/utils"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type EndpointTestResult struct {
	Endpoint string
	Passed   bool
	Duration time.Duration
	Response string
	Error    error
}

// TestCmd creates a new test command
func TestCmd(App *app.App) *cobra.Command {
	App.VerifySetupOrExit()

	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Run tests and diagnostics",
		Long:  `Test the BadgerMaps CLI functionality, including API connectivity and database functionality.`,
		Run: func(cmd *cobra.Command, args []string) {
			runTests(App)
		},
	}

	testCmd.AddCommand(testDatabaseCmd(App))
	testCmd.AddCommand(testApiCmd(App))
	return testCmd
}

func runTests(App *app.App) {
	fmt.Println(color.CyanString("Running all tests..."))
	testDatabase(App)
	testApi(App, false)
	fmt.Println(color.GreenString("All tests completed successfully"))
}

func testDatabaseCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Test database functionality",
		Long:  `Test database connectivity and verify that all required tables exist with the correct schema.`,
		Run: func(cmd *cobra.Command, args []string) {
			testDatabase(App)
		},
	}
	return cmd
}

func testApiCmd(App *app.App) *cobra.Command {
	var save bool
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Test API functionality",
		Long:  `Test API connectivity and verify that all endpoints are responding correctly.`,
		Run: func(cmd *cobra.Command, args []string) {
			testApi(App, save)
		},
	}
	cmd.Flags().BoolVarP(&save, "save", "s", false, "Save test output to a log file and separate files for each endpoint response")
	return cmd
}

func testApi(App *app.App, save bool) {
	var output io.Writer = os.Stdout
	var testDir string

	if save {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		testDir = fmt.Sprintf("test-run-%s", timestamp)
		if err := os.Mkdir(testDir, 0755); err != nil {
			fmt.Println(color.RedString("FAILED: Could not create test output directory: %v", err))
			os.Exit(1)
		}

		logFilePath := filepath.Join(testDir, "test-run.log")
		logFile, err := os.Create(logFilePath)
		if err != nil {
			fmt.Println(color.RedString("FAILED: Could not create log file: %v", err))
			os.Exit(1)
		}
		defer logFile.Close()

		output = io.MultiWriter(os.Stdout, logFile)
		fmt.Fprintf(output, "Saving test results to %s\n", testDir)
	}

	fmt.Fprintln(output, color.CyanString("Testing API..."))
	api := App.API
	if api == nil {
		fmt.Fprintln(output, color.RedString("FAILED: API client not initialized in App state"))
		os.Exit(1)
	}

	fmt.Fprintln(output, color.CyanString("Connecting to API..."))
	start := time.Now()
	if err := api.TestAPIConnection(); err != nil {
		fmt.Fprintln(output, color.RedString("FAILED: Could not connect to API"))
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	duration := time.Since(start)
	fmt.Fprintln(output, color.GreenString("PASSED: API connection successful (%dms)", duration.Milliseconds()))

	fmt.Fprintln(output, color.CyanString("\nTesting all API endpoints..."))

	results := []EndpointTestResult{}

	// GET endpoints
	getTests := []func(App *app.App) EndpointTestResult{
		testCustomersEndpoint,
		testRoutesEndpoint,
		testAppointmentsEndpoint,
		testProfilesEndpoint,
	}
	for _, t := range getTests {
		results = append(results, t(App))
	}

	// Account lifecycle tests
	var testAccountId int
	createResult := testCreateAccount(App)
	results = append(results, createResult)
	if createResult.Passed {
		var createdAccount struct {
			Customer struct {
				ID int `json:"id"`
			}
		}
		json.Unmarshal([]byte(createResult.Response), &createdAccount)
		testAccountId = createdAccount.Customer.ID

		if testAccountId != 0 {
			detailsResult := testGetAccountDetails(App, testAccountId)
			results = append(results, detailsResult)

			updateResult := testUpdateAccount(App, testAccountId)
			results = append(results, updateResult)

			checkinResult := testCreateCheckin(App, testAccountId)
			results = append(results, checkinResult)

			if checkinResult.Passed {
				checkinsListResult := testGetAccountCheckins(App, testAccountId)
				results = append(results, checkinsListResult)
			}

			deleteResult := testDeleteAccount(App, testAccountId)
			results = append(results, deleteResult)
		}
	}

	hasErrors := false
	for _, result := range results {
		if !result.Passed {
			hasErrors = true
			fmt.Fprintf(output, "%s: %s (%dms)\n", result.Endpoint, color.RedString("FAILED"), result.Duration.Milliseconds())
			if App.State.Debug {
				fmt.Fprintf(os.Stderr, "Error: %v\n", result.Error)
				fmt.Fprintf(os.Stderr, "Response: %s\n", result.Response)
			}
		} else {
			fmt.Fprintf(output, "%s: %s (%dms)\n", result.Endpoint, color.GreenString("PASSED"), result.Duration.Milliseconds())
		}
		if save && result.Response != "" {
			responseFileName := fmt.Sprintf("%s_response.json", strings.ReplaceAll(result.Endpoint, " ", "_"))
			responseFilePath := filepath.Join(testDir, responseFileName)
			if err := os.WriteFile(responseFilePath, []byte(result.Response), 0644); err != nil {
				fmt.Fprintf(output, color.RedString("FAILED: Could not save response for %s: %v\n", result.Endpoint, err))
			}
		}
	}

	if hasErrors {
		fmt.Fprintln(output, color.RedString("\nSome API endpoint tests failed."))
		os.Exit(1)
	}
	fmt.Fprintln(output, color.GreenString("\nPASSED: All API endpoints responded successfully"))
}

func testCustomersEndpoint(App *app.App) EndpointTestResult {
	start := time.Now()
	resp, err := App.API.GetRaw("customers")
	duration := time.Since(start)
	return EndpointTestResult{
		Endpoint: "get customers",
		Passed:   err == nil,
		Duration: duration,
		Response: resp,
		Error:    err,
	}
}

func testRoutesEndpoint(App *app.App) EndpointTestResult {
	start := time.Now()
	resp, err := App.API.GetRaw("routes")
	duration := time.Since(start)
	return EndpointTestResult{
		Endpoint: "get routes",
		Passed:   err == nil,
		Duration: duration,
		Response: resp,
		Error:    err,
	}
}

func testAppointmentsEndpoint(App *app.App) EndpointTestResult {
	start := time.Now()
	resp, err := App.API.GetRaw("appointments")
	duration := time.Since(start)
	return EndpointTestResult{
		Endpoint: "get appointments",
		Passed:   err == nil,
		Duration: duration,
		Response: resp,
		Error:    err,
	}
}

func testProfilesEndpoint(App *app.App) EndpointTestResult {
	start := time.Now()
	resp, err := App.API.GetRaw("profiles")
	duration := time.Since(start)
	return EndpointTestResult{
		Endpoint: "get profiles",
		Passed:   err == nil,
		Duration: duration,
		Response: resp,
		Error:    err,
	}
}

func testCreateAccount(App *app.App) EndpointTestResult {
	start := time.Now()
	data := map[string]string{
		"last_name":     "Test Account",
		"address":       "123 Test St, Test City, TS 12345",
		"email":         "test@example.com",
		"phone_number":  "",
		"account_owner": strconv.Itoa(App.API.UserID),
	}
	resp, err := App.API.PostRaw("customers", data)
	duration := time.Since(start)

	passed := err == nil
	if passed {
		var createdAccount struct {
			Customer struct {
				LastName string `json:"last_name"`
			}
		}
		json.Unmarshal([]byte(resp), &createdAccount)
		if createdAccount.Customer.LastName != "Test Account" {
			passed = false
			err = fmt.Errorf("validation failed: expected last name 'Test Account', got '%s'", createdAccount.Customer.LastName)
		}
	}

	return EndpointTestResult{
		Endpoint: "create account",
		Passed:   passed,
		Duration: duration,
		Response: resp,
		Error:    err,
	}
}

func testGetAccountDetails(App *app.App, accountId int) EndpointTestResult {
	start := time.Now()
	endpoint := "customers/" + strconv.Itoa(accountId)
	resp, err := App.API.GetRaw(endpoint)
	duration := time.Since(start)

	passed := err == nil
	if passed {
		var fetchedAccount struct {
			ID int `json:"id"`
		}
		json.Unmarshal([]byte(resp), &fetchedAccount)
		if fetchedAccount.ID != accountId {
			passed = false
			err = fmt.Errorf("validation failed: expected account ID %d, got %d", accountId, fetchedAccount.ID)
		}
	}

	return EndpointTestResult{
		Endpoint: "get account details",
		Passed:   passed,
		Duration: duration,
		Response: resp,
		Error:    err,
	}
}

func testUpdateAccount(App *app.App, accountId int) EndpointTestResult {
	start := time.Now()
	data := map[string]string{
		"last_name": "Test Account Updated",
	}
	resp, err := App.API.PatchRaw("customers/"+strconv.Itoa(accountId), data)
	duration := time.Since(start)

	passed := err == nil
	if passed {
		var updatedAccount struct {
			LastName string `json:"last_name"`
		}
		json.Unmarshal([]byte(resp), &updatedAccount)
		if updatedAccount.LastName != "Test Account Updated" {
			passed = false
			err = fmt.Errorf("validation failed: expected last name 'Test Account Updated', got '%s'", updatedAccount.LastName)
		}
	}

	return EndpointTestResult{
		Endpoint: "update account",
		Passed:   passed,
		Duration: duration,
		Response: resp,
		Error:    err,
	}
}

func testCreateCheckin(App *app.App, accountId int) EndpointTestResult {
	start := time.Now()
	data := map[string]string{
		"customer": strconv.Itoa(accountId),
		"comments": "Test checkin",
		"type":     "Drop-in",
	}
	resp, err := App.API.PostRaw("appointments", data)
	duration := time.Since(start)

	passed := err == nil
	if passed {
		var createdCheckin struct {
			Comments string `json:"comments"`
		}
		json.Unmarshal([]byte(resp), &createdCheckin)
		if createdCheckin.Comments != "Test checkin" {
			passed = false
			err = fmt.Errorf("validation failed: expected comments 'Test checkin', got '%s'", createdCheckin.Comments)
		}
	}

	return EndpointTestResult{
		Endpoint: "create checkin",
		Passed:   passed,
		Duration: duration,
		Response: resp,
		Error:    err,
	}
}

func testGetAccountCheckins(App *app.App, accountId int) EndpointTestResult {
	start := time.Now()
	checkins, err := App.API.GetCheckinsForAccount(accountId)
	duration := time.Since(start)

	passed := err == nil
	if passed {
		if len(checkins) == 0 {
			passed = false
			err = fmt.Errorf("validation failed: expected at least one checkin, got 0")
		} else {
			found := false
			for _, checkin := range checkins {
				if checkin.Comments.String == "Test checkin" && checkin.AccountId.Int64 == int64(accountId) {
					found = true
					break
				}
			}
			if !found {
				passed = false
				err = fmt.Errorf("validation failed: created checkin not found in list")
			}
		}
	}

	return EndpointTestResult{
		Endpoint: "get account checkins",
		Passed:   passed,
		Duration: duration,
		Response: "", // No raw response to save
		Error:    err,
	}
}

func testDeleteAccount(App *app.App, accountId int) EndpointTestResult {
	start := time.Now()
	resp, err := App.API.DeleteRaw("customers/" + strconv.Itoa(accountId))
	duration := time.Since(start)

	passed := err == nil
	if passed {
		// To validate deletion, we can try to get the account and expect a 404
		_, getErr := App.API.GetRaw("customers/" + strconv.Itoa(accountId))
		if getErr == nil || !strings.Contains(getErr.Error(), "404") {
			passed = false
			err = fmt.Errorf("validation failed: account still exists after deletion")
		}
	}

	return EndpointTestResult{
		Endpoint: "delete account",
		Passed:   passed,
		Duration: duration,
		Response: resp,
		Error:    err,
	}
}

func testDatabase(App *app.App) {
	fmt.Println(color.CyanString("Testing database..."))
	db := App.DB
	if db == nil {
		fmt.Println(color.RedString("FAILED: Database not initialized in App state"))
		os.Exit(1)
	}

	fmt.Println(color.CyanString("Connecting to %s database...", db.GetType()))
	if err := db.TestConnection(); err != nil {
		fmt.Println(color.RedString("FAILED: Could not connect to database"))
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(color.GreenString("PASSED: Database connection successful"))

	fmt.Println(color.CyanString("\nValidating database schema..."))
	if err := db.ValidateSchema(); err != nil {
		fmt.Println(color.RedString("FAILED: Schema validation failed"))
		if App.State.Debug {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		reader := bufio.NewReader(os.Stdin)
		if utils.PromptBool(reader, "Would you like to drop all tables and re-initialize the schema?", false) {
			fmt.Println(color.YellowString("Dropping all tables..."))
			if err := db.DropAllTables(); err != nil {
				fmt.Println(color.RedString("FAILED: Could not drop tables"))
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(color.GreenString("Tables dropped successfully."))

			fmt.Println(color.CyanString("Re-initializing schema..."))
			if err := db.EnforceSchema(); err != nil {
				fmt.Println(color.RedString("FAILED: Could not enforce schema"))
				if App.State.Debug {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
				os.Exit(1)
			}
			fmt.Println(color.GreenString("Schema re-initialized successfully."))

			fmt.Println(color.CyanString("\nRe-validating database schema..."))
			if err := db.ValidateSchema(); err != nil {
				fmt.Println(color.RedString("FAILED: Schema validation failed again after re-initialization"))
				if App.State.Debug {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
				os.Exit(1)
			}
		} else {
			os.Exit(1)
		}
	}
	fmt.Println(color.GreenString("PASSED: All required tables exist"))
}
