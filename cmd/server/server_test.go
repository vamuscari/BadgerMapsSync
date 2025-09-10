package server

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/database"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestHandleAccountCreateWebhook(t *testing.T) {
	// Create a temporary directory for the test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Set viper values for the test
	viper.Set("DB_TYPE", "sqlite3")
	viper.Set("DB_PATH", dbPath)

	app := app.NewApplication()

	db, err := database.LoadDatabaseSettings(app.State)
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}
	if err := db.EnforceSchema(); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}

	app.DB = db
	app.API = api.NewAPIClient()

	s := &server{App: app}
	account := map[string]interface{}{
		"id":        123456,
		"full_name": "Test Account",
	}
	body, _ := json.Marshal(account)
	req, _ := http.NewRequest("POST", "/webhook/account/create", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	s.handleAccountCreateWebhook(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
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
