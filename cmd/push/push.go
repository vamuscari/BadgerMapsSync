package push

import (
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

	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Send data to BadgerMaps API",
		Long:  `Push data from your local database to the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a data type to push (account, checkin, route, profile, all)")
			os.Exit(1)
		},
	}

	pushCmd.AddCommand(pushAccountCmd(App))
	pushCmd.AddCommand(pushCheckinCmd(App))
	pushCmd.AddCommand(pushRouteCmd(App))
	pushCmd.AddCommand(pushProfileCmd(App))
	return pushCmd
}

func pushAccountCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [id]",
		Short: "Push a single account to BadgerMaps",
		Long:  `Push a single account from your local database to the BadgerMaps API.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pushing account with ID: %s", args[0]))
			}
			accountID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(color.RedString("Invalid account ID: %s", args[0]))
				os.Exit(1)
			}
			data := make(map[string]string)
			_, err = App.API.UpdateAccount(accountID, data)
			if err != nil {
				fmt.Println(color.RedString("Error pushing account: %v", err))
				os.Exit(1)
			}
			fmt.Println(color.GreenString("Successfully pushed account with ID: %d", accountID))
		},
	}
	return cmd
}

func pushCheckinCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkin [id]",
		Short: "Push a single checkin to BadgerMaps",
		Long:  `Push a single checkin from your local database to the BadgerMaps API.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Implement
		},
	}
	return cmd
}

func pushRouteCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [id]",
		Short: "Push a single route to BadgerMaps",
		Long:  `Push a single route from your local database to the BadgerMaps API.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Implement
		},
	}
	return cmd
}

func pushProfileCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Push user profile to BadgerMaps",
		Long:  `Push the user profile from your local database to the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Implement
		},
	}
	return cmd
}