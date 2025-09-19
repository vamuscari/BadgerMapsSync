package pull_test

import (
	"badgermaps/api"
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
	testApp.Events.Subscribe(events.PullComplete, func(e events.Event) {
		if e.Source == "account" { // Filter for the correct event source
			pullCompleteChan <- e
		}
	})

	// Call the function to be tested
	err := pull.PullAccount(testApp, 123)
	if err != nil {
		t.Fatalf("PullAccount returned an unexpected error: %v", err)
	}

	// Verify that the PullComplete event was dispatched
	select {
	case e := <-pullCompleteChan:
		if e.Type != events.PullComplete {
			t.Errorf("Expected event type PullComplete, got %s", e.Type)
		}
		if e.Payload.(int) != 123 {
			t.Errorf("Expected payload to be 123, got %v", e.Payload)
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
	mockAccount := &api.Account{
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
