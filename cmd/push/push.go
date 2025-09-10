package push

import (
	"badgermaps/app"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// PushCmd creates a new push command
func PushCmd(App *app.App) *cobra.Command {
	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Send data to BadgerMaps API",
		Long:  `Push data from your local database to the BadgerMaps API.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			App.VerifySetupOrExit(cmd)
		},
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
		RunE: func(cmd *cobra.Command, args []string) error {
			log := func(message string) {
				if strings.Contains(message, "Error") || strings.Contains(message, "error") {
					fmt.Println(color.RedString(message))
				} else if strings.Contains(message, "Successfully") || strings.Contains(message, "Finished") {
					fmt.Println(color.GreenString(message))
				} else {
					fmt.Println(color.CyanString(message))
				}
			}
			return app.RunPushAccounts(App, log)
		},
	}
	return cmd
}

func pushCheckinsCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins",
		Short: "Push pending check-in changes to BadgerMaps",
		Long:  `Push pending check-in changes from your local database to the BadgerMaps API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log := func(message string) {
				if strings.Contains(message, "Error") || strings.Contains(message, "error") {
					fmt.Println(color.RedString(message))
				} else if strings.Contains(message, "Successfully") || strings.Contains(message, "Finished") {
					fmt.Println(color.GreenString(message))
				} else {
					fmt.Println(color.CyanString(message))
				}
			}
			return app.RunPushCheckins(App, log)
		},
	}
	return cmd
}

func pushAllCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Push all pending changes to BadgerMaps",
		Long:  `Push all pending changes (accounts and check-ins) from your local database to the BadgerMaps API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log := func(message string) {
				if strings.Contains(message, "Error") || strings.Contains(message, "error") {
					fmt.Println(color.RedString(message))
				} else if strings.Contains(message, "Successfully") || strings.Contains(message, "Finished") {
					fmt.Println(color.GreenString(message))
				} else {
					fmt.Println(color.CyanString(message))
				}
			}
			if err := app.RunPushAccounts(App, log); err != nil {
				return err
			}
			return app.RunPushCheckins(App, log)
		},
	}
	return cmd
}

