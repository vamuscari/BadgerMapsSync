package events

import (
	"badgermaps/api"
	"badgermaps/app/state"
	"badgermaps/database"
	"database/sql"
	"testing"
	"time"
)

// MockApp is a mock implementation of the AppInterface for testing.
type MockApp struct {
	MockDB       *MockDB
	MockAPI      *MockAPI
	AppConfig    *Config
	AppState     *state.State
	EventActions []EventAction
}

func (a *MockApp) GetDB() database.DB {
	return a.MockDB
}

func (a *MockApp) GetAPI() *api.APIClient {
	if a.MockAPI == nil {
		return nil
	}
	return &a.MockAPI.APIClient
}

func (a *MockApp) GetConfig() *Config {
	return a.AppConfig
}

func (a *MockApp) GetState() *state.State {
	return a.AppState
}

func (a *MockApp) GetRawFromAPI(endpoint string) ([]byte, error) {
	if a.MockAPI == nil {
		return nil, nil
	}
	s, err := a.MockAPI.GetRaw(endpoint)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func (a *MockApp) GetEventActions() []EventAction {
	return a.EventActions
}

// MockDB is a mock implementation of the database.DB interface.
type MockDB struct {
	FunctionCalls map[string]int
}

func (db *MockDB) GetType() string {
	return "mock"
}

func (db *MockDB) DatabaseConnection() string {
	return ""
}

func (db *MockDB) LoadConfig(config *database.DBConfig) error {
	return nil
}

func (db *MockDB) GetUsername() string {
	return ""
}

func (db *MockDB) SaveConfig(config *database.DBConfig) error {
	return nil
}

func (db *MockDB) PromptDatabaseSettings() {}

func (db *MockDB) TableExists(tableName string) (bool, error) {
	return true, nil
}

func (db *MockDB) ViewExists(viewName string) (bool, error) {
	return true, nil
}

func (db *MockDB) ProcedureExists(procedureName string) (bool, error) {
	return true, nil
}

func (db *MockDB) TriggerExists(triggerName string) (bool, error) {
	return true, nil
}

func (db *MockDB) GetTableColumns(tableName string) ([]string, error) {
	return []string{"col1", "col2"}, nil
}

func (db *MockDB) ValidateSchema(s *state.State) error {
	return nil
}

func (db *MockDB) EnforceSchema(s *state.State) error {
	return nil
}

func (db *MockDB) TestConnection() error {
	return nil
}

func (db *MockDB) DropAllTables() error {
	return nil
}

func (db *MockDB) Connect() error {
	return nil
}

func (db *MockDB) Close() error {
	return nil
}

func (db *MockDB) GetDB() *sql.DB {
	return nil
}

func (db *MockDB) GetSQL(command string) string {
	return ""
}

func (db *MockDB) RunFunction(name string) error {
	if db.FunctionCalls == nil {
		db.FunctionCalls = make(map[string]int)
	}
	db.FunctionCalls[name]++
	return nil
}

func (db *MockDB) GetTables() ([]string, error) {
	return []string{"table1", "table2"}, nil
}

func (db *MockDB) ExecuteQuery(query string) (*sql.Rows, error) {
	return nil, nil
}

func (db *MockDB) IsConnected() bool {
	return true
}

func (db *MockDB) SetConnected(b bool) {
	// Do nothing for mock
}

// MockAPI is a mock implementation of the API interface.
type MockAPI struct {
	api.APIClient
	GetRawCalls map[string]int
}

func (api *MockAPI) GetRaw(endpoint string) (string, error) {
	if api.GetRawCalls == nil {
		api.GetRawCalls = make(map[string]int)
	}
	api.GetRawCalls[endpoint]++
	return "mock data", nil
}

func TestNewActionFromConfig(t *testing.T) {
	// Test case for exec action
	execConfig := ActionConfig{
		Type: "exec",
		Args: map[string]interface{}{
			"command": "echo",
			"args":    []interface{}{"hello"},
		},
	}
	action, err := NewActionFromConfig(execConfig)
	if err != nil {
		t.Fatalf("failed to create exec action: %v", err)
	}
	if _, ok := action.(*ExecAction); !ok {
		t.Errorf("expected ExecAction, got %T", action)
	}

	// Test case for db action
	dbConfig := ActionConfig{
		Type: "db",
		Args: map[string]interface{}{
			"function": "my_function",
		},
	}
	action, err = NewActionFromConfig(dbConfig)
	if err != nil {
		t.Fatalf("failed to create db action: %v", err)
	}
	if _, ok := action.(*DbAction); !ok {
		t.Errorf("expected DbAction, got %T", action)
	}

	// Test case for api action
	apiConfig := ActionConfig{
		Type: "api",
		Args: map[string]interface{}{
			"endpoint": "my_endpoint",
		},
	}
	action, err = NewActionFromConfig(apiConfig)
	if err != nil {
		t.Fatalf("failed to create api action: %v", err)
	}
	if _, ok := action.(*ApiAction); !ok {
		t.Errorf("expected ApiAction, got %T", action)
	}

	// Test case for unknown action
	unknownConfig := ActionConfig{
		Type: "unknown",
	}
	_, err = NewActionFromConfig(unknownConfig)
	if err == nil {
		t.Errorf("expected error for unknown action type, got nil")
	}
}

func TestExecAction_Execute(t *testing.T) {
	action := &ExecAction{
		Command: "echo",
		Args:    []string{"hello"},
	}
	err := action.Execute(nil)
	if err != nil {
		t.Errorf("unexpected error executing command: %v", err)
	}
}

func TestDbAction_Execute(t *testing.T) {
	mockDB := &MockDB{}
	app := &MockApp{MockDB: mockDB, AppConfig: &Config{}}
	action := &DbAction{
		Function: "my_function",
	}
	err := action.Execute(app)
	if err != nil {
		t.Errorf("unexpected error executing db function: %v", err)
	}
	if mockDB.FunctionCalls["my_function"] != 1 {
		t.Errorf("expected function to be called once, got %d", mockDB.FunctionCalls["my_function"])
	}
}

func TestApiAction_Execute(t *testing.T) {
	mockAPI := &MockAPI{}
	app := &MockApp{MockAPI: mockAPI, AppConfig: &Config{}}
	action := &ApiAction{
		Endpoint: "my_endpoint",
	}
	err := action.Execute(app)
	if err != nil {
		t.Errorf("unexpected error executing api call: %v", err)
	}
	if mockAPI.GetRawCalls["my_endpoint"] != 1 {
		t.Errorf("expected api to be called once, got %d", mockAPI.GetRawCalls["my_endpoint"])
	}
}

func TestAction_Validate(t *testing.T) {
	// Test ExecAction
	execAction := &ExecAction{}
	if err := execAction.Validate(); err == nil {
		t.Error("expected error for exec action with no command, got nil")
	}
	execAction.Command = "echo"
	if err := execAction.Validate(); err != nil {
		t.Errorf("unexpected error for valid exec action: %v", err)
	}

	// Test DbAction
	dbAction := &DbAction{}
	if err := dbAction.Validate(); err == nil {
		t.Error("expected error for db action with no function, got nil")
	}
	dbAction.Function = "my_func"
	if err := dbAction.Validate(); err != nil {
		t.Errorf("unexpected error for valid db action: %v", err)
	}

	// Test ApiAction
	apiAction := &ApiAction{}
	if err := apiAction.Validate(); err == nil {
		t.Error("expected error for api action with no endpoint, got nil")
	}
	apiAction.Endpoint = "my_endpoint"
	if err := apiAction.Validate(); err != nil {
		t.Errorf("unexpected error for valid api action: %v", err)
	}
}

func TestEventDispatcher_Dispatch(t *testing.T) {
	t.Run("Matching event and source", func(t *testing.T) {
		mockDB := &MockDB{}
		mockAPI := &MockAPI{}
		app := &MockApp{
			MockDB:  mockDB,
			MockAPI: mockAPI,
			AppConfig: &Config{
				Events: []EventAction{
					{
						Name:   "test_event",
						Event:  "PullComplete",
						Source: "accounts",
						Run: []ActionConfig{
							{
								Type: "db",
								Args: map[string]interface{}{"function": "my_function"},
							},
							{
								Type: "api",
								Args: map[string]interface{}{"endpoint": "my_endpoint"},
							},
						},
					},
				},
			},
			AppState: &state.State{},
		}

		dispatcher := NewEventDispatcher(app)
		event := Event{
			Type:   PullComplete,
			Source: "accounts",
		}
		dispatcher.Dispatch(event)

		time.Sleep(100 * time.Millisecond)

		if mockDB.FunctionCalls["my_function"] != 1 {
			t.Errorf("expected db function to be called once, got %d", mockDB.FunctionCalls["my_function"])
		}
		if mockAPI.GetRawCalls["my_endpoint"] != 1 {
			t.Errorf("expected api to be called once, got %d", mockAPI.GetRawCalls["my_endpoint"])
		}
	})

	t.Run("Event with no matching action", func(t *testing.T) {
		mockDB := &MockDB{}
		app := &MockApp{
			MockDB: mockDB,
			AppConfig: &Config{
				Events: []EventAction{
					{
						Name:  "test_event",
						Event: "PushComplete",
						Run:   []ActionConfig{{Type: "db", Args: map[string]interface{}{"function": "my_function"}}},
					},
				},
			},
			AppState: &state.State{},
		}

		dispatcher := NewEventDispatcher(app)
		event := Event{Type: PullComplete}
		dispatcher.Dispatch(event)

		time.Sleep(100 * time.Millisecond)

		if len(mockDB.FunctionCalls) != 0 {
			t.Errorf("expected no function calls, got %d", len(mockDB.FunctionCalls))
		}
	})

	t.Run("Action with source that does not match", func(t *testing.T) {
		mockDB := &MockDB{}
		app := &MockApp{
			MockDB: mockDB,
			AppConfig: &Config{
				Events: []EventAction{
					{
						Name:   "test_event",
						Event:  "PullComplete",
						Source: "accounts",
						Run:    []ActionConfig{{Type: "db", Args: map[string]interface{}{"function": "my_function"}}},
					},
				},
			},
			AppState: &state.State{},
		}

		dispatcher := NewEventDispatcher(app)
		event := Event{Type: PullComplete, Source: "checkins"}
		dispatcher.Dispatch(event)

		time.Sleep(100 * time.Millisecond)

		if len(mockDB.FunctionCalls) != 0 {
			t.Errorf("expected no function calls, got %d", len(mockDB.FunctionCalls))
		}
	})

	t.Run("Action with invalid config", func(t *testing.T) {
		var actionError error
		app := &MockApp{
			AppConfig: &Config{
				Events: []EventAction{
					{
						Name:  "test_event",
						Event: "PullComplete",
						Run:   []ActionConfig{{Type: "db", Args: map[string]interface{}{"bad_arg": "my_function"}}},
					},
				},
			},
			AppState: &state.State{},
		}

		dispatcher := NewEventDispatcher(app)
		dispatcher.Subscribe(ActionError, func(e Event) {
			actionError = e.Payload.(error)
		})
		event := Event{Type: PullComplete}
		dispatcher.Dispatch(event)

		time.Sleep(100 * time.Millisecond)

		if actionError == nil {
			t.Error("expected action error, got nil")
		}
	})
}