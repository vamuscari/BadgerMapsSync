package gui

import (
	"badgermaps/app"
	"badgermaps/app/state"
	"badgermaps/database"
	"path/filepath"
	"testing"
)

func newPresenterWithCommandLogDB(t *testing.T) (*GuiPresenter, *app.App, *testGuiView) {
	t.Helper()

	a := app.NewApp()
	a.SetConfigFilePath(filepath.Join(t.TempDir(), "config.yaml"))

	dbPath := filepath.Join(t.TempDir(), "presenter_commands.db")
	db, err := database.NewDB(&database.DBConfig{Type: "sqlite3", Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create db config: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect db: %v", err)
	}
	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}
	db.SetConnected(true)
	a.DB = db

	view := &testGuiView{}
	presenter := NewGuiPresenter(a, view)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return presenter, a, view
}

func TestHandleSaveServerConfigWritesOperationalCommandLog(t *testing.T) {
	presenter, a, _ := newPresenterWithCommandLogDB(t)

	presenter.HandleSaveServerConfig("localhost", "8080", "UTC", false, "", "", "webhook-secret", "internal-token", true)

	var (
		command string
		success bool
	)
	if err := a.DB.GetDB().QueryRow(
		"SELECT Command, Success FROM CommandLog ORDER BY LogId DESC LIMIT 1",
	).Scan(&command, &success); err != nil {
		t.Fatalf("expected command log row, query failed: %v", err)
	}
	if command != "gui.server.save_config" {
		t.Fatalf("expected command %q, got %q", "gui.server.save_config", command)
	}
	if !success {
		t.Fatalf("expected successful command log entry")
	}
}

func TestHandleViewConfigDoesNotWriteOperationalCommandLog(t *testing.T) {
	presenter, a, _ := newPresenterWithCommandLogDB(t)

	presenter.HandleViewConfig()

	var count int
	if err := a.DB.GetDB().QueryRow("SELECT COUNT(*) FROM CommandLog").Scan(&count); err != nil {
		t.Fatalf("failed to count command log rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no command log entries for passive view action, got %d", count)
	}
}
