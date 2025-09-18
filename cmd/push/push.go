package push

import (
	"badgermaps/app"
	"badgermaps/app/push"
	"badgermaps/database"
	"badgermaps/events"
	"fmt"
	"os"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// PushCmd creates a new push command
func PushCmd(App *app.App) *cobra.Command {
	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Send data to BadgerMaps API",
		Long:  `Push data from your local database to the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
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

			pushListener := func(e events.Event) {
				if e.Source != "accounts" {
					return
				}
				switch e.Type {
				case events.PushScanStart:
					a.Events.Dispatch(events.Infof("push", "Scanning for pending %s changes...", e.Source))
				                case events.PushItemError:
				                    err := e.Payload.(error)
				                    a.Events.Dispatch(events.Errorf("push", "An error occurred during push: %v", err))
				                case events.PushError:
				                    err := e.Payload.(error)
				                    a.Events.Dispatch(events.Errorf("push", "An error occurred during push scan: %v", err))
				                case events.PushComplete:					if bar != nil {
						bar.Finish()
					}
					errorCount := e.Payload.(int)
					a.Events.Dispatch(events.Infof("push", "✔ Push for %s complete. Encountered %d errors.", e.Source, errorCount))
				}
			}

			a.Events.Subscribe(events.PushScanStart, pushListener)
			a.Events.Subscribe(events.PushScanComplete, pushListener)
			a.Events.Subscribe(events.PushItemSuccess, pushListener)
			a.Events.Subscribe(events.PushItemError, pushListener)
			a.Events.Subscribe(events.PushComplete, pushListener)

			return push.RunPushAccounts(a)
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

			pushListener := func(e events.Event) {
				// Only listen for checkin events
				if e.Source != "checkins" {
					return
				}

				switch e.Type {
				case events.PushScanStart:
					a.Events.Dispatch(events.Infof("push", "Scanning for pending %s changes...", e.Source))
				case events.PushScanComplete:
					changes := e.Payload.([]database.CheckinPendingChange)
					if len(changes) > 0 {
						bar = progressbar.NewOptions(len(changes),
							progressbar.OptionSetDescription(fmt.Sprintf("Pushing %d %s changes", len(changes), e.Source)),
							progressbar.OptionSetWriter(os.Stderr),
							progressbar.OptionEnableColorCodes(true),
						)
					}
				case events.PushItemSuccess:
					if bar != nil {
						bar.Add(1)
					}
				case events.PushItemError:
					err := e.Payload.(error)
					a.Events.Dispatch(events.Errorf("push", "An error occurred during push: %v", err))
				case events.PushError:
					err := e.Payload.(error)
					a.Events.Dispatch(events.Errorf("push", "An error occurred during push scan: %v", err))
				case events.PushComplete:
					if bar != nil {
						bar.Finish()
					}
					errorCount := e.Payload.(int)
					a.Events.Dispatch(events.Infof("push", "✔ Push for %s complete. Encountered %d errors.", e.Source, errorCount))
				}
			}

			a.Events.Subscribe(events.PushScanStart, pushListener)
			a.Events.Subscribe(events.PushScanComplete, pushListener)
			a.Events.Subscribe(events.PushItemSuccess, pushListener)
			a.Events.Subscribe(events.PushItemError, pushListener)
			a.Events.Subscribe(events.PushComplete, pushListener)

			return push.RunPushCheckins(a)
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
			if err := push.RunPushAccounts(App); err != nil {
				return err
			}
			return push.RunPushCheckins(App)
		},
	}
	return cmd
}

