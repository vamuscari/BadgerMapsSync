package push

import (
	"badgermaps/app"
	"badgermaps/database"
	"badgermaps/events"
	"encoding/json"
	"fmt"
)

// RunPushAccounts orchestrates pushing pending account changes to the API.
func RunPushAccounts(a *app.App) error {
	a.Events.Dispatch(events.Event{Type: "push.scan.start", Source: "accounts", Payload: events.PushScanStartPayload{}})
	changes, err := database.GetPendingAccountChanges(a.DB)
	if err != nil {
		err = fmt.Errorf("error getting pending account changes: %w", err)
		a.Events.Dispatch(events.Event{Type: "push.error", Source: "accounts", Payload: events.ErrorPayload{Error: err}})
		return err
	}

	a.Events.Dispatch(events.Event{Type: "push.scan.complete", Source: "accounts", Payload: events.PushScanCompletePayload{Changes: changes}})
	if len(changes) == 0 {
		a.Events.Dispatch(events.Infof("push", "No pending account changes to push."))
		a.Events.Dispatch(events.Event{Type: "push.complete", Source: "accounts", Payload: events.PushCompletePayload{ErrorCount: 0}})
		return nil
	}

	errorCount := 0
	for _, change := range changes {
		a.Events.Dispatch(events.Event{Type: "push.item.start", Source: "accounts", Payload: events.PushItemStartPayload{Change: change}})
		database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "processing")

		data := make(map[string]string)
		if err := json.Unmarshal([]byte(change.Changes), &data); err != nil {
			parseErr := fmt.Errorf("invalid pending change payload (change_id=%d): %w", change.ChangeId, err)
			a.Events.Dispatch(events.Event{Type: "push.item.error", Source: "accounts", Payload: events.PushItemErrorPayload{Error: parseErr}})
			database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "failed")
			errorCount++
			continue
		}

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
			a.Events.Dispatch(events.Event{Type: "push.item.error", Source: "accounts", Payload: events.PushItemErrorPayload{Error: apiErr}})
			database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "failed")
			errorCount++
		} else {
			a.Events.Dispatch(events.Event{Type: "push.item.success", Source: "accounts", Payload: events.PushItemSuccessPayload{Change: change}})
			database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "completed")
		}
	}
	a.Events.Dispatch(events.Event{Type: "push.complete", Source: "accounts", Payload: events.PushCompletePayload{ErrorCount: errorCount}})
	a.Events.Dispatch(events.Infof("push", "Finished pushing account changes."))
	return nil
}

// RunPushCheckins orchestrates pushing pending check-in changes to the API.
func RunPushCheckins(a *app.App) error {
	a.Events.Dispatch(events.Event{Type: "push.scan.start", Source: "checkins", Payload: events.PushScanStartPayload{}})
	changes, err := database.GetPendingCheckinChanges(a.DB)
	if err != nil {
		err = fmt.Errorf("error getting pending check-in changes: %w", err)
		a.Events.Dispatch(events.Event{Type: "push.error", Source: "checkins", Payload: events.ErrorPayload{Error: err}})
		return err
	}

	a.Events.Dispatch(events.Event{Type: "push.scan.complete", Source: "checkins", Payload: events.PushScanCompletePayload{Changes: changes}})
	if len(changes) == 0 {
		a.Events.Dispatch(events.Infof("push", "No pending check-in changes to push."))
		a.Events.Dispatch(events.Event{Type: "push.complete", Source: "checkins", Payload: events.PushCompletePayload{ErrorCount: 0}})
		return nil
	}

	errorCount := 0
	for _, change := range changes {
		a.Events.Dispatch(events.Event{Type: "push.item.start", Source: "checkins", Payload: events.PushItemStartPayload{Change: change}})
		database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "processing")

		data := make(map[string]string)
		if err := json.Unmarshal([]byte(change.Changes), &data); err != nil {
			parseErr := fmt.Errorf("invalid pending change payload (change_id=%d): %w", change.ChangeId, err)
			a.Events.Dispatch(events.Event{Type: "push.item.error", Source: "checkins", Payload: events.PushItemErrorPayload{Error: parseErr}})
			database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "failed")
			errorCount++
			continue
		}

		var apiErr error
		switch change.ChangeType {
		case "CREATE":
			_, apiErr = a.API.CreateCheckin(data)
		}

		if apiErr != nil {
			a.Events.Dispatch(events.Event{Type: "push.item.error", Source: "checkins", Payload: events.PushItemErrorPayload{Error: apiErr}})
			database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "failed")
			errorCount++
		} else {
			a.Events.Dispatch(events.Event{Type: "push.item.success", Source: "checkins", Payload: events.PushItemSuccessPayload{Change: change}})
			database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "completed")
		}
	}
	a.Events.Dispatch(events.Event{Type: "push.complete", Source: "checkins", Payload: events.PushCompletePayload{ErrorCount: errorCount}})
	a.Events.Dispatch(events.Infof("push", "Finished pushing check-in changes."))
	return nil
}
