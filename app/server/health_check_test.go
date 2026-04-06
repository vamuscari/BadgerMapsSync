package server

import (
	"badgermaps/app/state"
	"badgermaps/database"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func setupHealthCheckTestDB(t *testing.T) database.DB {
	t.Helper()

	config := &database.DBConfig{
		Type: "sqlite3",
		Path: filepath.Join(t.TempDir(), "health_check.db"),
	}

	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect test database: %v", err)
	}

	if err := db.EnforceSchema(state.NewState()); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func TestBuildVersionFromBuildInfoUsesMainVersion(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
	}

	if got, want := buildVersionFromBuildInfo(info, true), "v1.2.3"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildVersionFromBuildInfoUsesRevision(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "0123456789abcdef"},
			{Key: "vcs.modified", Value: "false"},
		},
	}

	if got, want := buildVersionFromBuildInfo(info, true), "0123456789ab"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildVersionFromBuildInfoUsesDirtySuffix(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "fedcba9876543210"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	if got, want := buildVersionFromBuildInfo(info, true), "fedcba987654-dirty"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildVersionFromBuildInfoFallsBackToDevelopment(t *testing.T) {
	if got, want := buildVersionFromBuildInfo(nil, false), "development"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestValidateAccountDataRequiresExistingAccount(t *testing.T) {
	db := setupHealthCheckTestDB(t)
	validator := NewDataValidator(db, nil, nil)

	if err := validator.ValidateAccountData(999); err == nil {
		t.Fatal("expected missing account validation to fail")
	}
}

func TestValidateAccountDataPassesForExistingAccount(t *testing.T) {
	db := setupHealthCheckTestDB(t)
	validator := NewDataValidator(db, nil, nil)

	if _, err := db.GetDB().Exec("INSERT INTO Accounts (AccountId, FullName) VALUES (?, ?)", 42, "Test Account"); err != nil {
		t.Fatalf("failed to seed account: %v", err)
	}

	if err := validator.ValidateAccountData(42); err != nil {
		t.Fatalf("expected account validation to pass, got %v", err)
	}
}

func TestValidateDataConsistencyDetectsOrphanedRouteWaypoints(t *testing.T) {
	db := setupHealthCheckTestDB(t)
	validator := NewDataValidator(db, nil, nil)

	if _, err := db.GetDB().Exec("INSERT INTO RouteWaypoints (WaypointId, RouteId, Name) VALUES (?, ?, ?)", 1001, 9999, "Orphan Waypoint"); err != nil {
		t.Fatalf("failed to seed orphaned route waypoint: %v", err)
	}

	err := validator.ValidateDataConsistency()
	if err == nil {
		t.Fatal("expected consistency validation to fail for orphaned route waypoint")
	}
	if !strings.Contains(err.Error(), "orphaned_route_waypoints") {
		t.Fatalf("expected orphaned route waypoint violation, got %v", err)
	}
}

func TestValidateAccountsSchemaFailsWhenRequiredTableMissing(t *testing.T) {
	db := setupHealthCheckTestDB(t)
	validator := NewDataValidator(db, nil, nil)

	if _, err := db.GetDB().Exec("DROP TABLE AccountsPendingChanges"); err != nil {
		t.Fatalf("failed to drop required table: %v", err)
	}

	if err := validator.validateAccountsSchema(); err == nil {
		t.Fatal("expected account schema validation to fail when required table is missing")
	}
}

func TestHealthCheckerValidateDatabaseSchemaPassesOnValidSchema(t *testing.T) {
	db := setupHealthCheckTestDB(t)
	checker := NewHealthChecker(db, nil, nil, nil)

	if err := checker.validateDatabaseSchema(); err != nil {
		t.Fatalf("expected schema validation to pass, got %v", err)
	}
}
