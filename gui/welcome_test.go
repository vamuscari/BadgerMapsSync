package gui

import (
	"badgermaps/app"
	"os"
	"path/filepath"
	"testing"
)

func TestWelcomeValidateStepUsesWizardState(t *testing.T) {
	a := app.NewApp()
	w := NewWelcomeScreen(a, nil, func() {})

	// API step should validate from wizard fields even when a.API is nil.
	w.apiKey = "1234567890"
	w.apiBaseURL = "https://example.com/api"
	if !w.validateStep(1) {
		t.Fatalf("expected API step validation to pass using wizard state")
	}

	// Database step should validate from wizard fields even when a.DB is nil.
	w.dbType = "sqlite3"
	w.dbPath = "badgermaps.db"
	if !w.validateStep(2) {
		t.Fatalf("expected database step validation to pass using wizard state")
	}

	// Server step should validate against wizard host/port fields.
	w.serverHost = "127.0.0.1"
	w.serverPort = "8080"
	if !w.validateStep(3) {
		t.Fatalf("expected server step validation to pass using wizard state")
	}
}

func TestWelcomeApplyWizardConfigurationPersistsConfig(t *testing.T) {
	a := app.NewApp()
	tempDir := t.TempDir()
	a.ConfigFile = filepath.Join(tempDir, "config.yaml")

	w := NewWelcomeScreen(a, nil, func() {})
	w.apiKey = "1234567890"
	w.apiBaseURL = "http://127.0.0.1:1"
	w.dbType = "sqlite3"
	w.dbPath = filepath.Join(tempDir, "wizard.db")
	w.serverHost = "127.0.0.1"
	w.serverPort = "8081"
	w.serverTLSEnabled = true

	if err := w.applyWizardConfiguration(); err != nil {
		t.Fatalf("applyWizardConfiguration failed: %v", err)
	}

	if _, err := os.Stat(a.ConfigFile); err != nil {
		t.Fatalf("expected saved config file at %s: %v", a.ConfigFile, err)
	}

	if a.Config.API.APIKey != w.apiKey {
		t.Fatalf("expected API key %q, got %q", w.apiKey, a.Config.API.APIKey)
	}
	if a.Config.DB.Type != "sqlite3" {
		t.Fatalf("expected DB type sqlite3, got %q", a.Config.DB.Type)
	}
	if a.Config.Server.Port != 8081 {
		t.Fatalf("expected server port 8081, got %d", a.Config.Server.Port)
	}
	if !a.Config.Server.TLSEnabled {
		t.Fatalf("expected TLS to be enabled from wizard state")
	}
}
