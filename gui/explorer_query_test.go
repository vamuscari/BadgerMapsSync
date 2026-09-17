package gui

import (
	"strings"
	"testing"
)

func TestBuildExplorerWhereClause(t *testing.T) {
	tests := []struct {
		name        string
		column      string
		mode        ExplorerFilterMode
		value       string
		expectedSQL string
		expectedArg string
		dbType      string
	}{
		{
			name:        "contains",
			column:      "Name",
			mode:        FilterModeContains,
			value:       "Acme",
			expectedSQL: `"Name" LIKE ?`,
			expectedArg: "%Acme%",
			dbType:      "sqlite3",
		},
		{
			name:        "equals",
			column:      "AccountID",
			mode:        FilterModeEquals,
			value:       "1234",
			expectedSQL: `"AccountID" = ?`,
			expectedArg: "1234",
			dbType:      "sqlite3",
		},
		{
			name:        "starts with",
			column:      "Email",
			mode:        FilterModeStartsWith,
			value:       "info@",
			expectedSQL: `"Email" LIKE ?`,
			expectedArg: "info@%",
			dbType:      "sqlite3",
		},
		{
			name:        "ends with",
			column:      "Email",
			mode:        FilterModeEndsWith,
			value:       "@badgermaps.com",
			expectedSQL: `"Email" LIKE ?`,
			expectedArg: "%@badgermaps.com",
			dbType:      "sqlite3",
		},
		{
			name:        "not equals binds quotes",
			column:      "Status",
			mode:        FilterModeNotEquals,
			value:       "O'Reilly",
			expectedSQL: `"Status" <> ?`,
			expectedArg: "O'Reilly",
			dbType:      "sqlite3",
		},
		{
			name:        "postgres uses ilike",
			column:      "Name",
			mode:        FilterModeContains,
			value:       "Acme",
			expectedSQL: `"Name" ILIKE $1`,
			expectedArg: "%Acme%",
			dbType:      "postgres",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			filters := []ExplorerFilterClause{{Column: tc.column, Mode: tc.mode, Value: tc.value}}
			got, args := buildExplorerWhereClause(filters, tc.dbType)
			if got != tc.expectedSQL {
				t.Fatalf("expected %q, got %q", tc.expectedSQL, got)
			}
			if len(args) != 1 || args[0] != tc.expectedArg {
				t.Fatalf("expected argument %q, got %#v", tc.expectedArg, args)
			}
		})
	}

	if got, _ := buildExplorerWhereClause([]ExplorerFilterClause{{Column: "", Mode: FilterModeContains, Value: "value"}}, "sqlite3"); got != "" {
		t.Fatalf("expected empty where clause when column missing, got %q", got)
	}

	if got, _ := buildExplorerWhereClause([]ExplorerFilterClause{{Column: "Name", Mode: FilterModeNone, Value: "value"}}, "sqlite3"); got != "" {
		t.Fatalf("expected empty where clause when mode none, got %q", got)
	}

	multi, args := buildExplorerWhereClause([]ExplorerFilterClause{
		{Column: "Status", Mode: FilterModeEquals, Value: "pending"},
		{Column: "Name", Mode: FilterModeContains, Value: "Acme"},
	}, "postgres")
	expectedMulti := `"Status" = $1 AND "Name" ILIKE $2`
	if multi != expectedMulti {
		t.Fatalf("expected combined clause %q, got %q", expectedMulti, multi)
	}
	if len(args) != 2 || args[0] != "pending" || args[1] != "%Acme%" {
		t.Fatalf("unexpected combined arguments: %#v", args)
	}
}

func TestBuildExplorerOrderClause(t *testing.T) {
	columns := []string{"ID", "Name", "CreatedAt"}

	got := buildExplorerOrderClause(columns, "Name", false, "sqlite3")
	want := `ORDER BY "Name" ASC`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	got = buildExplorerOrderClause(columns, "", true, "postgres")
	want = `ORDER BY "CreatedAt" DESC`
	if got != want {
		t.Fatalf("expected fallback order %q, got %q", want, got)
	}

	got = buildExplorerOrderClause([]string{}, "", false, "mssql")
	want = "ORDER BY 1 ASC"
	if got != want {
		t.Fatalf("expected MSSQL fallback %q, got %q", want, got)
	}

	got = buildExplorerOrderClause(nil, "", false, "sqlite3")
	if got != "" {
		t.Fatalf("expected empty order clause when no columns, got %q", got)
	}
}

func TestBuildExplorerSelectQueryDialects(t *testing.T) {
	table := "Accounts"
	where := `"Name" LIKE ?`
	order := `ORDER BY "CreatedAt" DESC`
	page := 0
	pageSize := 25

	tests := []struct {
		name     string
		dbType   string
		expected string
	}{
		{
			name:     "sqlite uses limit",
			dbType:   "sqlite3",
			expected: `SELECT * FROM "Accounts" WHERE "Name" LIKE ? ORDER BY "CreatedAt" DESC LIMIT 25 OFFSET 0`,
		},
		{
			name:     "postgres uses limit",
			dbType:   "postgres",
			expected: `SELECT * FROM "Accounts" WHERE "Name" LIKE ? ORDER BY "CreatedAt" DESC LIMIT 25 OFFSET 0`,
		},
		{
			name:     "mssql uses offset fetch",
			dbType:   "mssql",
			expected: `SELECT * FROM [Accounts] WHERE "Name" LIKE ? ORDER BY "CreatedAt" DESC OFFSET 0 ROWS FETCH NEXT 25 ROWS ONLY`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := buildExplorerSelectQuery(table, where, order, page, pageSize, tc.dbType)
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}

	nonPositive := buildExplorerSelectQuery(table, "", "", -1, 0, "sqlite3")
	if !strings.HasSuffix(nonPositive, "LIMIT 50 OFFSET 0") {
		t.Fatalf("expected default pagination when page/pageSize invalid, got %q", nonPositive)
	}
}

func TestExplorerQueryCompositionAcrossDialects(t *testing.T) {
	columns := []string{"ID", "Name", "CreatedAt"}
	baseOpts := ExplorerQueryOptions{
		Filters: []ExplorerFilterClause{
			{Column: "name", Mode: FilterModeStartsWith, Value: "Ac"},
		},
		OrderColumn:     "createdat",
		OrderDescending: true,
	}

	tests := []struct {
		name           string
		dbType         string
		expectedOrder  string
		expectedSelect string
	}{
		{
			name:           "sqlite normalized",
			dbType:         "sqlite3",
			expectedOrder:  `ORDER BY "CreatedAt" DESC`,
			expectedSelect: `SELECT * FROM "Accounts" WHERE "Name" LIKE ? ORDER BY "CreatedAt" DESC LIMIT 25 OFFSET 25`,
		},
		{
			name:           "postgres normalized",
			dbType:         "postgres",
			expectedOrder:  `ORDER BY "CreatedAt" DESC`,
			expectedSelect: `SELECT * FROM "Accounts" WHERE "Name" ILIKE $1 ORDER BY "CreatedAt" DESC LIMIT 25 OFFSET 25`,
		},
		{
			name:           "mssql normalized",
			dbType:         "mssql",
			expectedOrder:  "ORDER BY [CreatedAt] DESC",
			expectedSelect: "SELECT * FROM [Accounts] WHERE [Name] LIKE ? ORDER BY [CreatedAt] DESC OFFSET 25 ROWS FETCH NEXT 25 ROWS ONLY",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			normalized := normalizeExplorerOptions(baseOpts)
			resolved := resolveExplorerFilters(normalized.Filters, columns)
			whereClause, args := buildExplorerWhereClause(resolved, tc.dbType)
			if whereClause == "" {
				t.Fatalf("expected where clause for %s", tc.name)
			}
			if len(args) != 1 || args[0] != "Ac%" {
				t.Fatalf("unexpected filter arguments for %s: %#v", tc.name, args)
			}

			orderColumn := matchColumn(columns, normalized.OrderColumn)
			orderClause := buildExplorerOrderClause(columns, orderColumn, normalized.OrderDescending, tc.dbType)
			if orderClause != tc.expectedOrder {
				t.Fatalf("expected order clause %q, got %q", tc.expectedOrder, orderClause)
			}

			selectQuery := buildExplorerSelectQuery("Accounts", whereClause, orderClause, 1, 25, tc.dbType)
			if selectQuery != tc.expectedSelect {
				t.Fatalf("expected select query %q, got %q", tc.expectedSelect, selectQuery)
			}

			countQuery := buildExplorerCountQuery("Accounts", whereClause, tc.dbType)
			expectedWhere := `WHERE "Name" LIKE ?`
			if strings.EqualFold(tc.dbType, "postgres") {
				expectedWhere = `WHERE "Name" ILIKE $1`
			} else if strings.EqualFold(tc.dbType, "mssql") {
				expectedWhere = "WHERE [Name] LIKE ?"
			}
			if !strings.Contains(countQuery, expectedWhere) {
				t.Fatalf("expected count query to include where clause, got %q", countQuery)
			}
		})
	}
}

func TestExplorerQueriesBindValuesAndQuoteIdentifiers(t *testing.T) {
	maliciousValue := `x' OR 1=1; --`
	whereClause, args := buildExplorerWhereClause([]ExplorerFilterClause{{
		Column: `Display"Name`,
		Mode:   FilterModeEquals,
		Value:  maliciousValue,
	}}, "postgres")
	if strings.Contains(whereClause, maliciousValue) {
		t.Fatalf("filter value was interpolated into SQL: %q", whereClause)
	}
	if whereClause != `"Display""Name" = $1` {
		t.Fatalf("unexpected quoted PostgreSQL condition: %q", whereClause)
	}
	if len(args) != 1 || args[0] != maliciousValue {
		t.Fatalf("expected malicious value to remain a bound argument, got %#v", args)
	}

	tests := []struct {
		dbType     string
		identifier string
		want       string
	}{
		{dbType: "sqlite3", identifier: `odd"table`, want: `"odd""table"`},
		{dbType: "postgres", identifier: `odd"table`, want: `"odd""table"`},
		{dbType: "mssql", identifier: `odd]table`, want: `[odd]]table]`},
	}
	for _, tc := range tests {
		if got := quoteExplorerIdentifier(tc.identifier, tc.dbType); got != tc.want {
			t.Errorf("quoteExplorerIdentifier(%q, %q) = %q, want %q", tc.identifier, tc.dbType, got, tc.want)
		}
	}
}
