package push

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/app/state"
	"badgermaps/database"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer sqlDB.Close()

	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}
	_, err = sqlDB.Exec("INSERT INTO AccountsPendingChanges (AccountId, ChangeType, Changes) VALUES (?, ?, ?)", 123, "UPDATE", `{"last_name":"Test Account"}`)
	if err != nil {
		t.Fatalf("Failed to insert test pending change: %v", err)
	}
	app.DB = db
	app.API = api.NewAPIClient(&api.APIConfig{BaseURL: server.URL})

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

	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer sqlDB.Close()

	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}

	_, err = sqlDB.Exec(
		"INSERT INTO AccountCheckinsPendingChanges (CheckinId, AccountId, Type, Comments, EndpointType, ChangeType) VALUES (?, ?, ?, ?, ?, ?)",
		456,
		123,
		"Phone Call",
		"Test",
		"standard",
		"CREATE",
	)
	if err != nil {
		t.Fatalf("Failed to insert test pending change: %v", err)
	}

	app.DB = db
	app.API = api.NewAPIClient(&api.APIConfig{BaseURL: server.URL})

	cmd := PushCmd(app)
	cmd.SetArgs([]string{"checkins"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("push checkins failed with error: %v", err)
	}
}

func TestPushCheckinsCmd_CustomEndpoint(t *testing.T) {
	var postedForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/appointments/":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed reading request body: %v", err)
			}
			postedForm, err = url.ParseQuery(string(body))
			if err != nil {
				t.Fatalf("failed parsing request body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id": 789, "customer": 123, "type": "Phone Call"}`))
		case r.URL.Path == "/profiles/":
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id": 1}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id": 1}`))
		}
	}))
	defer server.Close()

	app := app.NewApp()
	app.State.NoColor = true

	dbPath := filepath.Join(t.TempDir(), "test.db")
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

	sqlDB, err := sql.Open("sqlite3", db.DatabaseConnection())
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer sqlDB.Close()

	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}

	_, err = sqlDB.Exec(
		"INSERT INTO AccountCheckinsPendingChanges (CheckinId, AccountId, Type, Comments, ExtraFields, EndpointType, ChangeType) VALUES (?, ?, ?, ?, ?, ?, ?)",
		789,
		123,
		"Phone Call",
		"Reviewed proposal",
		`{"Foo":"Bar"}`,
		"custom",
		"CREATE",
	)
	if err != nil {
		t.Fatalf("Failed to insert test pending change: %v", err)
	}

	app.DB = db
	app.API = api.NewAPIClient(&api.APIConfig{BaseURL: server.URL})

	cmd := PushCmd(app)
	cmd.SetArgs([]string{"checkins"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("push checkins failed with error: %v", err)
	}

	if postedForm == nil {
		t.Fatalf("expected custom checkin request to be posted")
	}

	rawExtraFields := postedForm.Get("extra_fields")
	if rawExtraFields == "" {
		t.Fatalf("expected extra_fields in request body")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(rawExtraFields), &decoded); err != nil {
		t.Fatalf("failed to decode extra_fields: %v", err)
	}

	if decoded["Log Type"] != "Phone Call" {
		t.Fatalf("expected Log Type to be mapped from Type, got %v", decoded["Log Type"])
	}
	if decoded["Meeting Notes"] != "Reviewed proposal" {
		t.Fatalf("expected Meeting Notes to be mapped from Comments, got %v", decoded["Meeting Notes"])
	}
	if decoded["Foo"] != "Bar" {
		t.Fatalf("expected raw extra_fields to be preserved, got %v", decoded["Foo"])
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
