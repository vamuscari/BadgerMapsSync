package action_test

import (
	"badgermaps/api"
	"badgermaps/app/action"
	"badgermaps/app/state"
	"badgermaps/database"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestExecutor initializes a mock Executor for testing actions.
func setupTestExecutor(t *testing.T) (*action.Executor, func()) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create a temporary directory for the SQLite database
	tempDir, err := os.MkdirTemp("", "testdb_actions")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tempDir, "test.db")

	// Create a new DB for testing
	dbConfig := &database.DBConfig{Type: "sqlite3", Path: dbPath}
	db, err := database.NewDB(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Initialize the database schema
	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to db: %v", err)
	}
	// EnforceSchema needs a state, let's create a dummy one
	dummyState := &state.State{}
	if err := db.EnforceSchema(dummyState); err != nil {
		t.Fatalf("Failed to enforce schema: %v", err)
	}

	apiConfig := &api.APIConfig{BaseURL: server.URL, APIKey: "test-key"}
	apiClient := api.NewAPIClient(apiConfig)

	executor := action.NewExecutor(db, apiClient)

	// Teardown function to clean up resources
	teardown := func() {
		server.Close()
		db.Close()
		os.RemoveAll(tempDir)
	}

	return executor, teardown
}

func TestExecAction(t *testing.T) {
	executor, teardown := setupTestExecutor(t)
	defer teardown()

	// Create a temporary file to write to
	tmpfile, err := os.CreateTemp("", "test_exec_action_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	timestamp := time.Now().Format(time.RFC3339Nano)
	command := "echo " + timestamp + " > " + tmpfile.Name()

	execAction := &action.ExecAction{
		Command: command,
	}

	err = execAction.Execute(executor)
	if err != nil {
		t.Fatalf("ExecAction.Execute() failed: %v", err)
	}

	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Trim newline from echo command
	trimmedContent := strings.TrimSpace(string(content))
	if trimmedContent != timestamp {
		t.Errorf("Expected file content '%s', got '%s'", timestamp, trimmedContent)
	}
}

func TestExecActionWithArgs(t *testing.T) {
	executor, teardown := setupTestExecutor(t)
	defer teardown()

	outputFile := filepath.Join(t.TempDir(), "direct_exec_output.txt")
	payload := time.Now().Format(time.RFC3339Nano)

	execAction := &action.ExecAction{
		Command:  os.Args[0],
		Args:     []string{"-test.run=TestExecActionHelperProcess", "--", outputFile, payload},
		UseShell: boolPtr(false),
	}

	t.Setenv("GO_WANT_EXEC_ACTION_HELPER", "1")
	if err := execAction.Execute(executor); err != nil {
		t.Fatalf("ExecAction.Execute() with args failed: %v", err)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read helper output: %v", err)
	}

	if strings.TrimSpace(string(content)) != payload {
		t.Errorf("Expected helper payload '%s', got '%s'", payload, strings.TrimSpace(string(content)))
	}
}

func TestDbAction(t *testing.T) {
	executor, teardown := setupTestExecutor(t)
	defer teardown()

	// Create a test table
	_, err := executor.DB.GetDB().Exec("CREATE TABLE test_table (id INTEGER, value TEXT)")
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	timestamp := time.Now().Format(time.RFC3339Nano)
	query := "INSERT INTO test_table (id, value) VALUES (1, '" + timestamp + "')"

	dbActionConfig := action.ActionConfig{
		Type: "db",
		Args: map[string]interface{}{
			"query": query,
		},
	}

	dbAction, err := action.NewActionFromConfig(dbActionConfig)
	if err != nil {
		t.Fatalf("Failed to create DbAction: %v", err)
	}

	err = dbAction.Execute(executor)
	if err != nil {
		t.Fatalf("DbAction.Execute() failed: %v", err)
	}

	// Verify the data was inserted
	var value string
	err = executor.DB.GetDB().QueryRow("SELECT value FROM test_table WHERE id = 1").Scan(&value)
	if err != nil {
		t.Fatalf("Failed to query database for inserted value: %v", err)
	}

	if value != timestamp {
		t.Errorf("Expected database value '%s', got '%s'", timestamp, value)
	}
}

func TestExecActionHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_EXEC_ACTION_HELPER") != "1" {
		return
	}

	for i, arg := range os.Args {
		if arg == "--" && i+2 < len(os.Args) {
			output := os.Args[i+1]
			payload := os.Args[i+2]
			if err := os.WriteFile(output, []byte(payload), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write helper output: %v", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	os.Exit(1)
}

func boolPtr(v bool) *bool {
	return &v
}
