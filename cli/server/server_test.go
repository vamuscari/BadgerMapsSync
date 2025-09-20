package server

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/app/state"
	"badgermaps/database"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHandleAccountCreateWebhook(t *testing.T) {
	// Create a temporary directory for the test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	app := app.NewApp()

	db, err := database.NewDB(&database.DBConfig{
		Type: "sqlite3",
		Path: dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to temporary database: %v", err)
	}
	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}

	app.DB = db
	app.API = api.NewAPIClient(&api.APIConfig{})

	presenter := NewCliPresenter(app)
	account := map[string]interface{}{
		"id":        123456,
		"full_name": "Test Account",
	}
	body, _ := json.Marshal(account)
	req, _ := http.NewRequest("POST", "/webhook/account/create", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	presenter.HandleAccountCreateWebhook(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleHealthCheck(t *testing.T) {
	app := app.NewApp()
	presenter := NewCliPresenter(app)

	// Test case 1: DB is connected
	app.DB = &database.SQLiteConfig{}
	app.DB.SetConnected(true)
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	presenter.HandleHealthCheck(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Test case 2: DB is not connected
	app.DB.SetConnected(false)
	req, _ = http.NewRequest("GET", "/health", nil)
	rr = httptest.NewRecorder()
	presenter.HandleHealthCheck(rr, req)
	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
	}
}

func TestHandleReplayWebhook(t *testing.T) {
	// Create a temporary directory for the test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	app := app.NewApp()

	db, err := database.NewDB(&database.DBConfig{
		Type: "sqlite3",
		Path: dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to temporary database: %v", err)
	}
	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}

	app.DB = db
	app.API = api.NewAPIClient(&api.APIConfig{})

	presenter := NewCliPresenter(app)

	// Log a webhook to the database
	account := map[string]interface{}{
		"id":        123456,
		"full_name": "Test Account",
	}
	body, _ := json.Marshal(account)
	headers, _ := json.Marshal(http.Header{"Content-Type": []string{"application/json"}})
	err = database.LogWebhook(db, time.Now(), "POST", "/webhook/account/create", string(headers), string(body))
	if err != nil {
		t.Fatalf("Failed to log webhook: %v", err)
	}

	// Replay the webhook
	presenter.HandleReplayWebhook(1)

	// Verify that the account was created
	row := db.GetDB().QueryRow("SELECT FullName FROM accounts WHERE AccountId = ?", 123456)
	var fullName string
	if err := row.Scan(&fullName); err != nil {
		t.Fatalf("Failed to query database for stored account: %v", err)
	}
	if fullName != "Test Account" {
		t.Errorf("Expected fullName to be 'Test Account', got '%s'", fullName)
	}
}

func TestWebhookLoggingMiddleware(t *testing.T) {
	// Create a temporary directory for the test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	app := app.NewApp()

	db, err := database.NewDB(&database.DBConfig{
		Type: "sqlite3",
		Path: dbPath,
	})
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to temporary database: %v", err)
	}
	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}

	app.DB = db

	// Create a mock handler to be wrapped by the middleware
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a request to test the middleware
	body := []byte(`{"key":"value"}`)
	req, _ := http.NewRequest("POST", "/test/webhook", bytes.NewBuffer(body))
	req.RequestURI = "/test/webhook"
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Wrap the mock handler with the middleware and serve the request
	WebhookLoggingMiddleware(mockHandler, app).ServeHTTP(rr, req)

	// Verify that the webhook was logged to the database
	var method, uri, headers, loggedBody string
	err = db.GetDB().QueryRow("SELECT Method, Uri, Headers, Body FROM WebhookLog WHERE Id = 1").Scan(&method, &uri, &headers, &loggedBody)
	if err != nil {
		t.Fatalf("Failed to query database for logged webhook: %v", err)
	}

	if method != "POST" {
		t.Errorf("Expected method to be 'POST', got '%s'", method)
	}
	if uri != "/test/webhook" {
		t.Errorf("Expected uri to be '/test/webhook', got '%s'", uri)
	}
	if loggedBody != `{"key":"value"}` {
		t.Errorf("Expected body to be '{\"key\":\"value\"}', got '%s'", loggedBody)
	}
}

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			break
		}
		if wd == filepath.Dir(wd) {
			os.Exit(1)
		}
		wd = filepath.Dir(wd)
	}
	if err := os.Chdir(wd); err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}
