package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLFiles(t *testing.T) {
	expectedFiles := []string{
		"check_column_exists.sql",
		"check_index_exists.sql",
		"check_table_exists.sql",
		"create_account_checkins_pending_changes_table.sql",
		"create_account_checkins_table.sql",
		"create_account_locations_table.sql",
		"create_accounts_pending_changes_table.sql",
		"create_accounts_table.sql",
		"create_data_set_values_table.sql",
		"create_data_sets_table.sql",
		"create_indexes.sql",
		"create_route_waypoints_table.sql",
		"create_routes_table.sql",
		"create_user_profiles_table.sql",
		"delete_account_locations.sql",
		"delete_data_set_values.sql",
		"delete_data_sets.sql",
		"delete_route_waypoints.sql",
		"get_account_by_id.sql",
		"get_all_account_ids.sql",
		"get_checkin_by_id.sql",
		"get_pending_account_changes.sql",
		"get_pending_checkin_changes.sql",
		"get_profile.sql",
		"get_route_by_id.sql",
		"get_table_columns.sql",
		"insert_account_locations.sql",
		"insert_data_set_values.sql",
		"insert_data_sets.sql",
		"insert_route_waypoints.sql",
		"merge_account_checkins.sql",
		"merge_accounts_basic.sql",
		"merge_accounts_detailed.sql",
		"merge_routes.sql",
		"merge_user_profiles.sql",
		"search_accounts.sql",
		"search_routes.sql",
		"update_pending_change_status.sql",
	}

	checkFiles := func(t *testing.T, dir string, expected []string) {
		actualFiles := make(map[string]bool)
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read directory %s: %v", dir, err)
		}

		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
				actualFiles[file.Name()] = true
			}
		}

		for _, f := range expected {
			if _, ok := actualFiles[f]; !ok {
				t.Errorf("missing file in %s: %s", dir, f)
			}
		}

		if len(expected) != len(actualFiles) {
			for f := range actualFiles {
				found := false
				for _, e := range expected {
					if f == e {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("unexpected file in %s: %s", dir, f)
				}
			}
		}
	}

	t.Run("sqlite3", func(t *testing.T) {
		checkFiles(t, filepath.Join("sqlite3"), expectedFiles)
	})

	t.Run("postgres", func(t *testing.T) {
		checkFiles(t, filepath.Join("postgres"), expectedFiles)
	})

	t.Run("mssql", func(t *testing.T) {
		checkFiles(t, filepath.Join("mssql"), expectedFiles)
	})
}
