package pull_test

import (
	"badgermaps/api"
	"badgermaps/api/models"
	"badgermaps/app"
	"badgermaps/app/pull"
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/events"
	"database/sql"
	"encoding/json"
	"github.com/guregu/null/v6"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestApp initializes a mock App object for testing.
// It creates a mock HTTP server to simulate the API and an in-memory SQLite database.
func setupTestApp(t *testing.T, apiHandler http.Handler) (*app.App, func()) {
	// Create a mock server
	server := httptest.NewServer(apiHandler)

	// Create a temporary directory for the SQLite database
	tempDir, err := os.MkdirTemp("", "testdb")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tempDir, "test.db")

	// Create a new App for testing
	testState := &state.State{Verbose: true}
	dbConfig := &database.DBConfig{Type: "sqlite3", Path: dbPath}
	db, err := database.NewDB(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	apiConfig := &api.APIConfig{BaseURL: server.URL, APIKey: "test-key"}
	apiClient := api.NewAPIClient(apiConfig)

	testApp := &app.App{
		Config: &app.Config{
			DB: database.DBConfig{Type: "sqlite3", Path: dbPath},
		},
		State:  testState,
		DB:     db,
		API:    apiClient,
		Events: events.NewEventDispatcher(), // Dispatcher is tricky, might need a mock app interface
	}

	// Initialize the database schema
	if err := testApp.DB.Connect(); err != nil {
		t.Fatalf("Failed to connect to db: %v", err)
	}
	if err := testApp.DB.EnforceSchema(testApp.State); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}

	// Teardown function to clean up resources
	teardown := func() {
		server.Close()
		testApp.DB.Close()
		os.RemoveAll(tempDir)
	}

	return testApp, teardown
}

func TestPullAccount(t *testing.T) {
	// Mock API response for a detailed account
	mockAccountResponse := map[string]interface{}{
		"id":         123,
		"first_name": "John",
		"last_name":  "Doe",
		"full_name":  "John Doe",
	}

	// Create a handler for the mock API server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockAccountResponse); err != nil {
			t.Fatalf("Failed to encode mock response: %v", err)
		}
	})

	// Setup the test app
	testApp, teardown := setupTestApp(t, handler)
	defer teardown()

	// Use a channel to wait for the async event
	pullCompleteChan := make(chan events.Event)
	testApp.Events.Subscribe("pull.complete", func(e events.Event) {
		if e.Source == "account" { // Filter for the correct event source
			pullCompleteChan <- e
		}
	})

	// Call the function to be tested
	account, err := pull.PullAccount(testApp, 123)
	if err != nil {
		t.Fatalf("PullAccount returned an unexpected error: %v", err)
	}
	if account == nil {
		t.Fatal("PullAccount returned a nil account without error")
	}

	// Verify that the PullComplete event was dispatched
	select {
	case e := <-pullCompleteChan:
		if e.Type != "pull.complete" {
			t.Errorf("Expected event type pull.complete, got %s", e.Type)
		}
		payload := e.Payload.(events.CompletionPayload)
		if payload.Count != 1 {
			t.Errorf("Expected payload count to be 1, got %v", payload.Count)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for PullComplete event")
	}

	// Verify that the data was stored in the database
	row := testApp.DB.GetDB().QueryRow("SELECT FirstName, LastName FROM accounts WHERE AccountId = ?", 123)
	var firstName, lastName string
	if err := row.Scan(&firstName, &lastName); err != nil {
		t.Fatalf("Failed to query database for stored account: %v", err)
	}

	if firstName != "John" {
		t.Errorf("Expected firstName to be 'John', got '%s'", firstName)
	}
	if lastName != "Doe" {
		t.Errorf("Expected lastName to be 'Doe', got '%s'", lastName)
	}
}

func TestStoreAccountDetailed(t *testing.T) {
	// Setup a test app with an in-memory database
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	testApp, teardown := setupTestApp(t, handler)
	defer teardown()

	// Create a mock account object
	mockAccount := &models.Account{
		AccountId: null.NewInt(456, true),
		FirstName: &null.String{NullString: sql.NullString{String: "Jane", Valid: true}},
		LastName:  null.NewString("Smith", true),
		FullName:  null.NewString("Jane Smith", true),
	}

	// Call the function to be tested
	err := pull.StoreAccountDetailed(testApp, mockAccount)
	if err != nil {
		t.Fatalf("StoreAccountDetailed returned an unexpected error: %v", err)
	}

	// Verify that the data was stored in the database
	row := testApp.DB.GetDB().QueryRow("SELECT FirstName, LastName FROM accounts WHERE AccountId = ?", 456)
	var firstName, lastName string
	if err := row.Scan(&firstName, &lastName); err != nil {
		t.Fatalf("Failed to query database for stored account: %v", err)
	}

	if firstName != "Jane" {
		t.Errorf("Expected firstName to be 'Jane', got '%s'", firstName)
	}
	if lastName != "Smith" {
		t.Errorf("Expected lastName to be 'Smith', got '%s'", lastName)
	}
}

func TestStoreProfileRefreshesSQLiteAccountsWithLabels(t *testing.T) {
	testApp, teardown := setupTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer teardown()

	profile := testProfile(42, []models.DataField{
		testDataField("ct", "Customer Tier", "CustomText"),
	})
	if err := pull.StoreProfile(testApp, profile); err != nil {
		t.Fatalf("StoreProfile returned an unexpected error: %v", err)
	}

	var count int
	err := testApp.DB.GetDB().QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('AccountsWithLabels') WHERE name = ?",
		"Customer Tier",
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to inspect AccountsWithLabels: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected AccountsWithLabels to expose Customer Tier, found %d matching columns", count)
	}
}

func TestStoreProfileRefreshesSQLiteAccountsIndexed(t *testing.T) {
	testApp, teardown := setupTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer teardown()

	first := testDataField("ct2", "First Custom", "CustomText2")
	first.Position = null.IntFrom(1)
	second := testDataField("cn", "Second Custom", "CustomNumeric")
	second.Position = null.IntFrom(2)
	if err := pull.StoreProfile(testApp, testProfile(42, []models.DataField{second, first})); err != nil {
		t.Fatalf("StoreProfile returned an unexpected error: %v", err)
	}

	var customColumns string
	err := testApp.DB.GetDB().QueryRow(`
		SELECT COALESCE(group_concat(name, '|'), '')
		FROM (
			SELECT name
			FROM pragma_table_info('AccountsIndexed')
			WHERE name IN ('First Custom', 'Second Custom')
			ORDER BY cid
		)
	`).Scan(&customColumns)
	if err != nil {
		t.Fatalf("failed to inspect AccountsIndexed: %v", err)
	}
	if customColumns != "First Custom|Second Custom" {
		t.Fatalf("expected profile-ordered custom columns, got %q", customColumns)
	}

	var unusedCount int
	if err := testApp.DB.GetDB().QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('AccountsIndexed') WHERE name = ?",
		"CustomText",
	).Scan(&unusedCount); err != nil {
		t.Fatalf("failed to inspect unused AccountsIndexed columns: %v", err)
	}
	if unusedCount != 0 {
		t.Fatalf("expected AccountsIndexed to omit unused custom fields, found %d", unusedCount)
	}
}

func TestStoreProfileRollsBackDatasetReplacement(t *testing.T) {
	testApp, teardown := setupTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer teardown()

	initial := testProfile(42, []models.DataField{
		testDataField("ct", "Original Label", "CustomText"),
	})
	if err := pull.StoreProfile(testApp, initial); err != nil {
		t.Fatalf("failed to store initial profile: %v", err)
	}

	replacementField := testDataField("ct2", "Replacement Label", "CustomText2")
	replacementField.Values = []models.FieldValue{{
		Text:  null.StringFrom("invalid"),
		Value: struct{}{},
	}}
	err := pull.StoreProfile(testApp, testProfile(42, []models.DataField{replacementField}))
	if err == nil {
		t.Fatal("expected unsupported dataset value to fail")
	}

	var name, label string
	if err := testApp.DB.GetDB().QueryRow(
		"SELECT Name, Label FROM DataSets WHERE ProfileId = ?",
		42,
	).Scan(&name, &label); err != nil {
		t.Fatalf("failed to read retained dataset: %v", err)
	}
	if name != "ct" || label != "Original Label" {
		t.Fatalf("expected original dataset after rollback, got name=%q label=%q", name, label)
	}
}

func TestStoreProfileRejectsDuplicateViewLabels(t *testing.T) {
	testApp, teardown := setupTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer teardown()

	profile := testProfile(42, []models.DataField{
		testDataField("ct", "Duplicate", "CustomText"),
		testDataField("ct2", "Duplicate", "CustomText2"),
	})
	err := pull.StoreProfile(testApp, profile)
	if err == nil {
		t.Fatal("expected duplicate view labels to be rejected")
	}
	if !strings.Contains(err.Error(), "duplicate account view label") {
		t.Fatalf("expected duplicate-label error, got %v", err)
	}
}

func testProfile(id int64, datafields []models.DataField) *models.UserProfile {
	return &models.UserProfile{
		ProfileId:  null.IntFrom(id),
		Email:      null.StringFrom("owner@example.com"),
		FirstName:  null.StringFrom("Test"),
		LastName:   null.StringFrom("Owner"),
		Datafields: datafields,
		Company: models.Company{
			Id:        null.IntFrom(7),
			Name:      null.StringFrom("Example Company"),
			ShortName: null.StringFrom("EX"),
		},
	}
}

func testDataField(name, label, accountField string) models.DataField {
	return models.DataField{
		Name:         null.StringFrom(name),
		Label:        null.StringFrom(label),
		AccountField: null.StringFrom(accountField),
	}
}
