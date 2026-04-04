package app

import (
	"os"
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

func TestGetConfigFilePathPrefersExecutableDirectoryConfig(t *testing.T) {
	a := NewApp()

	exeDir := t.TempDir()
	exePath := filepath.Join(exeDir, "badgermaps")
	exeConfigPath := filepath.Join(exeDir, "config.yaml")
	if err := os.WriteFile(exeConfigPath, []byte("db:\n  type: sqlite3\n"), 0644); err != nil {
		t.Fatalf("failed to write executable config: %v", err)
	}

	workingDir := t.TempDir()
	workingConfigPath := filepath.Join(workingDir, "config.yaml")
	if err := os.WriteFile(workingConfigPath, []byte("db:\n  type: sqlite3\n"), 0644); err != nil {
		t.Fatalf("failed to write working directory config: %v", err)
	}

	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to read current working directory: %v", err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalCwd)
	}()

	originalResolver := resolveExecutablePath
	resolveExecutablePath = func() (string, error) {
		return exePath, nil
	}
	defer func() {
		resolveExecutablePath = originalResolver
	}()

	gotPath, ok, err := a.GetConfigFilePath()
	if err != nil {
		t.Fatalf("expected no error resolving config file path, got %v", err)
	}
	if !ok {
		t.Fatal("expected config file to be detected")
	}

	wantPath, err := filepath.Abs(exeConfigPath)
	if err != nil {
		t.Fatalf("failed to make expected path absolute: %v", err)
	}
	if gotPath != wantPath {
		t.Fatalf("expected executable directory config %q, got %q", wantPath, gotPath)
	}
}
