package push

import (
	"badgermaps/app"
	"badgermaps/app/state"
	"badgermaps/database"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHandleListFormatsCreatedAtInDisplayTimezone(t *testing.T) {
	a := app.NewApp()
	a.State.NoColor = true
	a.Config.Server.Timezone = "America/Los_Angeles"

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.NewDB(&database.DBConfig{
		Type: "sqlite3",
		Path: dbPath,
	})
	if err != nil {
		t.Fatalf("failed to create test database config: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect test database: %v", err)
	}
	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}
	defer db.Close()

	createdAt := time.Date(2026, time.April, 2, 0, 30, 0, 0, time.UTC)
	_, err = db.GetDB().Exec(
		"INSERT INTO AccountsPendingChanges (AccountId, ChangeType, Changes, Status, CreatedAt) VALUES (?, ?, ?, ?, ?)",
		123,
		"UPDATE",
		`{"name":"example"}`,
		"pending",
		createdAt.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("failed to seed pending change: %v", err)
	}

	a.DB = db
	presenter := NewCliPresenter(a)

	changes, err := database.GetPendingAccountChanges(db)
	if err != nil {
		t.Fatalf("failed to read seeded pending changes: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 pending change, got %d", len(changes))
	}
	expectedTS := a.FormatTimestampInDisplayTimezone(changes[0].CreatedAt, time.RFC3339)

	output := captureStdout(t, func() {
		if err := presenter.HandleList("accounts", "pending", "", 0, "date_desc"); err != nil {
			t.Fatalf("HandleList returned error: %v", err)
		}
	})

	if !strings.Contains(output, expectedTS) {
		t.Fatalf("expected output to include %q, got: %s", expectedTS, output)
	}
	if !strings.Contains(output, "["+a.DisplayTimezoneName()+"]") {
		t.Fatalf("expected output to include timezone label %q, got: %s", a.DisplayTimezoneName(), output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	os.Stdout = w
	defer func() {
		os.Stdout = originalStdout
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}

	return string(data)
}
