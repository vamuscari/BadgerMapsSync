package app

import (
	"badgermaps/app/server"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigRejectsLegacyEventActions(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	dbPath := filepath.Join(tempDir, "test.db")
	configYAML := strings.Join([]string{
		"db:",
		"  type: sqlite3",
		"  path: " + dbPath,
		"event_actions:",
		"  - name: on_pull_complete",
		"    event: pull.complete",
		"    source: accounts",
		"    run:",
		"      - type: exec",
		"        args:",
		"          command: echo hi",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	a := NewApp()
	a.SetConfigFilePath(configPath)
	err := a.LoadConfig()
	if err == nil {
		t.Fatalf("expected legacy event_actions validation failure")
	}
	if !strings.Contains(err.Error(), "event_actions") {
		t.Fatalf("expected event_actions error, got %v", err)
	}
}

func TestLoadConfigSeedsDefaultWorkflowProfilesWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	dbPath := filepath.Join(tempDir, "test.db")
	configYAML := strings.Join([]string{
		"db:",
		"  type: sqlite3",
		"  path: " + dbPath,
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	a := NewApp()
	a.SetConfigFilePath(configPath)
	if err := a.LoadConfig(); err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}
	defer a.Close()

	if _, ok := a.Config.WorkflowProfiles["pull_all"]; !ok {
		t.Fatalf("expected pull_all workflow profile to be seeded")
	}
	if _, ok := a.Config.WorkflowProfiles["push_all"]; !ok {
		t.Fatalf("expected push_all workflow profile to be seeded")
	}
	if _, ok := a.Config.WorkflowProfiles["pull_push"]; !ok {
		t.Fatalf("expected pull_push workflow profile to be seeded")
	}

	pullAll := a.Config.WorkflowProfiles["pull_all"]
	if got, want := len(pullAll.Steps), 4; got != want {
		t.Fatalf("expected pull_all to have %d steps, got %d", want, got)
	}
	wantModes := []server.SyncMode{
		server.SyncModePullAccounts,
		server.SyncModePullCheckins,
		server.SyncModePullRoutes,
		server.SyncModePullProfile,
	}
	for i, wantMode := range wantModes {
		if gotMode := pullAll.Steps[i].SyncMode; gotMode != wantMode {
			t.Fatalf("expected pull_all step %d mode %q, got %q", i, wantMode, gotMode)
		}
	}
}
