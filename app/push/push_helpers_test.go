package push

import (
	"badgermaps/app"
	"badgermaps/database"
	"testing"
	"time"
)

func TestFilterAndSortAccountChangesDateUsesDisplayTimezone(t *testing.T) {
	a := app.NewApp()
	a.Config.Server.Timezone = "America/Los_Angeles"

	changes := []database.AccountPendingChange{
		{
			ChangeId:   1,
			AccountId:  100,
			ChangeType: "UPDATE",
			Status:     "pending",
			CreatedAt:  time.Date(2026, time.April, 2, 0, 30, 0, 0, time.UTC), // 2026-04-01 in PDT
		},
		{
			ChangeId:   2,
			AccountId:  101,
			ChangeType: "UPDATE",
			Status:     "pending",
			CreatedAt:  time.Date(2026, time.April, 2, 8, 0, 0, 0, time.UTC), // 2026-04-02 in PDT
		},
	}

	filtered := filterAndSortAccountChanges(a, changes, PushFilterOptions{
		Status:  "pending",
		Date:    "2026-04-01",
		OrderBy: "date",
	})

	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered account change, got %d", len(filtered))
	}
	if got, want := filtered[0].ChangeId, 1; got != want {
		t.Fatalf("expected account change id %d, got %d", want, got)
	}
}

func TestFilterAndSortCheckinChangesDateUsesDisplayTimezone(t *testing.T) {
	a := app.NewApp()
	a.Config.Server.Timezone = "America/Los_Angeles"

	changes := []database.CheckinPendingChange{
		{
			ChangeId:   1,
			CheckinId:  200,
			AccountId:  100,
			ChangeType: "CREATE",
			Status:     "pending",
			CreatedAt:  time.Date(2026, time.April, 2, 0, 30, 0, 0, time.UTC), // 2026-04-01 in PDT
		},
		{
			ChangeId:   2,
			CheckinId:  201,
			AccountId:  101,
			ChangeType: "CREATE",
			Status:     "pending",
			CreatedAt:  time.Date(2026, time.April, 2, 8, 0, 0, 0, time.UTC), // 2026-04-02 in PDT
		},
	}

	filtered := filterAndSortCheckinChanges(a, changes, PushFilterOptions{
		Status:  "pending",
		Date:    "2026-04-01",
		OrderBy: "date",
	})

	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered checkin change, got %d", len(filtered))
	}
	if got, want := filtered[0].ChangeId, 1; got != want {
		t.Fatalf("expected checkin change id %d, got %d", want, got)
	}
}
