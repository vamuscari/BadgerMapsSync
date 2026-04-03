package server

import (
	"badgermaps/app/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateScheduledJobIDUsesShortAlphanumericCode(t *testing.T) {
	id := GenerateScheduledJobID()
	if !strings.HasPrefix(id, scheduledJobIDPrefix) {
		t.Fatalf("expected prefix %q, got %q", scheduledJobIDPrefix, id)
	}

	code := strings.TrimPrefix(id, scheduledJobIDPrefix)
	if len(code) != scheduledJobIDLength {
		t.Fatalf("expected code length %d, got %d (%q)", scheduledJobIDLength, len(code), code)
	}

	for _, r := range code {
		if !strings.ContainsRune(scheduledJobIDAlphabet, r) {
			t.Fatalf("unexpected id character %q in %q", r, code)
		}
	}
}

func TestSaveScheduledJobsDropsLegacyWorkflowProfileField(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	st := state.NewState()
	*st.ConfigFile = configPath

	jobsFile := filepath.Join(tempDir, "scheduled_jobs.json")
	legacyPayload := `{
  "job_legacy01": {
    "id": "job_legacy01",
    "name": "Legacy Profile Job",
    "schedule": "0 0 * * * *",
    "workflow_profile": "pull_all",
    "steps": [
      {
        "id": "pull_accounts",
        "type": "sync",
        "sync_type": "pull_accounts"
      }
    ],
    "enabled": true,
    "run_count": 0,
    "error_count": 0,
    "retry_on_error": false,
    "max_retries": 1
  }
}`
	if err := os.WriteFile(jobsFile, []byte(legacyPayload), 0644); err != nil {
		t.Fatalf("failed to write legacy jobs file: %v", err)
	}

	jobs, err := LoadScheduledJobs(st)
	if err != nil {
		t.Fatalf("failed to load scheduled jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one job, got %d", len(jobs))
	}

	if err := SaveScheduledJobs(st, jobs); err != nil {
		t.Fatalf("failed to save scheduled jobs: %v", err)
	}

	updatedBytes, err := os.ReadFile(jobsFile)
	if err != nil {
		t.Fatalf("failed to read rewritten jobs file: %v", err)
	}
	updated := string(updatedBytes)
	if strings.Contains(updated, "\"workflow_profile\"") {
		t.Fatalf("expected workflow_profile to be omitted from saved jobs, got: %s", updated)
	}
}
