package push

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/database"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPushAccountsCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 123, "full_name": "Test Account"}`))
	}))
	defer server.Close()

	app := app.NewApp()
	app.State.NoColor = true

	dbPath := filepath.Join(t.TempDir(), "test.db")
	os.Setenv("DB_TYPE", "sqlite3")
	os.Setenv("DB_PATH", dbPath)

	db, err := database.LoadDatabaseSettings(app.State)
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}

	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer sqlDB.Close()

	if err := db.EnforceSchema(); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}

	_, err = sqlDB.Exec("INSERT INTO AccountsPendingChanges (AccountId, ChangeType, Changes) VALUES (?, ?, ?)", 123, "UPDATE", `{"last_name":"Test Account"}`)
	if err != nil {
		t.Fatalf("Failed to insert test pending change: %v", err)
	}

	app.DB = db
	app.API = api.NewAPIClient()

	cmd := PushCmd(app)
	cmd.SetArgs([]string{"accounts"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("push accounts failed with error: %v", err)
	}
}

func TestPushCheckinsCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 456, "customer": 123}`))
	}))
	defer server.Close()

	app := app.NewApp()
	app.State.NoColor = true

	dbPath := filepath.Join(t.TempDir(), "test.db")
	os.Setenv("DB_TYPE", "sqlite3")
	os.Setenv("DB_PATH", dbPath)

	db, err := database.LoadDatabaseSettings(app.State)
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}

	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer sqlDB.Close()

	if err := db.EnforceSchema(); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}

	_, err = sqlDB.Exec("INSERT INTO AccountCheckinsPendingChanges (CheckinId, AccountId, ChangeType, Changes) VALUES (?, ?, ?, ?)", 456, 123, "CREATE", `{"customer":123,"comments":"Test"}`)
	if err != nil {
		t.Fatalf("Failed to insert test pending change: %v", err)
	}

	app.DB = db
	app.API = api.NewAPIClient()

	cmd := PushCmd(app)
	cmd.SetArgs([]string{"checkins"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("push checkins failed with error: %v", err)
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
