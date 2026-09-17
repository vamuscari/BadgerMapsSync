//go:build integration

package database

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"badgermaps/app/state"
)

func TestServerDatabaseSchemaIntegration(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		dbType string
	}{
		{name: "postgres", prefix: "BADGERMAPS_TEST_POSTGRES", dbType: "postgres"},
		{name: "mssql", prefix: "BADGERMAPS_TEST_MSSQL", dbType: "mssql"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config, ok := serverIntegrationConfig(tc.prefix, tc.dbType)
			if !ok {
				t.Skipf("%s_HOST is not configured", tc.prefix)
			}

			db, err := NewDB(&config)
			if err != nil {
				t.Fatalf("failed to create %s database: %v", tc.name, err)
			}
			if err := db.Connect(); err != nil {
				t.Fatalf("failed to open %s database: %v", tc.name, err)
			}
			t.Cleanup(func() {
				if err := db.Close(); err != nil {
					t.Logf("failed to close %s database: %v", tc.name, err)
				}
			})
			waitForServerDatabase(t, db)

			if err := db.DropAllTables(); err != nil {
				t.Fatalf("failed to clean %s schema: %v", tc.name, err)
			}
			t.Cleanup(func() {
				if err := db.DropAllTables(); err != nil {
					t.Logf("failed to clean %s schema after test: %v", tc.name, err)
				}
			})

			quietState := &state.State{Quiet: true}
			if err := db.EnforceSchema(quietState); err != nil {
				t.Fatalf("failed to enforce %s schema: %v", tc.name, err)
			}
			if err := db.ValidateSchema(quietState); err != nil {
				t.Fatalf("failed to validate %s schema: %v", tc.name, err)
			}

			if err := LogCommand(db, "integration", []string{"quoted", "argument"}, true, ""); err != nil {
				t.Fatalf("failed to log command on %s: %v", tc.name, err)
			}
			if err := LogWebhook(db, time.Now().UTC(), "POST", "/integration", `{}`, `{"ok":true}`); err != nil {
				t.Fatalf("failed to log webhook on %s: %v", tc.name, err)
			}

			if err := RunCommand(db, "MergeUserProfiles",
				42, "owner@example.com", "Test", "Owner", true, false, nil, nil,
				"email,phone_number", nil, nil, nil, nil, nil, true, true, true, true,
				30, true, 0, 7, "Example Company", "EX",
			); err != nil {
				t.Fatalf("failed to store profile on %s: %v", tc.name, err)
			}
			profile, err := GetProfile(db)
			if err != nil {
				t.Fatalf("failed to read profile on %s: %v", tc.name, err)
			}
			if profile.ProfileId.Int64 != 42 || profile.Company.Name.String != "Example Company" {
				t.Fatalf("unexpected profile returned by %s: %#v", tc.name, profile)
			}

			if err := UpdateConfiguration(db, "ApiProfileId", "42"); err != nil {
				t.Fatalf("failed to select profile on %s: %v", tc.name, err)
			}
			if err := RunCommand(db, "InsertDataSets",
				"email", 42, true, "Work Email", 1, "text", true, false,
				nil, nil, nil, nil, "Email",
			); err != nil {
				t.Fatalf("failed to store data set on %s: %v", tc.name, err)
			}
			if err := RefreshGeneratedViews(db, db.GetDB()); err != nil {
				t.Fatalf("failed to refresh generated views on %s: %v", tc.name, err)
			}
			if tc.dbType == "postgres" {
				if _, err := db.GetDB().Exec("GRANT SELECT ON accountsindexed TO PUBLIC"); err != nil {
					t.Fatalf("failed to grant AccountsIndexed access on %s: %v", tc.name, err)
				}
			}
			if err := RunCommand(db, "InsertDataSets",
				"priority", 42, true, "Priority", 2, "text", true, false,
				nil, nil, nil, nil, "CustomText",
			); err != nil {
				t.Fatalf("failed to add custom data set on %s: %v", tc.name, err)
			}
			if err := RefreshGeneratedViews(db, db.GetDB()); err != nil {
				t.Fatalf("failed to expand generated views on %s: %v", tc.name, err)
			}
			indexedColumns, err := db.GetTableColumns("AccountsIndexed")
			if err != nil {
				t.Fatalf("failed to inspect expanded AccountsIndexed on %s: %v", tc.name, err)
			}
			if !containsFold(indexedColumns, "Priority") {
				t.Fatalf("expanded AccountsIndexed on %s is missing Priority: %v", tc.name, indexedColumns)
			}
			if err := RunCommand(db, "DeleteDataSets", 42); err != nil {
				t.Fatalf("failed to remove data sets on %s: %v", tc.name, err)
			}
			if err := RefreshGeneratedViews(db, db.GetDB()); err != nil {
				t.Fatalf("failed to contract generated views on %s: %v", tc.name, err)
			}
			indexedColumns, err = db.GetTableColumns("AccountsIndexed")
			if err != nil {
				t.Fatalf("failed to inspect contracted AccountsIndexed on %s: %v", tc.name, err)
			}
			if containsFold(indexedColumns, "Priority") {
				t.Fatalf("contracted AccountsIndexed on %s retained Priority: %v", tc.name, indexedColumns)
			}
			if tc.dbType == "postgres" {
				var publicGrantCount int
				if err := db.GetDB().QueryRow(`
					SELECT COUNT(*)
					FROM information_schema.table_privileges
					WHERE table_schema = current_schema()
					  AND table_name = 'accountsindexed'
					  AND grantee = 'PUBLIC'
					  AND privilege_type = 'SELECT'
				`).Scan(&publicGrantCount); err != nil {
					t.Fatalf("failed to inspect AccountsIndexed grants on %s: %v", tc.name, err)
				}
				if publicGrantCount != 1 {
					t.Fatalf("AccountsIndexed SELECT grant was not preserved on %s", tc.name)
				}
			}
			for _, viewName := range requiredViews() {
				exists, err := db.ViewExists(viewName)
				if err != nil {
					t.Fatalf("failed to inspect %s on %s: %v", viewName, tc.name, err)
				}
				if !exists {
					t.Fatalf("expected %s on %s", viewName, tc.name)
				}
			}

			if err := UpgradeExistingSchema(db); err != nil {
				t.Fatalf("first %s schema upgrade failed: %v", tc.name, err)
			}
			if err := UpgradeExistingSchema(db); err != nil {
				t.Fatalf("second %s schema upgrade failed: %v", tc.name, err)
			}
			var migrationCount int
			query := "SELECT COUNT(*) FROM SchemaMigrations WHERE Version = " + integrationPlaceholder(tc.dbType)
			if err := db.GetDB().QueryRow(query, 1).Scan(&migrationCount); err != nil {
				t.Fatalf("failed to inspect %s migration metadata: %v", tc.name, err)
			}
			if migrationCount != 1 {
				t.Fatalf("expected one recorded migration on %s, found %d", tc.name, migrationCount)
			}
		})
	}
}

func serverIntegrationConfig(prefix, dbType string) (DBConfig, bool) {
	host := os.Getenv(prefix + "_HOST")
	if host == "" {
		return DBConfig{}, false
	}
	defaultPort := 5432
	if dbType == "mssql" {
		defaultPort = 1433
	}
	port := defaultPort
	if configuredPort := os.Getenv(prefix + "_PORT"); configuredPort != "" {
		if parsed, err := strconv.Atoi(configuredPort); err == nil {
			port = parsed
		}
	}
	return DBConfig{
		Type:     dbType,
		Host:     host,
		Port:     port,
		Database: os.Getenv(prefix + "_DATABASE"),
		Username: os.Getenv(prefix + "_USERNAME"),
		Password: os.Getenv(prefix + "_PASSWORD"),
		SSLMode:  os.Getenv(prefix + "_SSL_MODE"),
	}, true
}

func waitForServerDatabase(t *testing.T, db DB) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := db.TestConnection(); err == nil {
			return
		} else {
			lastErr = err
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("database did not become ready: %v", lastErr)
}

func integrationPlaceholder(dbType string) string {
	if dbType == "postgres" {
		return "$1"
	}
	return "?"
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func TestServerIntegrationConfiguration(t *testing.T) {
	const prefix = "BADGERMAPS_TEST_CONFIG"
	t.Setenv(prefix+"_HOST", "db.example")
	t.Setenv(prefix+"_PORT", "15432")
	t.Setenv(prefix+"_DATABASE", "badgermaps")
	t.Setenv(prefix+"_USERNAME", "tester")
	t.Setenv(prefix+"_PASSWORD", "secret")
	t.Setenv(prefix+"_SSL_MODE", "disable")

	config, ok := serverIntegrationConfig(prefix, "postgres")
	if !ok {
		t.Fatal("expected configured integration database")
	}
	if got := fmt.Sprintf("%s:%d/%s", config.Host, config.Port, config.Database); got != "db.example:15432/badgermaps" {
		t.Fatalf("unexpected integration config: %s", got)
	}
}
