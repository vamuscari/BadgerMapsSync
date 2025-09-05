package server

import (
	"badgermaps/app"
	"badgermaps/database"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleAccountCreateWebhook(t *testing.T) {
	// Create a temporary database for testing
	db, err := database.LoadDatabaseSettings("sqlite3")
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}

	// Create an app state with the temporary database
	appState := &app.State{
		DB: db,
		Config: &app.Config{
			APIKey: "test-api-key",
		},
	}

	// Create a server instance with the app state
	s := &server{App: appState}

	// Create a mock account and marshal it to JSON
	account := map[string]interface{}{
		"id":        123,
		"full_name": "Test Account",
	}
	body, _ := json.Marshal(account)

	// Create a mock HTTP request
	req, err := http.NewRequest("POST", "/webhook/account/create", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler function
	s.handleAccountCreateWebhook(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestHandleCheckinWebhook(t *testing.T) {
	// Create a temporary database for testing
	db, err := database.LoadDatabaseSettings("sqlite3")
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}

	// Create an app state with the temporary database
	appState := &app.State{
		DB: db,
		Config: &app.Config{
			APIKey: "test-api-key",
		},
	}

	// Create a server instance with the app state
	s := &server{App: appState}

	// Create a mock checkin and marshal it to JSON
	checkin := map[string]interface{}{
		"id":       456,
		"customer": 123,
	}
	body, _ := json.Marshal(checkin)

	// Create a mock HTTP request
	req, err := http.NewRequest("POST", "/webhook/checkin", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler function
	s.handleCheckinWebhook(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestMain(m *testing.M) {
	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}

	// Find the project root by looking for the "go.mod" file
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			break
		}
		if wd == filepath.Dir(wd) {
			os.Exit(1)
		}
		wd = filepath.Dir(wd)
	}

	// Change the working directory to the project root
	if err := os.Chdir(wd); err != nil {
		os.Exit(1)
	}

	// Run the tests
	exitCode := m.Run()

	// Exit with the appropriate exit code
	os.Exit(exitCode)
}
