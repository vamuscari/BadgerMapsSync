package server

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/app/action"
	appserver "badgermaps/app/server"
	"badgermaps/app/state"
	"badgermaps/database"
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHandleAccountCreateWebhook(t *testing.T) {
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
	app.API = api.NewAPIClient(&api.APIConfig{})

	presenter := NewCliPresenter(app)
	account := map[string]interface{}{
		"id":        123456,
		"full_name": "Test Account",
	}
	body, _ := json.Marshal(account)
	req, _ := http.NewRequest("POST", "/webhook/account/create", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	presenter.HandleAccountCreateWebhook(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleHealthCheck(t *testing.T) {
	app := app.NewApp()
	presenter := NewCliPresenter(app)

	// Test case 1: DB is connected
	app.DB = &database.SQLiteConfig{}
	app.DB.SetConnected(true)
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	presenter.HandleHealthCheck(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Test case 2: DB is not connected
	app.DB.SetConnected(false)
	req, _ = http.NewRequest("GET", "/health", nil)
	rr = httptest.NewRecorder()
	presenter.HandleHealthCheck(rr, req)
	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
	}
}

func TestHandleReplayWebhook(t *testing.T) {
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
	app.API = api.NewAPIClient(&api.APIConfig{})

	presenter := NewCliPresenter(app)

	// Log a webhook to the database
	account := map[string]interface{}{
		"id":        123456,
		"full_name": "Test Account",
	}
	body, _ := json.Marshal(account)
	headers, _ := json.Marshal(http.Header{"Content-Type": []string{"application/json"}})
	err = database.LogWebhook(db, time.Now(), "POST", "/webhook/account/create", string(headers), string(body))
	if err != nil {
		t.Fatalf("Failed to log webhook: %v", err)
	}

	// Replay the webhook
	presenter.HandleReplayWebhook(1)

	// Verify that the account was created
	row := db.GetDB().QueryRow("SELECT FullName FROM accounts WHERE AccountId = ?", 123456)
	var fullName string
	if err := row.Scan(&fullName); err != nil {
		t.Fatalf("Failed to query database for stored account: %v", err)
	}
	if fullName != "Test Account" {
		t.Errorf("Expected fullName to be 'Test Account', got '%s'", fullName)
	}
}

func TestNormalizeServerHost(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "hostname", in: "localhost", want: "localhost"},
		{name: "trim spaces", in: "  localhost  ", want: "localhost"},
		{name: "ipv4", in: "127.0.0.1", want: "127.0.0.1"},
		{name: "ipv6", in: "::1", want: "::1"},
		{name: "bracketed ipv6", in: "[::1]", want: "::1"},
		{name: "invalid bracketed host remains unchanged", in: "[localhost]", want: "[localhost]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeServerHost(tt.in); got != tt.want {
				t.Fatalf("normalizeServerHost(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeServerHostBuildsValidBracketedIPv6ListenAddress(t *testing.T) {
	addr := net.JoinHostPort(normalizeServerHost("[::1]"), "8080")
	if addr != "[::1]:8080" {
		t.Fatalf("expected normalized listen address [::1]:8080, got %q", addr)
	}
}

func TestWebhookLoggingMiddleware(t *testing.T) {
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

	// Create a mock handler to be wrapped by the middleware
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a request to test the middleware
	body := []byte(`{"key":"value"}`)
	req, _ := http.NewRequest("POST", "/test/webhook", bytes.NewBuffer(body))
	req.RequestURI = "/test/webhook"
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Wrap the mock handler with the middleware and serve the request
	WebhookLoggingMiddleware(mockHandler, app).ServeHTTP(rr, req)

	// Verify that the webhook was logged to the database
	var method, uri, headers, loggedBody string
	err = db.GetDB().QueryRow("SELECT Method, Uri, Headers, Body FROM WebhookLog WHERE Id = 1").Scan(&method, &uri, &headers, &loggedBody)
	if err != nil {
		t.Fatalf("Failed to query database for logged webhook: %v", err)
	}

	if method != "POST" {
		t.Errorf("Expected method to be 'POST', got '%s'", method)
	}
	if uri != "/test/webhook" {
		t.Errorf("Expected uri to be '/test/webhook', got '%s'", uri)
	}
	if loggedBody != `{"key":"value"}` {
		t.Errorf("Expected body to be '{\"key\":\"value\"}', got '%s'", loggedBody)
	}
}

func TestHandleInternalSyncJobsIncludesCurrentAction(t *testing.T) {
	a := app.NewApp()
	*a.State.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	presenter := NewCliPresenter(a)
	queue := appserver.NewSyncJobCoordinator(a.State, a.Events)
	defer queue.Stop()
	presenter.syncQueue = queue

	runGate := make(chan struct{})
	job, err := queue.Submit(appserver.SyncJobRequest{
		Name:   "internal-jobs-test",
		Source: "test",
		Mode:   appserver.SyncModePull,
		Run: func(_ context.Context) error {
			<-runGate
			return nil
		},
	})
	if err != nil {
		t.Fatalf("failed to queue test job: %v", err)
	}

	waitForQueueJobStatus(t, queue, job.ID, appserver.SyncJobRunning)
	queue.SetActiveJobAction("Pulling accounts")

	req := httptest.NewRequest(http.MethodGet, "/internal/jobs", nil)
	rr := httptest.NewRecorder()
	presenter.HandleInternalSyncJobs(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var snapshot appserver.SyncJobListResponse
	if err := json.NewDecoder(rr.Body).Decode(&snapshot); err != nil {
		t.Fatalf("failed to decode internal jobs response: %v", err)
	}
	if snapshot.Activity.ActiveJobAction != "Pulling accounts" {
		t.Fatalf("expected active job action in activity, got %q", snapshot.Activity.ActiveJobAction)
	}

	foundAction := ""
	for _, listed := range snapshot.Jobs {
		if listed != nil && listed.ID == job.ID {
			foundAction = listed.CurrentAction
			break
		}
	}
	if foundAction != "Pulling accounts" {
		t.Fatalf("expected queued job action in snapshot, got %q", foundAction)
	}

	close(runGate)
	waitForQueueJobStatus(t, queue, job.ID, appserver.SyncJobCompleted)
}

func TestHandleInternalScheduledJobRunMethodGuard(t *testing.T) {
	a := app.NewApp()
	presenter := NewCliPresenter(a)
	req := httptest.NewRequest(http.MethodGet, "/internal/scheduled-jobs/run", nil)
	rr := httptest.NewRecorder()

	presenter.HandleInternalScheduledJobRun(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandleInternalScheduledJobRunMissingJobID(t *testing.T) {
	a := app.NewApp()
	*a.State.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	presenter := NewCliPresenter(a)
	queue := appserver.NewSyncJobCoordinator(a.State, a.Events)
	defer queue.Stop()
	presenter.syncQueue = queue

	scheduler := appserver.NewScheduler(
		a.State,
		nil,
		nil,
		a.Events,
		nil,
		nil,
		queue,
		a.Config.WorkflowProfiles,
		a.Config.Server.Timezone,
	)
	if err := scheduler.Start(); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()
	presenter.scheduler = scheduler

	req := httptest.NewRequest(http.MethodPost, "/internal/scheduled-jobs/run", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()

	presenter.HandleInternalScheduledJobRun(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleInternalScheduledJobRunUnknownJob(t *testing.T) {
	a := app.NewApp()
	*a.State.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	presenter := NewCliPresenter(a)
	queue := appserver.NewSyncJobCoordinator(a.State, a.Events)
	defer queue.Stop()
	presenter.syncQueue = queue

	scheduler := appserver.NewScheduler(
		a.State,
		nil,
		nil,
		a.Events,
		nil,
		nil,
		queue,
		a.Config.WorkflowProfiles,
		a.Config.Server.Timezone,
	)
	if err := scheduler.Start(); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()
	presenter.scheduler = scheduler

	reqBody := bytes.NewBufferString(`{"job_id":"job_missing"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/scheduled-jobs/run", reqBody)
	rr := httptest.NewRecorder()

	presenter.HandleInternalScheduledJobRun(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandleInternalScheduledJobRunQueuesJob(t *testing.T) {
	a := app.NewApp()
	*a.State.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	jobs := map[string]*appserver.ScheduledJob{
		"job_run_now": {
			ID:       "job_run_now",
			Name:     "Run Now Test",
			Schedule: "0 0 * * * *",
			Steps: []appserver.WorkflowStep{
				{
					ID:   "action_echo",
					Type: appserver.WorkflowStepTypeAction,
					Action: action.ActionConfig{
						Type: "exec",
						Args: map[string]interface{}{
							"command": "echo run-now",
						},
					},
				},
			},
			Enabled: true,
		},
	}
	if err := appserver.SaveScheduledJobs(a.State, jobs); err != nil {
		t.Fatalf("failed to seed scheduled jobs: %v", err)
	}

	presenter := NewCliPresenter(a)
	queue := appserver.NewSyncJobCoordinator(a.State, a.Events)
	defer queue.Stop()
	presenter.syncQueue = queue

	scheduler := appserver.NewScheduler(
		a.State,
		nil,
		nil,
		a.Events,
		nil,
		nil,
		queue,
		a.Config.WorkflowProfiles,
		a.Config.Server.Timezone,
	)
	if err := scheduler.Start(); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()
	presenter.scheduler = scheduler

	reqBody := bytes.NewBufferString(`{"job_id":"job_run_now"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/scheduled-jobs/run", reqBody)
	rr := httptest.NewRecorder()

	presenter.HandleInternalScheduledJobRun(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rr.Code)
	}

	var response appserver.ScheduledJobRunResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode scheduled job run response: %v", err)
	}
	if response.JobID != "job_run_now" {
		t.Fatalf("expected job id %q, got %q", "job_run_now", response.JobID)
	}
	if response.Status != "queued" {
		t.Fatalf("expected response status %q, got %q", "queued", response.Status)
	}
}

func waitForQueueJobStatus(t *testing.T, queue *appserver.SyncJobCoordinator, jobID string, expected appserver.SyncJobStatus) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, exists := queue.GetJob(jobID)
		if exists && job.Status == expected {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s to reach status %s", jobID, expected)
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
