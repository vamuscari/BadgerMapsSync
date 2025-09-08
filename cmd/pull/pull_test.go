package pull

import (
	"badgermaps/api"
	"badgermaps/app"
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

	app := app.NewApplication()

	db, err := database.LoadDatabaseSettings(app.State)
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}

	app.DB = db
	app.API = api.NewAPIClient("test-api-key", server.URL)

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
