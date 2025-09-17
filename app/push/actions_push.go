package push

import (
	"badgermaps/app"
	"badgermaps/events"
	"badgermaps/database"
	"encoding/json"
	"fmt"
)

// RunPushAccounts orchestrates pushing pending account changes to the API.
func RunPushAccounts(a *app.App) error {
	a.Events.Dispatch(events.Event{Type: events.PushScanStart, Source: "accounts"})
	changes, err := database.GetPendingAccountChanges(a.DB)
	if err != nil {
		err = fmt.Errorf("error getting pending account changes: %w", err)
		a.Events.Dispatch(events.Event{Type: events.PushError, Source: "accounts", Payload: err})
		return err
	}

	a.Events.Dispatch(events.Event{Type: events.PushScanComplete, Source: "accounts", Payload: changes})
	if len(changes) == 0 {
		a.Events.Dispatch(events.Infof("push", "No pending account changes to push."))
		a.Events.Dispatch(events.Event{Type: events.PushComplete, Source: "accounts", Payload: 0})
		return nil
	}

	errorCount := 0
	for _, change := range changes {
		a.Events.Dispatch(events.Event{Type: events.PushItemStart, Source: "accounts", Payload: change})
		database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "processing")

		var data map[string]string
		json.Unmarshal([]byte(change.Changes), &data)

		var apiErr error
		switch change.ChangeType {
		case "CREATE":
			_, apiErr = a.API.CreateAccount(data)
		case "UPDATE":
			_, apiErr = a.API.UpdateAccount(change.AccountId, data)
		case "DELETE":
			apiErr = a.API.DeleteAccount(change.AccountId)
		}

		if apiErr != nil {
			a.Events.Dispatch(events.Event{Type: events.PushItemError, Source: "accounts", Payload: apiErr})
			database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "failed")
			errorCount++
		} else {
			a.Events.Dispatch(events.Event{Type: events.PushItemSuccess, Source: "accounts", Payload: change})
			database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "completed")
		}
	}
	a.Events.Dispatch(events.Event{Type: events.PushComplete, Source: "accounts", Payload: errorCount})
	a.Events.Dispatch(events.Infof("push", "Finished pushing account changes."))
	return nil
}

// RunPushCheckins orchestrates pushing pending check-in changes to the API.
func RunPushCheckins(a *app.App) error {
	a.Events.Dispatch(events.Event{Type: events.PushScanStart, Source: "checkins"})
	changes, err := database.GetPendingCheckinChanges(a.DB)
	if err != nil {
		err = fmt.Errorf("error getting pending check-in changes: %w", err)
		a.Events.Dispatch(events.Event{Type: events.PushError, Source: "checkins", Payload: err})
		return err
	}

	a.Events.Dispatch(events.Event{Type: events.PushScanComplete, Source: "checkins", Payload: changes})
	if len(changes) == 0 {
		a.Events.Dispatch(events.Infof("push", "No pending check-in changes to push."))
		a.Events.Dispatch(events.Event{Type: events.PushComplete, Source: "checkins", Payload: 0})
		return nil
	}

	errorCount := 0
	for _, change := range changes {
		a.Events.Dispatch(events.Event{Type: events.PushItemStart, Source: "checkins", Payload: change})
		database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "processing")

		var data map[string]string
		json.Unmarshal([]byte(change.Changes), &data)

		var apiErr error
		switch change.ChangeType {
		case "CREATE":
			_, apiErr = a.API.CreateCheckin(data)
		}

		if apiErr != nil {
			a.Events.Dispatch(events.Event{Type: events.PushItemError, Source: "checkins", Payload: apiErr})
			database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "failed")
			errorCount++
		} else {
			a.Events.Dispatch(events.Event{Type: events.PushItemSuccess, Source: "checkins", Payload: change})
			database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "completed")
		}
	}
	a.Events.Dispatch(events.Event{Type: events.PushComplete, Source: "checkins", Payload: errorCount})
	a.Events.Dispatch(events.Infof("push", "Finished pushing check-in changes."))
	return nil
}
