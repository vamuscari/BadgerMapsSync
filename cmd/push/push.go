package push

import (
	"badgermaps/app"
	"badgermaps/database"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
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
	pushCmd.AddCommand(listCmd(App))
	return pushCmd
}

func pushAccountsCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Push pending account changes to BadgerMaps",
		Long:  `Push pending account changes from your local database to the BadgerMaps API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var bar *progressbar.ProgressBar

			pushListener := func(e app.Event) {
				switch e.Type {
				case app.PushScanStart:
					fmt.Println(color.CyanString("Scanning for pending %s changes...", e.Source))
				case app.PushScanComplete:
					changes := e.Payload.([]database.AccountPendingChange)
					if len(changes) > 0 {
						bar = progressbar.NewOptions(len(changes),
							progressbar.OptionSetDescription(fmt.Sprintf("Pushing %d %s changes", len(changes), e.Source)),
							progressbar.OptionSetWriter(os.Stderr),
							progressbar.OptionEnableColorCodes(true),
						)
					}
				case app.PushItemSuccess:
					if bar != nil {
						bar.Add(1)
					}
				case app.PushItemError:
					err := e.Payload.(error)
					fmt.Println(color.RedString("An error occurred during push: %v", err))
				case app.PushComplete:
					if bar != nil {
						bar.Finish()
					}
					errorCount := e.Payload.(int)
					fmt.Println(color.GreenString("✔ Push for %s complete. Encountered %d errors.", e.Source, errorCount))
				}
			}

			a.Events.Subscribe(app.PushScanStart, pushListener)
			a.Events.Subscribe(app.PushScanComplete, pushListener)
			a.Events.Subscribe(app.PushItemSuccess, pushListener)
			a.Events.Subscribe(app.PushItemError, pushListener)
			a.Events.Subscribe(app.PushComplete, pushListener)

			return app.RunPushAccounts(a, func(s string) {}) // Pass an empty logger
		},
	}
	return cmd
}

func pushCheckinsCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins",
		Short: "Push pending check-in changes to BadgerMaps",
		Long:  `Push pending check-in changes from your local database to the BadgerMaps API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var bar *progressbar.ProgressBar

			pushListener := func(e app.Event) {
				// Only listen for checkin events
				if e.Source != "checkins" {
					return
				}

				switch e.Type {
				case app.PushScanStart:
					fmt.Println(color.CyanString("Scanning for pending %s changes...", e.Source))
				case app.PushScanComplete:
					changes := e.Payload.([]database.CheckinPendingChange)
					if len(changes) > 0 {
						bar = progressbar.NewOptions(len(changes),
							progressbar.OptionSetDescription(fmt.Sprintf("Pushing %d %s changes", len(changes), e.Source)),
							progressbar.OptionSetWriter(os.Stderr),
							progressbar.OptionEnableColorCodes(true),
						)
					}
				case app.PushItemSuccess:
					if bar != nil {
						bar.Add(1)
					}
				case app.PushItemError:
					err := e.Payload.(error)
					fmt.Println(color.RedString("An error occurred during push: %v", err))
				case app.PushComplete:
					if bar != nil {
						bar.Finish()
					}
					errorCount := e.Payload.(int)
					fmt.Println(color.GreenString("✔ Push for %s complete. Encountered %d errors.", e.Source, errorCount))
				}
			}

			a.Events.Subscribe(app.PushScanStart, pushListener)
			a.Events.Subscribe(app.PushScanComplete, pushListener)
			a.Events.Subscribe(app.PushItemSuccess, pushListener)
			a.Events.Subscribe(app.PushItemError, pushListener)
			a.Events.Subscribe(app.PushComplete, pushListener)

			return app.RunPushCheckins(a, func(s string) {}) // Pass an empty logger
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

