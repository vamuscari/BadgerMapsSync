package app

import (
	"badgermaps/database"
	"encoding/json"
	"fmt"
)

// RunPushAccounts orchestrates pushing pending account changes to the API.
func RunPushAccounts(a *App, log func(string)) error {
	log("Pushing pending account changes...")
	changes, err := database.GetPendingAccountChanges(a.DB)
	if err != nil {
		return fmt.Errorf("error getting pending account changes: %w", err)
	}

	if len(changes) == 0 {
		log("No pending account changes to push.")
		return nil
	}

	for _, change := range changes {
		log(fmt.Sprintf("Processing account change %d (%s)...", change.ChangeId, change.ChangeType))
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
			log(fmt.Sprintf("Error pushing account change %d: %v", change.ChangeId, apiErr))
			database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "failed")
		} else {
			log(fmt.Sprintf("Successfully pushed account change %d.", change.ChangeId))
			database.UpdatePendingChangeStatus(a.DB, "AccountsPendingChanges", change.ChangeId, "completed")
		}
	}
	log("Finished pushing account changes.")
	return nil
}

// RunPushCheckins orchestrates pushing pending check-in changes to the API.
func RunPushCheckins(a *App, log func(string)) error {
	log("Pushing pending check-in changes...")
	changes, err := database.GetPendingCheckinChanges(a.DB)
	if err != nil {
		return fmt.Errorf("error getting pending check-in changes: %w", err)
	}

	if len(changes) == 0 {
		log("No pending check-in changes to push.")
		return nil
	}

	for _, change := range changes {
		log(fmt.Sprintf("Processing check-in change %d (%s)...", change.ChangeId, change.ChangeType))
		database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "processing")

		var data map[string]string
		json.Unmarshal([]byte(change.Changes), &data)

		var apiErr error
		switch change.ChangeType {
		case "CREATE":
			_, apiErr = a.API.CreateCheckin(data)
		}

		if apiErr != nil {
			log(fmt.Sprintf("Error pushing check-in change %d: %v", change.ChangeId, apiErr))
			database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "failed")
		} else {
			log(fmt.Sprintf("Successfully pushed check-in change %d.", change.ChangeId))
			database.UpdatePendingChangeStatus(a.DB, "AccountCheckinsPendingChanges", change.ChangeId, "completed")
		}
	}
	log("Finished pushing check-in changes.")
	return nil
}
