//go:build windows

package server

import (
	"badgermaps/app"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWindowsServiceConfigPathUsesExplicitPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "service-config.yaml")
	if err := os.WriteFile(configPath, []byte("server:\n  host: localhost\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	a := app.NewApp()
	got, err := resolveWindowsServiceConfigPath(a, configPath, filepath.Join(dir, "BadgerMapsSync.exe"))
	if err != nil {
		t.Fatalf("expected config path to resolve: %v", err)
	}
	if got != configPath {
		t.Fatalf("expected %q, got %q", configPath, got)
	}
	if a.ConfigFile != configPath {
		t.Fatalf("expected app config path %q, got %q", configPath, a.ConfigFile)
	}
}

func TestResolveWindowsServiceConfigPathDefaultsBesideExecutable(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("server:\n  host: localhost\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	got, err := resolveWindowsServiceConfigPath(nil, "", filepath.Join(dir, "BadgerMapsSync.exe"))
	if err != nil {
		t.Fatalf("expected default config path to resolve: %v", err)
	}
	if got != configPath {
		t.Fatalf("expected %q, got %q", configPath, got)
	}
}

func TestResolveWindowsServiceConfigPathRejectsMissingConfig(t *testing.T) {
	_, err := resolveWindowsServiceConfigPath(nil, filepath.Join(t.TempDir(), "missing.yaml"), `C:\BadgerMapsSync.exe`)
	if err == nil {
		t.Fatalf("expected missing config error")
	}
}
