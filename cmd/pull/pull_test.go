package pull

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/app/state"
	"badgermaps/database"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPullAccountCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 123, "full_name": "Test Account"})
	}))
	defer server.Close()

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
	app.API = api.NewAPIClient(&api.APIConfig{BaseURL: server.URL})

	cmd := pullAccountCmd(app)
	cmd.SetArgs([]string{"123"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pullAccountCmd() failed with error: %v", err)
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
