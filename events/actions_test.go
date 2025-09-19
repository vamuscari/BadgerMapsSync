package events

import (
	"badgermaps/api"
	"badgermaps/app/state"
	"badgermaps/database"
	"database/sql"
	"errors"
	"os"
	"testing"
)

// MockApp is a mock implementation of the AppInterface for testing.
type MockApp struct {
	db  *MockDB
	api *MockAPI
}

func (a *MockApp) GetState() *state.State   { return nil }
func (a *MockApp) GetConfig() *Config       { return nil }
func (a *MockApp) GetDB() database.DB       { return a.db }
func (a *MockApp) GetAPI() *api.APIClient   { return nil }
func (a *MockApp) GetEventActions() []EventAction { return nil }
func (a *MockApp) RawRequest(method, endpoint string, data map[string]string) ([]byte, error) {
	return a.api.RawRequest(method, endpoint, data)
}

// MockDB is a mock implementation of the DB interface for testing.
type MockDB struct {
	FunctionError error
}

func (db *MockDB) GetType() string { return "mock" }
func (db *MockDB) DatabaseConnection() string { return "" }
func (db *MockDB) LoadConfig(config *database.DBConfig) error { return nil }
func (db *MockDB) GetUsername() string { return "" }
func (db *MockDB) SaveConfig(config *database.DBConfig) error { return nil }
func (db *MockDB) PromptDatabaseSettings() {}
func (db *MockDB) TableExists(tableName string) (bool, error) { return true, nil }
func (db *MockDB) ViewExists(viewName string) (bool, error) { return true, nil }
func (db *MockDB) ProcedureExists(procedureName string) (bool, error) { return true, nil }
func (db *MockDB) TriggerExists(triggerName string) (bool, error) { return true, nil }
func (db *MockDB) GetTableColumns(tableName string) ([]string, error) { return nil, nil }
func (db *MockDB) ValidateSchema(s *state.State) error { return nil }
func (db *MockDB) EnforceSchema(s *state.State) error { return nil }
func (db *MockDB) TestConnection() error { return nil }
func (db *MockDB) DropAllTables() error { return nil }
func (db *MockDB) Connect() error { return nil }
func (db *MockDB) Close() error { return nil }
func (db *MockDB) GetDB() *sql.DB { return nil }
func (db *MockDB) GetSQL(command string) string { return "" }
func (db *MockDB) GetTables() ([]string, error) { return nil, nil }
func (db *MockDB) ExecuteQuery(query string) (*sql.Rows, error) { return nil, nil }
func (db *MockDB) IsConnected() bool { return true }
func (db *MockDB) SetConnected(connected bool) {}
func (db *MockDB) RunFunction(name string) error {
	if db.FunctionError != nil {
		return db.FunctionError
	}
	if name == "test_function" {
		return nil
	}
	return errors.New("function not found")
}

// MockAPI is a mock implementation of the API client for testing.
type MockAPI struct {
	RequestError error
}

func (api *MockAPI) RawRequest(method, endpoint string, data map[string]string) ([]byte, error) {
	if api.RequestError != nil {
		return nil, api.RequestError
	}
	if endpoint == "/test_endpoint" {
		return []byte("ok"), nil
	}
	return nil, errors.New("endpoint not found")
}

func TestExecAction(t *testing.T) {
	app := &MockApp{}

	// Test valid command
	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	action := &ExecAction{Command: "echo hello > " + tmpfile.Name()}
	if err := action.Execute(app); err != nil {
		t.Errorf("ExecAction failed: %v", err)
	}

	body, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "hello\n" {
		t.Errorf("ExecAction output incorrect, got: %s, want: %s", string(body), "hello\n")
	}

	// Test invalid command
	action = &ExecAction{Command: "invalid_command"}
	if err := action.Execute(app); err == nil {
		t.Errorf("ExecAction should have failed for invalid command")
	}
}

func TestDbAction(t *testing.T) {
	app := &MockApp{db: &MockDB{}}

	// Test valid function
	action := &DbAction{Function: "test_function"}
	if err := action.Execute(app); err != nil {
		t.Errorf("DbAction failed: %v", err)
	}

	// Test invalid function
	action = &DbAction{Function: "invalid_function"}
	if err := action.Execute(app); err == nil {
		t.Errorf("DbAction should have failed for invalid function")
	}

	// Test with DB error
	app.db.FunctionError = errors.New("db error")
	action = &DbAction{Function: "test_function"}
	if err := action.Execute(app); err == nil {
		t.Errorf("DbAction should have failed with DB error")
	}
}

func TestApiAction(t *testing.T) {
	app := &MockApp{api: &MockAPI{}}

	// Test valid endpoint
	action := &ApiAction{Endpoint: "/test_endpoint", Method: "GET"}
	if err := action.Execute(app); err != nil {
		t.Errorf("ApiAction failed: %v", err)
	}

	// Test invalid endpoint
	action = &ApiAction{Endpoint: "/invalid_endpoint", Method: "GET"}
	if err := action.Execute(app); err == nil {
		t.Errorf("ApiAction should have failed for invalid endpoint")
	}

	// Test with API error
	app.api.RequestError = errors.New("api error")
	action = &ApiAction{Endpoint: "/test_endpoint", Method: "GET"}
	if err := action.Execute(app); err == nil {
		t.Errorf("ApiAction should have failed with API error")
	}
}
