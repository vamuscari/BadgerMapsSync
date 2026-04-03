package app

import (
	"badgermaps/app/server"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigRejectsInvalidServerTimezone(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	dbPath := filepath.Join(tempDir, "test.db")
	configYAML := strings.Join([]string{
		"db:",
		"  type: sqlite3",
		"  path: " + dbPath,
		"server:",
		"  timezone: Mars/Olympus",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	a := NewApp()
	a.SetConfigFilePath(configPath)
	if err := a.LoadConfig(); err == nil {
		t.Fatalf("expected timezone validation error")
	}
}

func TestLoadConfigAppliesValidServerTimezoneToState(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	dbPath := filepath.Join(tempDir, "test.db")
	configYAML := strings.Join([]string{
		"db:",
		"  type: sqlite3",
		"  path: " + dbPath,
		"server:",
		"  timezone: America/New_York",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	a := NewApp()
	a.SetConfigFilePath(configPath)
	if err := a.LoadConfig(); err != nil {
		t.Fatalf("expected valid config to load, got error: %v", err)
	}
	defer a.Close()

	if got, want := a.Config.Server.Timezone, "America/New_York"; got != want {
		t.Fatalf("expected config server timezone %q, got %q", want, got)
	}
	if got, want := a.State.ServerTimezone, "America/New_York"; got != want {
		t.Fatalf("expected state server timezone %q, got %q", want, got)
	}
}

func TestFormatTimestampInDisplayTimezoneUsesConfiguredTimezone(t *testing.T) {
	a := NewApp()
	a.Config.Server.Timezone = "America/New_York"

	ts := time.Date(2026, time.April, 2, 20, 30, 0, 0, time.UTC)
	got := a.FormatTimestampInDisplayTimezone(ts, "2006-01-02 15:04:05")
	want := "2026-04-02 16:30:00 [America/New_York]"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatTimestampInDisplayTimezoneFallsBackToLocalTimezone(t *testing.T) {
	a := NewApp()
	a.Config.Server.Timezone = ""

	ts := time.Date(2026, time.April, 2, 20, 30, 0, 0, time.UTC)
	want := ts.In(time.Local).Format(time.RFC3339) + " [" + time.Local.String() + "]"
	got := a.FormatTimestampInDisplayTimezone(ts, "")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if got, wantName := a.DisplayTimezoneName(), time.Local.String(); got != wantName {
		t.Fatalf("expected display timezone %q, got %q", wantName, got)
	}
}

func TestDateInDisplayTimezoneRespectsConfiguredTimezone(t *testing.T) {
	a := NewApp()
	a.Config.Server.Timezone = "America/Los_Angeles"

	ts := time.Date(2026, time.April, 2, 0, 30, 0, 0, time.UTC)
	if got, want := a.DateInDisplayTimezone(ts), "2026-04-01"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestScheduledJobEditsAreBlockedWhileServerRunning(t *testing.T) {
	a := NewApp()
	a.SetConfigFilePath(filepath.Join(t.TempDir(), "config.yaml"))

	jobs := map[string]*server.ScheduledJob{
		"job_1": {
			ID:       "job_1",
			Name:     "nightly",
			Schedule: "0 0 20 * * *",
			Steps: []server.WorkflowStep{
				{ID: "pull_accounts", Type: server.WorkflowStepTypeSync, SyncMode: server.SyncModePullAccounts},
			},
			Enabled: true,
		},
	}
	if err := server.SaveScheduledJobs(a.State, jobs); err != nil {
		t.Fatalf("failed to seed scheduled jobs: %v", err)
	}

	if err := os.WriteFile(a.State.PIDFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		t.Fatalf("failed to write pid file: %v", err)
	}

	if err := a.UpsertScheduledJob(&server.ScheduledJob{
		ID:       "job_2",
		Name:     "hourly",
		Schedule: "0 0 * * * *",
		Steps: []server.WorkflowStep{
			{ID: "pull_accounts", Type: server.WorkflowStepTypeSync, SyncMode: server.SyncModePullAccounts},
		},
		Enabled: true,
	}); err == nil || !strings.Contains(err.Error(), "stop the server first") {
		t.Fatalf("expected running-server guard error from upsert, got %v", err)
	}

	if err := a.SetScheduledJobEnabled("job_1", false); err == nil || !strings.Contains(err.Error(), "stop the server first") {
		t.Fatalf("expected running-server guard error from toggle, got %v", err)
	}

	if err := a.DeleteScheduledJob("job_1"); err == nil || !strings.Contains(err.Error(), "stop the server first") {
		t.Fatalf("expected running-server guard error from delete, got %v", err)
	}
}
