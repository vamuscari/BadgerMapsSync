package push

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/database"
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// MockRoundTripper is a mock implementation of http.RoundTripper for testing purposes.
type MockRoundTripper struct {
	Response *http.Response
	Err      error
}

// RoundTrip is the mock implementation of the http.RoundTripper interface.
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.Response, m.Err
}

func TestPushAccountCmd(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Provide a mock response for the UpdateAccount API endpoint
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 123, "full_name": "Test Account"}`))
	}))
	defer server.Close()

	// Create a temporary database for testing
	db, err := database.LoadDatabaseSettings("sqlite3")
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}

	// Create an app state with the mock server's URL and the temporary database
	appState := &app.State{
		DB: db,
		Config: &app.Config{
			APIKey: "test-api-key",
		},
	}

	// Create a custom HTTP client with the mock round tripper
	httpClient := &http.Client{
		Transport: &MockRoundTripper{
			Response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"id": 123, "full_name": "Test Account"}`)),
				Header:     make(http.Header),
			},
		},
	}

	// Create a new API client with the mock http client
	apiClient := api.NewAPIClientWithClient("test-api-key", server.URL, httpClient)

	// Create the push account command
	cmd := pushAccountCmd(appState, apiClient)

	// Execute the command with a test account ID
	cmd.SetArgs([]string{"123"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pushAccountCmd() failed with error: %v", err)
	}
}

func TestMain(m *testing.M) {
	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}

	// Find the project root by looking for the "go.mod" file
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			break
		}
		if wd == filepath.Dir(wd) {
			os.Exit(1)
		}
		wd = filepath.Dir(wd)
	}

	// Change the working directory to the project root
	if err := os.Chdir(wd); err != nil {
		os.Exit(1)
	}

	// Run the tests
	exitCode := m.Run()

	// Exit with the appropriate exit code
	os.Exit(exitCode)
}
