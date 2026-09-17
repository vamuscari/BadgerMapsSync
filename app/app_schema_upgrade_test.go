package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"badgermaps/app/state"
	"badgermaps/database"
)

func TestLoadConfigUpgradesExistingSchemaObjects(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "existing.db")
	configPath := filepath.Join(tempDir, "config.yaml")

	db, err := database.NewDB(&database.DBConfig{Type: "sqlite3", Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	if err := db.EnforceSchema(&state.State{Quiet: true}); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}
	if _, err := db.GetDB().Exec("DROP VIEW AccountsIndexed; DROP VIEW AccountsWithLabels; CREATE VIEW AccountsWithLabels AS SELECT AccountId FROM Accounts;"); err != nil {
		t.Fatalf("failed to install stale view: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close seed database: %v", err)
	}

	configYAML := strings.Join([]string{
		"db:",
		"  type: sqlite3",
		"  path: " + dbPath,
		"server:",
		"  internal_api_token: fixed-token",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	a := NewApp()
	a.SetConfigFilePath(configPath)
	if err := a.LoadConfig(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	defer a.Close()

	var columnCount int
	if err := a.DB.GetDB().QueryRow("SELECT COUNT(*) FROM pragma_table_info('AccountsWithLabels')").Scan(&columnCount); err != nil {
		t.Fatalf("failed to inspect upgraded view: %v", err)
	}
	if columnCount <= 1 {
		t.Fatalf("expected startup to refresh the stale view, found %d column(s)", columnCount)
	}

	indexedExists, err := a.DB.ViewExists("AccountsIndexed")
	if err != nil {
		t.Fatalf("failed to inspect upgraded AccountsIndexed view: %v", err)
	}
	if !indexedExists {
		t.Fatal("expected startup to create the missing AccountsIndexed view")
	}
}

func TestLoadConfigReturnsSchemaUpgradeFailure(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "broken.db")
	configPath := filepath.Join(tempDir, "config.yaml")

	db, err := database.NewDB(&database.DBConfig{Type: "sqlite3", Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	if _, err := db.GetDB().Exec(`
		CREATE TABLE Configurations (InvalidColumn TEXT);
	`); err != nil {
		t.Fatalf("failed to seed broken schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close seed database: %v", err)
	}

	configYAML := strings.Join([]string{
		"db:",
		"  type: sqlite3",
		"  path: " + dbPath,
		"server:",
		"  internal_api_token: fixed-token",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configYAML), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	a := NewApp()
	a.SetConfigFilePath(configPath)
	err = a.LoadConfig()
	if err == nil {
		t.Fatal("expected schema upgrade failure to abort config loading")
	}
	if !strings.Contains(err.Error(), "schema upgrade") {
		t.Fatalf("expected schema upgrade context, got %v", err)
	}
}
