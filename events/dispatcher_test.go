package events

import (
	"badgermaps/api"
	"badgermaps/app/state"
	"badgermaps/database"
	"os"
	"sync"
	"testing"
	"time"
)

// mockState is a simple state for testing.
var mockState = &state.State{}

// MockAppForDispatcher is a mock implementation of AppInterface for dispatcher tests.
type MockAppForDispatcher struct {
	config *Config
	state  *state.State
}

func (m *MockAppForDispatcher) GetDB() database.DB                                       { return nil }
func (m *MockAppForDispatcher) GetAPI() *api.APIClient                                     { return nil }
func (m *MockAppForDispatcher) GetConfig() *Config                                         { return m.config }
func (m *MockAppForDispatcher) GetState() *state.State                                     { return m.state }
func (m *MockAppForDispatcher) GetEventActions() []EventAction                             { return m.config.Events }
func (m *MockAppForDispatcher) RawRequest(method, endpoint string, data map[string]string) ([]byte, error) { return nil, nil }

func TestEventDispatcher_Dispatch(t *testing.T) {
	t.Run("Listener receives event", func(t *testing.T) {
		app := &MockAppForDispatcher{config: &Config{}, state: mockState}
		dispatcher := NewEventDispatcher(app)

		var wg sync.WaitGroup
		wg.Add(1)

		received := false
		listener := func(e Event) {
			if e.Type == TestEvent {
				received = true
				wg.Done()
			}
		}

		dispatcher.Subscribe(TestEvent, listener)
		dispatcher.Dispatch(Event{Type: TestEvent})

		wg.Wait()

		if !received {
			t.Error("Expected listener to receive event, but it didn't")
		}
	})

	t.Run("Action with valid config", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test_action")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		app := &MockAppForDispatcher{
			config: &Config{
				Events: []EventAction{
					{
						Name:  "TestExecAction",
						Event: "TestEvent",
						Run: []ActionConfig{
							{
								Type: "exec",
								Args: map[string]interface{}{
									"command": "echo hello > " + tmpfile.Name(),
								},
							},
						},
					},
				},
			},
			state: mockState,
		}

		dispatcher := NewEventDispatcher(app)

		dispatcher.Dispatch(Event{Type: TestEvent})

		// Wait for the action to execute
		time.Sleep(100 * time.Millisecond)

		content, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			t.Fatalf("Failed to read temp file: %v", err)
		}
		if string(content) != "hello\n" {
			t.Errorf("Expected file content 'hello\\n', got '%s'", string(content))
		}
	})

	t.Run("Action with invalid config", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		// This listener will check for the error event that should be dispatched
		errorReceived := false
		errorListener := func(e Event) {
			if e.Type == LogEvent {
				if payload, ok := e.Payload.(LogEventPayload); ok {
					if payload.Level == LogLevelError {
						errorReceived = true
						wg.Done()
					}
				}
			}
		}

		app := &MockAppForDispatcher{
			config: &Config{
				Events: []EventAction{
					{
						Name:  "TestInvalidAction",
						Event: "TestEvent",
						Run: []ActionConfig{
							{
								Type: "exec",
								Args: map[string]interface{}{
									"command": nil, // Invalid command
								},
							},
						},
					},
				},
			},
			state: mockState,
		}
		dispatcher := NewEventDispatcher(app)
		dispatcher.Subscribe(LogEvent, errorListener)

		dispatcher.Dispatch(Event{Type: TestEvent})

		wg.Wait()

		if !errorReceived {
			t.Error("Expected an error event for invalid action config, but none was received")
		}
	})
}
