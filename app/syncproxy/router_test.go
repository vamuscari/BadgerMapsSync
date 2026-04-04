package syncproxy

import (
	"badgermaps/app"
	"badgermaps/app/action"
	appserver "badgermaps/app/server"
	"badgermaps/app/state"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServerBaseURLNormalizesWildcardHostsToLoopback(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "empty host", host: "", want: "http://127.0.0.1:8080"},
		{name: "ipv4 wildcard", host: "0.0.0.0", want: "http://127.0.0.1:8080"},
		{name: "ipv6 wildcard", host: "::", want: "http://127.0.0.1:8080"},
		{name: "asterisk wildcard", host: "*", want: "http://127.0.0.1:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &app.App{State: state.NewState()}
			a.State.ServerHost = tt.host
			a.State.ServerPort = 8080

			if got := serverBaseURL(a); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestServerBaseURLFormatsIPv6LiteralAndTLS(t *testing.T) {
	a := &app.App{State: state.NewState()}
	a.State.ServerHost = "::1"
	a.State.ServerPort = 9443
	a.State.TLSEnabled = true

	if got, want := serverBaseURL(a), "https://[::1]:9443"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRunScheduledJobNowRejectsEmptyJobID(t *testing.T) {
	a := app.NewApp()
	err := RunScheduledJobNow(a, "   ")
	if err == nil {
		t.Fatal("expected error for empty job id")
	}
	if !strings.Contains(err.Error(), "job id is required") {
		t.Fatalf("expected job id validation error, got %v", err)
	}
}

func TestRunScheduledJobNowQueuesLocallyWhenServerStopped(t *testing.T) {
	tempDir := t.TempDir()
	a := app.NewApp()
	a.SetConfigFilePath(filepath.Join(tempDir, "config.yaml"))

	jobs := map[string]*appserver.ScheduledJob{
		"job_run_local": {
			ID:       "job_run_local",
			Name:     "Run Local",
			Schedule: "0 0 * * * *",
			Steps: []appserver.WorkflowStep{
				{
					ID:   "action_echo",
					Type: appserver.WorkflowStepTypeAction,
					Action: action.ActionConfig{
						Type: "exec",
						Args: map[string]interface{}{
							"command": "echo local-run",
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

	if err := RunScheduledJobNow(a, "job_run_local"); err != nil {
		t.Fatalf("expected local scheduled job queueing to succeed, got %v", err)
	}

	queue := a.GetSyncCoordinator()
	if queue == nil {
		t.Fatal("expected local sync queue to be initialized")
	}
	defer queue.Stop()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(queue.ListJobs()) > 0 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("expected at least one queued local job after RunScheduledJobNow")
}

func TestRunScheduledJobNowReturnsErrorForUnknownLocalJob(t *testing.T) {
	tempDir := t.TempDir()
	a := app.NewApp()
	a.SetConfigFilePath(filepath.Join(tempDir, "config.yaml"))

	err := RunScheduledJobNow(a, "job_missing")
	if err == nil {
		t.Fatal("expected error when scheduled job does not exist")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "job not found") {
		t.Fatalf("expected job not found error, got %v", err)
	}
}

func TestApplyInternalAuthHeaderRejectsMissingToken(t *testing.T) {
	a := app.NewApp()
	a.Config.Server.InternalAPIToken = ""
	a.State.ServerInternalAPIToken = ""

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/internal/jobs", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	err = applyInternalAuthHeader(req, a)
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if !strings.Contains(err.Error(), "internal_api_token") {
		t.Fatalf("expected internal_api_token guidance, got %v", err)
	}
}

func TestApplyInternalAuthHeaderSetsBearerToken(t *testing.T) {
	a := app.NewApp()
	a.Config.Server.InternalAPIToken = "abc123"
	a.State.ServerInternalAPIToken = "abc123"

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/internal/jobs", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	if err := applyInternalAuthHeader(req, a); err != nil {
		t.Fatalf("expected token header to be set, got %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "Bearer abc123" {
		t.Fatalf("expected Authorization header %q, got %q", "Bearer abc123", got)
	}
}
