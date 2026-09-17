package main

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestReseedDemoCreatesCurrentSchema(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 CLI is not installed")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	repoRoot := filepath.Dir(workingDir)
	dbPath := filepath.Join(t.TempDir(), "demo.db")
	cmd := exec.Command("bash", "scripts/reseed_demo.sh", dbPath)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"BULK_ACCOUNTS=0",
		"BULK_ROUTES=0",
		"CHECKINS_PER=0",
		"SYNC_RUNS=0",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("reseed_demo.sh failed: %v\n%s", err, output)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open reseeded database: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"JobLog", "SchemaMigrations"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count); err != nil {
			t.Fatalf("failed to inspect %s: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("expected reseed script to create %s", table)
		}
	}

	var jobCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM JobLog").Scan(&jobCount); err != nil {
		t.Fatalf("failed to count demo JobLog rows: %v", err)
	}
	if jobCount < 2 {
		t.Fatalf("expected demo JobLog rows, found %d", jobCount)
	}
}
