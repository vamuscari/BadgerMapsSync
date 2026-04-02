package gui

import (
	"badgermaps/app"
	"badgermaps/app/state"
	"badgermaps/database"
	"path/filepath"
	"testing"
)

func TestDashboardPendingChangesCountOnlyIncludesPending(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "dashboard_pending.db")

	a := app.NewApp()
	a.DB, _ = database.NewDB(&database.DBConfig{Type: "sqlite3", Path: dbPath})
	if err := a.DB.Connect(); err != nil {
		t.Fatalf("failed to connect test db: %v", err)
	}
	defer a.DB.Close()
	if err := a.DB.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}
	a.DB.SetConnected(true)

	sqlDB := a.DB.GetDB()
	if sqlDB == nil {
		t.Fatalf("expected non-nil sql DB")
	}

	// AccountsPendingChanges: 1 pending, 1 completed
	if _, err := sqlDB.Exec(`
		INSERT INTO AccountsPendingChanges (AccountId, ChangeType, Changes, Status)
		VALUES
			(1, 'UPDATE', '{}', 'pending'),
			(2, 'UPDATE', '{}', 'completed')
	`); err != nil {
		t.Fatalf("failed seeding account pending changes: %v", err)
	}

	// AccountCheckinsPendingChanges: 2 pending, 1 failed
	if _, err := sqlDB.Exec(`
		INSERT INTO AccountCheckinsPendingChanges (CheckinId, AccountId, ChangeType, Status)
		VALUES
			(10, 1, 'UPDATE', 'pending'),
			(11, 1, 'UPDATE', 'pending'),
			(12, 1, 'UPDATE', 'failed')
	`); err != nil {
		t.Fatalf("failed seeding check-in pending changes: %v", err)
	}

	ui := &Gui{app: a}
	d := NewSmartDashboard(ui, nil)
	count := d.getPendingChangesCount()

	if count.Accounts != 1 {
		t.Fatalf("expected 1 pending account change, got %d", count.Accounts)
	}
	if count.Checkins != 2 {
		t.Fatalf("expected 2 pending check-in changes, got %d", count.Checkins)
	}
	if count.Total != 3 {
		t.Fatalf("expected total pending changes 3, got %d", count.Total)
	}
}
