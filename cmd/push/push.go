package push

import (
	"badgermaps/app"
	"badgermaps/database"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// PushCmd creates a new push command
func PushCmd(App *app.App) *cobra.Command {
	App.VerifySetupOrExit()

	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Send data to BadgerMaps API",
		Long:  `Push data from your local database to the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a data type to push (accounts, checkins, all)")
			os.Exit(1)
		},
	}

	pushCmd.AddCommand(pushAccountsCmd(App))
	pushCmd.AddCommand(pushCheckinsCmd(App))
	pushCmd.AddCommand(pushAllCmd(App))
	return pushCmd
}

func pushAccountsCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Push pending account changes to BadgerMaps",
		Long:  `Push pending account changes from your local database to the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			pushAccounts(App)
		},
	}
	return cmd
}

func pushCheckinsCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins",
		Short: "Push pending check-in changes to BadgerMaps",
		Long:  `Push pending check-in changes from your local database to the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			pushCheckins(App)
		},
	}
	return cmd
}

func pushAllCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Push all pending changes to BadgerMaps",
		Long:  `Push all pending changes (accounts and check-ins) from your local database to the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			pushAccounts(App)
			pushCheckins(App)
		},
	}
	return cmd
}

func pushAccounts(App *app.App) {
	fmt.Println(color.CyanString("Pushing pending account changes..."))
	changes, err := database.GetPendingAccountChanges(App.DB)
	if err != nil {
		fmt.Println(color.RedString("Error getting pending account changes: %v", err))
		return
	}

	if len(changes) == 0 {
		fmt.Println(color.GreenString("No pending account changes to push."))
		return
	}

	for _, change := range changes {
		database.UpdatePendingChangeStatus(App.DB, "AccountsPendingChanges", change.ChangeId, "processing")

		var data map[string]string
		json.Unmarshal([]byte(change.Changes), &data)

		var apiErr error
		switch change.ChangeType {
		case "CREATE":
			_, apiErr = App.API.CreateAccount(data)
		case "UPDATE":
			_, apiErr = App.API.UpdateAccount(change.AccountId, data)
		case "DELETE":
			apiErr = App.API.DeleteAccount(change.AccountId)
		}

		if apiErr != nil {
			fmt.Println(color.RedString("Error pushing account change %d: %v", change.ChangeId, apiErr))
			database.UpdatePendingChangeStatus(App.DB, "AccountsPendingChanges", change.ChangeId, "failed")
		} else {
			fmt.Println(color.GreenString("Successfully pushed account change %d.", change.ChangeId))
			database.UpdatePendingChangeStatus(App.DB, "AccountsPendingChanges", change.ChangeId, "completed")
		}
	}
}

func pushCheckins(App *app.App) {
	fmt.Println(color.CyanString("Pushing pending check-in changes..."))
	changes, err := database.GetPendingCheckinChanges(App.DB)
	if err != nil {
		fmt.Println(color.RedString("Error getting pending check-in changes: %v", err))
		return
	}

	if len(changes) == 0 {
		fmt.Println(color.GreenString("No pending check-in changes to push."))
		return
	}

	for _, change := range changes {
		database.UpdatePendingChangeStatus(App.DB, "AccountCheckinsPendingChanges", change.ChangeId, "processing")

		var data map[string]string
		json.Unmarshal([]byte(change.Changes), &data)

		var apiErr error
		switch change.ChangeType {
		case "CREATE":
			_, apiErr = App.API.CreateCheckin(data)
			// The BadgerMaps API does not currently support updating or deleting check-ins.
			// case "UPDATE":
			// 	_, apiErr = App.API.UpdateCheckin(change.CheckinId, data)
			// case "DELETE":
			// 	apiErr = App.API.DeleteCheckin(change.CheckinId)
		}

		if apiErr != nil {
			fmt.Println(color.RedString("Error pushing check-in change %d: %v", change.ChangeId, apiErr))
			database.UpdatePendingChangeStatus(App.DB, "AccountCheckinsPendingChanges", change.ChangeId, "failed")
		} else {
			fmt.Println(color.GreenString("Successfully pushed check-in change %d.", change.ChangeId))
			database.UpdatePendingChangeStatus(App.DB, "AccountCheckinsPendingChanges", change.ChangeId, "completed")
		}
	}
}
