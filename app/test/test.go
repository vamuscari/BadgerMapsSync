package test

import (
	"badgermaps/api"
	"badgermaps/api/models"
	"badgermaps/app"
	"badgermaps/events"
	"badgermaps/utils"
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type EndpointTestResult struct {
	Endpoint string
	Passed   bool
	Duration time.Duration
	Response string
	Error    error
}

// RunTests runs all tests
func RunTests(App *app.App) error {
	App.Events.Dispatch(events.Infof("test", "Running all tests..."))
	if err := TestDatabase(App); err != nil {
		return err
	}
	if err := TestApi(App, false); err != nil {
		return err
	}
	App.Events.Dispatch(events.Infof("test", "All tests completed successfully"))
	return nil
}

// TestApi tests the API
func TestApi(App *app.App, save bool) error {
	var testDir string
	var stopLogCapture func()

	if save {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		testDir = fmt.Sprintf("test-run-%s", timestamp)
		if err := os.Mkdir(testDir, 0755); err != nil {
			App.Events.Dispatch(events.Errorf("test", "FAILED: Could not create test output directory: %v", err))
			return fmt.Errorf("create test output directory: %w", err)
		}

		logFilePath := filepath.Join(testDir, "test-run.log")
		var err error
		stopLogCapture, err = startAPITestLogCapture(App, logFilePath)
		if err != nil {
			App.Events.Dispatch(events.Errorf("test", "FAILED: Could not initialize log capture: %v", err))
			return fmt.Errorf("initialize log capture: %w", err)
		}
		defer stopLogCapture()

		App.Events.Dispatch(events.Infof("test", "Saving test results to %s", testDir))
	}

	App.Events.Dispatch(events.Infof("test", "Testing API..."))
	api := App.API
	if api == nil {
		App.Events.Dispatch(events.Errorf("test", "FAILED: API client not initialized in App state"))
		return fmt.Errorf("api client not initialized in app state")
	}

	App.Events.Dispatch(events.Infof("test", "Connecting to API..."))
	start := time.Now()
	if !api.IsConnected() {
		App.Events.Dispatch(events.Errorf("test", "FAILED: Could not connect to API"))
		return fmt.Errorf("could not connect to API")
	}
	duration := time.Since(start)
	App.Events.Dispatch(events.Infof("test", "PASSED: API connection successful (%dms)", duration.Milliseconds()))

	App.Events.Dispatch(events.Infof("test", "\nTesting all API endpoints..."))
	customCheckinsEnabled := App.Config != nil && App.Config.CustomCheckins
	if customCheckinsEnabled {
		App.Events.Dispatch(events.Infof("test", "Custom checkin endpoint enabled."))
	}

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
			ID       int `json:"id"`
			Customer struct {
				ID int `json:"id"`
			} `json:"customer"`
		}
		json.Unmarshal([]byte(createResult.Response), &createdAccount)
		testAccountId = createdAccount.ID
		if testAccountId == 0 {
			testAccountId = createdAccount.Customer.ID
		}

		if testAccountId != 0 {
			detailsResult := testGetAccountDetails(App, testAccountId)
			results = append(results, detailsResult)

			updateResult := testUpdateAccount(App, testAccountId)
			results = append(results, updateResult)

			checkinResult := testCreateCheckin(App, testAccountId)
			results = append(results, checkinResult)

			if checkinResult.Passed {
				expectedCheckinID := 0
				var createdCheckin struct {
					ID int `json:"id"`
				}
				_ = json.Unmarshal([]byte(checkinResult.Response), &createdCheckin)
				expectedCheckinID = createdCheckin.ID

				checkinsListResult := testGetAccountCheckins(App, testAccountId, expectedCheckinID)
				results = append(results, checkinsListResult)
			} else {
				App.Events.Dispatch(events.Warningf("test", "get account checkins: SKIPPED (create checkin failed)"))
			}

			if customCheckinsEnabled {
				customCheckinResult := testCreateCustomCheckin(App, testAccountId)
				results = append(results, customCheckinResult)

				if customCheckinResult.Passed {
					expectedCustomCheckinID := 0
					var createdCheckin struct {
						ID int `json:"id"`
					}
					_ = json.Unmarshal([]byte(customCheckinResult.Response), &createdCheckin)
					expectedCustomCheckinID = createdCheckin.ID

					customCheckinsListResult := testGetAccountCheckins(App, testAccountId, expectedCustomCheckinID)
					customCheckinsListResult.Endpoint = "get account checkins (custom)"
					results = append(results, customCheckinsListResult)
				} else {
					App.Events.Dispatch(events.Warningf("test", "get account checkins (custom): SKIPPED (create custom checkin failed)"))
				}
			}

			deleteResult := testDeleteAccount(App, testAccountId)
			results = append(results, deleteResult)
		}
	}

	hasErrors := false
	failedEndpoints := []string{}
	for _, result := range results {
		if !result.Passed {
			hasErrors = true
			App.Events.Dispatch(events.Errorf("test", "%s: FAILED (%dms)", result.Endpoint, result.Duration.Milliseconds()))
			if result.Error != nil {
				failedEndpoints = append(failedEndpoints, fmt.Sprintf("%s (%v)", result.Endpoint, result.Error))
			} else {
				failedEndpoints = append(failedEndpoints, result.Endpoint)
			}
			if App.State.Debug {
				App.Events.Dispatch(events.Debugf("test", "Error: %v", result.Error))
				App.Events.Dispatch(events.Debugf("test", "Response: %s", result.Response))
			}
		} else {
			App.Events.Dispatch(events.Infof("test", "%s: PASSED (%dms)", result.Endpoint, result.Duration.Milliseconds()))
		}
		if save && result.Response != "" {
			responseFileName := fmt.Sprintf("%s_response.json", strings.ReplaceAll(result.Endpoint, " ", "_"))
			responseFilePath := filepath.Join(testDir, responseFileName)
			if err := os.WriteFile(responseFilePath, []byte(result.Response), 0644); err != nil {
				App.Events.Dispatch(events.Errorf("test", "FAILED: Could not save response for %s: %v", result.Endpoint, err))
			}
		}
	}

	if hasErrors {
		failureSummary := strings.Join(failedEndpoints, "; ")
		return fmt.Errorf("API endpoint tests failed: %s", failureSummary)
	}
	App.Events.Dispatch(events.Infof("test", "\nPASSED: All API endpoints responded successfully"))
	return nil
}

func startAPITestLogCapture(App *app.App, logFilePath string) (func(), error) {
	cloneState := *App.State
	cloneState.Quiet = false
	cloneState.LogFile = logFilePath

	listener, err := events.NewLogListener(&cloneState, logFilePath)
	if err != nil {
		return nil, err
	}

	var active atomic.Bool
	active.Store(true)

	App.Events.Subscribe("log", func(e events.Event) {
		if e.Type != events.EventType("log") || e.Source != "test" {
			return
		}
		if !active.Load() {
			return
		}
		listener.Handle(e)
	})

	return func() {
		if active.Swap(false) {
			listener.Close()
		}
	}, nil
}

func testCustomersEndpoint(App *app.App) EndpointTestResult {
	start := time.Now()
	result, err := App.API.GetAccounts()
	duration := time.Since(start)
	resp := ""
	if result != nil {
		resp = string(result.Raw)
	}
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
	result, err := App.API.GetRoutes()
	duration := time.Since(start)
	resp := ""
	if result != nil {
		resp = string(result.Raw)
	}
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
	result, err := App.API.GetCheckins()
	duration := time.Since(start)
	resp := ""
	if result != nil {
		resp = string(result.Raw)
	}
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
	result, err := App.API.GetUserProfile()
	duration := time.Since(start)
	resp := ""
	if result != nil {
		resp = string(result.Raw)
	}
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
	result, err := App.API.CreateAccount(models.AccountUpload{Fields: data})
	duration := time.Since(start)
	resp := ""
	if result != nil {
		resp = string(result.Raw)
	}

	passed := err == nil
	if passed {
		lastName := ""
		if result.Data.LastName.Valid {
			lastName = result.Data.LastName.String
		}
		if lastName == "" && len(resp) > 0 {
			var createdAccount struct {
				LastName string `json:"last_name"`
				Customer struct {
					LastName string `json:"last_name"`
				} `json:"customer"`
			}
			_ = json.Unmarshal([]byte(resp), &createdAccount)
			if createdAccount.LastName != "" {
				lastName = createdAccount.LastName
			} else {
				lastName = createdAccount.Customer.LastName
			}
		}
		if lastName != "Test Account" {
			passed = false
			err = fmt.Errorf("validation failed: expected last name 'Test Account', got '%s'", lastName)
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
	result, err := App.API.GetAccountDetailed(accountId)
	duration := time.Since(start)
	resp := ""
	if result != nil {
		resp = string(result.Raw)
	}

	passed := err == nil
	if passed {
		if !result.Data.AccountId.Valid || int(result.Data.AccountId.Int64) != accountId {
			passed = false
			err = fmt.Errorf("validation failed: expected account ID %d, got %d", accountId, result.Data.AccountId.Int64)
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
	result, err := App.API.UpdateAccount(accountId, models.AccountUpload{Fields: data})
	duration := time.Since(start)
	resp := ""
	if result != nil {
		resp = string(result.Raw)
	}

	passed := err == nil
	if passed {
		if !result.Data.LastName.Valid || result.Data.LastName.String != "Test Account Updated" {
			passed = false
			err = fmt.Errorf("validation failed: expected last name 'Test Account Updated', got '%s'", result.Data.LastName.String)
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
	checkinType := "test"
	data := map[string]string{
		"comments":     "Chatted with Cindy and asked how things were going and she said that they are plugging along and that she will let Dr Shockley know that I stopped by",
		"log_datetime": "2020-01-02-T13:45:01.187",
	}
	result, err := App.API.CreateCheckin(models.CheckinUpload{
		Customer: accountId,
		Type:     checkinType,
		Fields:   data,
	})
	duration := time.Since(start)
	resp := ""
	if result != nil {
		resp = string(result.Raw)
	}

	passed := err == nil
	if passed {
		checkin := result.Data
		if !checkin.CheckinId.Valid || checkin.CheckinId.Int64 == 0 {
			passed = false
			err = fmt.Errorf("validation failed: expected created checkin to include a non-zero id")
		}
		if passed && checkin.AccountId.Valid && int(checkin.AccountId.Int64) != accountId {
			passed = false
			err = fmt.Errorf("validation failed: expected customer %d, got %d", accountId, checkin.AccountId.Int64)
		}
		typeValue := strings.TrimSpace(checkin.Type.String)
		if checkin.Type.Valid && typeValue != "" && !strings.EqualFold(typeValue, checkinType) {
			passed = false
			err = fmt.Errorf("validation failed: expected type to be empty or 'test', got '%v'", checkin.Type.String)
		}
		commentsValue := strings.TrimSpace(checkin.Comments.String)
		expectedComments := strings.TrimSpace(data["comments"])
		if passed && checkin.Comments.Valid && commentsValue != "" && commentsValue != expectedComments {
			passed = false
			err = fmt.Errorf("validation failed: expected comments to be empty or match request, got '%v'", checkin.Comments.String)
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

func testCreateCustomCheckin(App *app.App, accountId int) EndpointTestResult {
	start := time.Now()
	checkinType := "Phone Call"
	data := map[string]string{
		"Meeting Notes": "Created by automated API test",
	}
	result, err := App.API.CreateCustomCheckin(models.CustomCheckinUpload{
		Customer: accountId,
		Type:     checkinType,
		Fields:   data,
	})
	duration := time.Since(start)
	resp := ""
	if result != nil {
		resp = string(result.Raw)
	}

	passed := err == nil
	if passed {
		checkin := result.Data
		if !checkin.CheckinId.Valid || checkin.CheckinId.Int64 == 0 {
			passed = false
			err = fmt.Errorf("validation failed: expected created checkin to include a non-zero id")
		}
		if passed && checkin.AccountId.Valid && int(checkin.AccountId.Int64) != accountId {
			passed = false
			err = fmt.Errorf("validation failed: expected customer %d, got %d", accountId, checkin.AccountId.Int64)
		}
	}
	return EndpointTestResult{
		Endpoint: "create custom checkin",
		Passed:   passed,
		Duration: duration,
		Response: resp,
		Error:    err,
	}
}

func testGetAccountCheckins(App *app.App, accountId int, expectedCheckinID int) EndpointTestResult {
	start := time.Now()
	var (
		checkinsResp *api.APIResponse[[]models.Checkin]
		err          error
	)

	// Account checkins can take a short moment to appear after creation.
	for attempt := 0; attempt < 3; attempt++ {
		checkinsResp, err = App.API.GetCheckinsForAccount(accountId)
		if err != nil {
			break
		}
		if expectedCheckinID == 0 {
			break
		}
		found := false
		for _, checkin := range checkinsResp.Data {
			if checkin.CheckinId.Valid && int(checkin.CheckinId.Int64) == expectedCheckinID {
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	duration := time.Since(start)
	resp := ""
	if checkinsResp != nil {
		resp = string(checkinsResp.Raw)
	}

	passed := err == nil
	if passed {
		checkins := checkinsResp.Data
		if len(checkins) == 0 {
			passed = false
			err = fmt.Errorf("validation failed: expected at least one checkin, got 0")
		} else {
			found := false
			for _, checkin := range checkins {
				if checkin.AccountId.Valid && checkin.AccountId.Int64 != int64(accountId) {
					continue
				}
				if expectedCheckinID != 0 {
					if checkin.CheckinId.Valid && int(checkin.CheckinId.Int64) == expectedCheckinID {
						found = true
						break
					}
					continue
				}
				checkinType := strings.TrimSpace(checkin.Type.String)
				if strings.EqualFold(checkinType, "test") {
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
		Response: resp,
		Error:    err,
	}
}

func testDeleteAccount(App *app.App, accountId int) EndpointTestResult {
	start := time.Now()
	err := App.API.DeleteAccount(accountId)
	duration := time.Since(start)
	resp := ""

	passed := err == nil
	if passed {
		// To validate deletion, we can try to get the account and expect a 404
		_, getErr := App.API.GetAccountDetailed(accountId)
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

// TestDatabase tests the database
func TestDatabase(App *app.App) error {
	App.Events.Dispatch(events.Infof("test", "Testing database..."))
	db := App.DB
	if db == nil {
		App.Events.Dispatch(events.Errorf("test", "FAILED: Database not initialized in App state"))
		return fmt.Errorf("database not initialized in app state")
	}

	App.Events.Dispatch(events.Infof("test", "Connecting to %s database...", db.GetType()))
	if err := db.TestConnection(); err != nil {
		App.Events.Dispatch(events.Errorf("test", "FAILED: Could not connect to database"))
		App.Events.Dispatch(events.Errorf("test", "Error: %v", err))
		return fmt.Errorf("could not connect to database: %w", err)
	}
	App.Events.Dispatch(events.Infof("test", "PASSED: Database connection successful"))

	App.Events.Dispatch(events.Infof("test", "\nValidating database schema..."))
	if err := db.ValidateSchema(App.State); err != nil {
		App.Events.Dispatch(events.Errorf("test", "FAILED: Schema validation failed"))
		if App.State.Debug {
			App.Events.Dispatch(events.Debugf("test", "Error: %v", err))
		}

		reader := bufio.NewReader(os.Stdin)
		if utils.PromptBool(reader, "Would you like to drop all tables and re-initialize the schema?", false) {
			App.Events.Dispatch(events.Warningf("test", "Resetting schema and deleting all existing data..."))
			if err := db.ResetSchema(App.State); err != nil {
				App.Events.Dispatch(events.Errorf("test", "FAILED: Could not enforce schema"))
				if App.State.Debug {
					App.Events.Dispatch(events.Debugf("test", "Error: %v", err))
				}
				return fmt.Errorf("could not enforce schema: %w", err)
			}
			App.Events.Dispatch(events.Infof("test", "Schema re-initialized successfully."))

			App.Events.Dispatch(events.Infof("test", "\nRe-validating database schema..."))
			if err := db.ValidateSchema(App.State); err != nil {
				App.Events.Dispatch(events.Errorf("test", "FAILED: Schema validation failed again after re-initialization"))
				if App.State.Debug {
					App.Events.Dispatch(events.Debugf("test", "Error: %v", err))
				}
				return fmt.Errorf("schema validation failed again after re-initialization: %w", err)
			}
		} else {
			return fmt.Errorf("schema validation failed")
		}
	}
	App.Events.Dispatch(events.Infof("test", "PASSED: All required tables exist"))
	return nil
}
