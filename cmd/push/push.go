package push

import (
	"badgermaps/api"
	"badgermaps/app"
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// PushCmd creates a new push command
func PushCmd(App *app.State) *cobra.Command {
	App.VerifySetupOrExit()

	apiClient := api.NewAPIClient(App.Config.APIKey)

	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Send data to BadgerMaps API",
		Long:  `Push data from your local database to the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a data type to push (account, checkin, route, profile, all)")
			os.Exit(1)
		},
	}

	// Add subcommands
	pushCmd.AddCommand(pushAccountCmd(App, apiClient))

	return pushCmd
}

// pushAccountCmd creates a command to push a single account
func pushAccountCmd(App *app.State, apiClient *api.APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [id]",
		Short: "Push a single account to BadgerMaps",
		Long:  `Push a single account from your local database to the BadgerMaps API.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pushing account with ID: %s", args[0]))
			}

			// Parse account ID
			accountID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(color.RedString("Invalid account ID: %s", args[0]))
				os.Exit(1)
			}

			// TODO: Get account data from the local database

			// Create a map of data to push
			data := make(map[string]string)
			// TODO: Populate the data map with account data from the database

			// Push the account to the API
			_, err = apiClient.UpdateAccount(accountID, data)
			if err != nil {
				fmt.Println(color.RedString("Error pushing account: %v", err))
				os.Exit(1)
			}

			fmt.Println(color.GreenString("Successfully pushed account with ID: %d", accountID))
		},
	}
	return cmd
}