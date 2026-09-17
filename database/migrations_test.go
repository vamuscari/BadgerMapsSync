package database

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestRunSchemaMigrationsRollsBackFailedMigration(t *testing.T) {
	db, err := NewDB(&DBConfig{Type: "sqlite3", Path: filepath.Join(t.TempDir(), "rollback.db")})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()
	if _, err := db.GetDB().Exec("CREATE TABLE MigrationProbe (Value TEXT NOT NULL)"); err != nil {
		t.Fatalf("failed to create migration probe: %v", err)
	}

	wantErr := errors.New("forced migration failure")
	err = runSchemaMigrations(db, []schemaMigration{{
		Version: 101,
		Name:    "rollback probe",
		Apply: func(_ DB, tx *sql.Tx) error {
			if _, err := tx.Exec("INSERT INTO MigrationProbe (Value) VALUES (?)", "must roll back"); err != nil {
				return err
			}
			return wantErr
		},
	}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected migration error %v, got %v", wantErr, err)
	}

	var probeCount int
	if err := db.GetDB().QueryRow("SELECT COUNT(*) FROM MigrationProbe").Scan(&probeCount); err != nil {
		t.Fatalf("failed to inspect migration probe: %v", err)
	}
	if probeCount != 0 {
		t.Fatalf("expected migration writes to roll back, found %d row(s)", probeCount)
	}

	var versionCount int
	if err := db.GetDB().QueryRow("SELECT COUNT(*) FROM SchemaMigrations WHERE Version = ?", 101).Scan(&versionCount); err != nil {
		t.Fatalf("failed to inspect migration version: %v", err)
	}
	if versionCount != 0 {
		t.Fatal("failed migration was recorded as applied")
	}
}

func TestRunSchemaMigrationsAppliesEachVersionOnce(t *testing.T) {
	db, err := NewDB(&DBConfig{Type: "sqlite3", Path: filepath.Join(t.TempDir(), "once.db")})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()
	if _, err := db.GetDB().Exec("CREATE TABLE MigrationProbe (Value TEXT NOT NULL)"); err != nil {
		t.Fatalf("failed to create migration probe: %v", err)
	}

	applyCount := 0
	migrations := []schemaMigration{{
		Version: 102,
		Name:    "run once probe",
		Apply: func(_ DB, tx *sql.Tx) error {
			applyCount++
			_, err := tx.Exec("INSERT INTO MigrationProbe (Value) VALUES (?)", "applied")
			return err
		},
	}}
	if err := runSchemaMigrations(db, migrations); err != nil {
		t.Fatalf("first migration run failed: %v", err)
	}
	if err := runSchemaMigrations(db, migrations); err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}
	if applyCount != 1 {
		t.Fatalf("expected migration to run once, ran %d times", applyCount)
	}

	var probeCount int
	if err := db.GetDB().QueryRow("SELECT COUNT(*) FROM MigrationProbe").Scan(&probeCount); err != nil {
		t.Fatalf("failed to inspect migration probe: %v", err)
	}
	if probeCount != 1 {
		t.Fatalf("expected one committed migration write, found %d", probeCount)
	}
}
