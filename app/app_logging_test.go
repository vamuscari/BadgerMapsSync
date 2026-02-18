package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultLogFilePathUsesConfigDirectory(t *testing.T) {
	t.Run("loaded config path", func(t *testing.T) {
		a := NewApp()
		configDir := t.TempDir()
		a.ConfigFile = filepath.Join(configDir, "config.yaml")

		got := a.defaultLogFilePath()
		want := filepath.Join(configDir, "badgermaps.log")
		if got != want {
			t.Fatalf("defaultLogFilePath mismatch: got %q want %q", got, want)
		}
	})

	t.Run("config flag path", func(t *testing.T) {
		a := NewApp()
		configPath := filepath.Join(t.TempDir(), "local-config.yaml")
		*a.State.ConfigFile = configPath

		got := a.defaultLogFilePath()
		absConfigPath, err := filepath.Abs(configPath)
		if err != nil {
			t.Fatalf("failed to resolve abs path: %v", err)
		}
		want := filepath.Join(filepath.Dir(absConfigPath), "badgermaps.log")
		if got != want {
			t.Fatalf("defaultLogFilePath mismatch: got %q want %q", got, want)
		}
	})
}

func TestInitLoggingFallsBackToConfigDirectory(t *testing.T) {
	a := NewApp()
	configDir := t.TempDir()
	a.ConfigFile = filepath.Join(configDir, "config.yaml")
	a.Config.LogFile = ""
	a.State.LogFile = ""

	if err := a.InitLogging(); err != nil {
		t.Fatalf("InitLogging failed: %v", err)
	}
	t.Cleanup(a.Close)

	want := filepath.Join(configDir, "badgermaps.log")
	if a.State.LogFile != want {
		t.Fatalf("State.LogFile mismatch: got %q want %q", a.State.LogFile, want)
	}
	if a.Config.LogFile != want {
		t.Fatalf("Config.LogFile mismatch: got %q want %q", a.Config.LogFile, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected log file to exist at %q: %v", want, err)
	}
}
