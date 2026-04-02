package app

import (
	"path/filepath"
	"testing"
)

func TestSetConfigFilePathUpdatesStateAndPIDPath(t *testing.T) {
	a := NewApp()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	a.SetConfigFilePath(configPath)

	if a.ConfigFile == "" {
		t.Fatalf("expected ConfigFile to be set")
	}
	if !filepath.IsAbs(a.ConfigFile) {
		t.Fatalf("expected ConfigFile to be absolute, got %q", a.ConfigFile)
	}
	if a.State.ConfigFile == nil || *a.State.ConfigFile != a.ConfigFile {
		t.Fatalf("expected state ConfigFile pointer to match app ConfigFile")
	}

	wantPIDPath := filepath.Join(filepath.Dir(a.ConfigFile), ".badgermaps.pid")
	if a.State.PIDFile != wantPIDPath {
		t.Fatalf("expected PID path %q, got %q", wantPIDPath, a.State.PIDFile)
	}
}
