package push

import (
	"badgermaps/app"
	"badgermaps/database"
	"fmt"
	"sort"
	"strings"
	"time"
)

// PushFilterOptions defines the criteria for filtering pending pushes.
type PushFilterOptions struct {
	Status    string
	Type      string
	Date      string
	AccountID int
	OrderBy   string
}

// GetFilteredPendingChanges retrieves pending account or check-in changes based on the provided filters.
func GetFilteredPendingChanges(a *app.App, entityType string, options PushFilterOptions) (interface{}, error) {
	var accountChanges []database.AccountPendingChange
	var checkinChanges []database.CheckinPendingChange
	var err error

	switch strings.ToLower(entityType) {
	case "accounts":
		accountChanges, err = database.GetPendingAccountChanges(a.DB)
		if err != nil {
			return nil, fmt.Errorf("error getting pending account changes: %w", err)
		}
		return filterAndSortAccountChanges(a, accountChanges, options), nil
	case "checkins":
		checkinChanges, err = database.GetPendingCheckinChanges(a.DB)
		if err != nil {
			return nil, fmt.Errorf("error getting pending check-in changes: %w", err)
		}
		return filterAndSortCheckinChanges(a, checkinChanges, options), nil
	default:
		return nil, fmt.Errorf("unsupported entity type for filtering: %s", entityType)
	}
}

// filterAndSortAccountChanges applies the given filters and sorting to a slice of account changes.
func filterAndSortAccountChanges(a *app.App, changes []database.AccountPendingChange, options PushFilterOptions) []database.AccountPendingChange {
	var filtered []database.AccountPendingChange

	for _, change := range changes {
		if options.Status != "" && !strings.EqualFold(change.Status, options.Status) {
			continue
		}
		if options.Type != "" && !strings.EqualFold(change.ChangeType, options.Type) {
			continue
		}
		if options.Date != "" {
			changeDate := dateInDisplayTimezone(a, change.CreatedAt)
			if changeDate != options.Date {
				continue
			}
		}
		if options.AccountID != 0 && change.AccountId != options.AccountID {
			continue
		}
		filtered = append(filtered, change)
	}

	// Sorting logic
	if options.OrderBy != "" {
		parts := strings.Split(options.OrderBy, "_")
		field := parts[0]
		desc := len(parts) > 1 && parts[1] == "desc"

		sort.Slice(filtered, func(i, j int) bool {
			var less bool
			switch field {
			case "status":
				less = filtered[i].Status < filtered[j].Status
			case "type":
				less = filtered[i].ChangeType < filtered[j].ChangeType
			case "date":
				less = filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
			case "account":
				less = filtered[i].AccountId < filtered[j].AccountId
			}
			if desc {
				return !less
			}
			return less
		})
	}

	return filtered
}

// filterAndSortCheckinChanges applies the given filters and sorting to a slice of check-in changes.
func filterAndSortCheckinChanges(a *app.App, changes []database.CheckinPendingChange, options PushFilterOptions) []database.CheckinPendingChange {
	var filtered []database.CheckinPendingChange

	for _, change := range changes {
		if options.Status != "" && !strings.EqualFold(change.Status, options.Status) {
			continue
		}
		if options.Type != "" && !strings.EqualFold(change.ChangeType, options.Type) {
			continue
		}
		if options.Date != "" {
			changeDate := dateInDisplayTimezone(a, change.CreatedAt)
			if changeDate != options.Date {
				continue
			}
		}
		if options.AccountID != 0 && change.AccountId != options.AccountID {
			continue
		}
		filtered = append(filtered, change)
	}

	// Sorting logic
	if options.OrderBy != "" {
		parts := strings.Split(options.OrderBy, "_")
		field := parts[0]
		desc := len(parts) > 1 && parts[1] == "desc"

		sort.Slice(filtered, func(i, j int) bool {
			var less bool
			switch field {
			case "status":
				less = filtered[i].Status < filtered[j].Status
			case "type":
				less = filtered[i].ChangeType < filtered[j].ChangeType
			case "date":
				less = filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
			case "account":
				less = filtered[i].AccountId < filtered[j].AccountId
			}
			if desc {
				return !less
			}
			return less
		})
	}

	return filtered
}

func dateInDisplayTimezone(a *app.App, ts time.Time) string {
	if a == nil {
		return ts.Format("2006-01-02")
	}
	return a.DateInDisplayTimezone(ts)
}
