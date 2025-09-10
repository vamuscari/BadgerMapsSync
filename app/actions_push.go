package app

import (
	"badgermaps/database"
	"encoding/json"
	"fmt"
)

// RunPushAccounts orchestrates pushing pending account changes to the API.
func RunPushAccounts(a *App, log func(string)) error {
	a.Events.Dispatch(Event{Type: PushScanStart, Source: "accounts"})
	changes, err := database.GetPendingAccountChanges(a.DB)
	if err != nil {
		err = fmt.Errorf("error getting pending account changes: %w", err)
		a.Events.Dispatch(Event{Type: PushError, Source: "accounts", Payload: err})
		return err
	}

	a.Events.Dispatch(Event{Type: PushScanComplete, Source: "accounts", Payload: changes})
	if len(changes) == 0 {
		log("No pending account changes to push.")
		a.Events.Dispatch(Event{Type: PushComplete, Source: "accounts", Payload: 0})
		return nil
	}

	errorCount := 0
	for _, change := range changes {
		a.Events.Dispatch(Event{Type: PushItemStart, Source: "accounts", Payload: change})
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
			a.Events.Dispatch(Event{Type: PushItemError, Source: "accounts", Payload: apiErr})
			database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "failed")
			errorCount++
		} else {
			a.Events.Dispatch(Event{Type: PushItemSuccess, Source: "accounts", Payload: change})
			database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "completed")
		}
	}
	a.Events.Dispatch(Event{Type: PushComplete, Source: "accounts", Payload: errorCount})
	log("Finished pushing account changes.")
	return nil
}

// RunPushCheckins orchestrates pushing pending check-in changes to the API.
func RunPushCheckins(a *App, log func(string)) error {
	a.Events.Dispatch(Event{Type: PushScanStart, Source: "checkins"})
	changes, err := database.GetPendingCheckinChanges(a.DB)
	if err != nil {
		err = fmt.Errorf("error getting pending check-in changes: %w", err)
		a.Events.Dispatch(Event{Type: PushError, Source: "checkins", Payload: err})
		return err
	}

	a.Events.Dispatch(Event{Type: PushScanComplete, Source: "checkins", Payload: changes})
	if len(changes) == 0 {
		log("No pending check-in changes to push.")
		a.Events.Dispatch(Event{Type: PushComplete, Source: "checkins", Payload: 0})
		return nil
	}

	errorCount := 0
	for _, change := range changes {
		a.Events.Dispatch(Event{Type: PushItemStart, Source: "checkins", Payload: change})
		database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "processing")

		var data map[string]string
		json.Unmarshal([]byte(change.Changes), &data)

		var apiErr error
		switch change.ChangeType {
		case "CREATE":
			_, apiErr = a.API.CreateCheckin(data)
		}

		if apiErr != nil {
			a.Events.Dispatch(Event{Type: PushItemError, Source: "checkins", Payload: apiErr})
			database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "failed")
			errorCount++
		} else {
			a.Events.Dispatch(Event{Type: PushItemSuccess, Source: "checkins", Payload: change})
			database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "completed")
		}
	}
	a.Events.Dispatch(Event{Type: PushComplete, Source: "checkins", Payload: errorCount})
	log("Finished pushing check-in changes.")
	return nil
}
