package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadConfigGeneratesAndPersistsInternalAPITokenWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	dbPath := filepath.Join(tempDir, "test.db")
	configYAML := strings.Join([]string{
		"db:",
		"  type: sqlite3",
		"  path: " + dbPath,
		"server:",
		"  host: localhost",
		"  port: 8080",
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

	if strings.TrimSpace(a.Config.Server.InternalAPIToken) == "" {
		t.Fatalf("expected generated internal API token in config")
	}
	if strings.TrimSpace(a.State.ServerInternalAPIToken) == "" {
		t.Fatalf("expected generated internal API token in state")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	var persisted struct {
		Server struct {
			InternalAPIToken string `yaml:"internal_api_token"`
		} `yaml:"server"`
	}
	if err := yaml.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("failed to parse persisted config: %v", err)
	}
	if strings.TrimSpace(persisted.Server.InternalAPIToken) == "" {
		t.Fatalf("expected persisted internal_api_token to be non-empty")
	}
}

func TestLoadConfigPreservesConfiguredInternalAPIToken(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	dbPath := filepath.Join(tempDir, "test.db")
	configYAML := strings.Join([]string{
		"db:",
		"  type: sqlite3",
		"  path: " + dbPath,
		"server:",
		"  host: localhost",
		"  port: 8080",
		"  internal_api_token: fixed-token",
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

	if got, want := a.Config.Server.InternalAPIToken, "fixed-token"; got != want {
		t.Fatalf("expected internal token %q, got %q", want, got)
	}
	if got, want := a.State.ServerInternalAPIToken, "fixed-token"; got != want {
		t.Fatalf("expected state internal token %q, got %q", want, got)
	}
}

func TestLoadConfigPersistsGeneratedInternalAPITokenWithoutExistingConfig(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	a := NewApp()
	if err := a.LoadConfig(); err != nil {
		t.Fatalf("expected config to load with defaults, got error: %v", err)
	}
	defer a.Close()

	expectedPath := filepath.Join(homeDir, ".config", "badgermaps", "config.yaml")
	if got, want := a.ConfigFile, expectedPath; got != want {
		t.Fatalf("expected generated config path %q, got %q", want, got)
	}

	if strings.TrimSpace(a.Config.Server.InternalAPIToken) == "" {
		t.Fatal("expected generated internal API token in config")
	}
	if strings.TrimSpace(a.State.ServerInternalAPIToken) == "" {
		t.Fatal("expected generated internal API token in state")
	}

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read generated config file: %v", err)
	}
	var persisted struct {
		Server struct {
			InternalAPIToken string `yaml:"internal_api_token"`
		} `yaml:"server"`
	}
	if err := yaml.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("failed to parse generated config: %v", err)
	}
	if strings.TrimSpace(persisted.Server.InternalAPIToken) == "" {
		t.Fatal("expected generated config to persist internal_api_token")
	}
}
