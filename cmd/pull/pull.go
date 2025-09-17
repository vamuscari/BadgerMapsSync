package pull

import (
	"badgermaps/app"
	"badgermaps/app/pull"
	"badgermaps/events"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// PullCmd creates a new pull command
func PullCmd(App *app.App) *cobra.Command {
	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Retrieve data from BadgerMaps API",
		Long:  `Pull data from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a data type to pull (account, checkin, route, profile, all)")
			os.Exit(1)
		},
	}

	pullCmd.AddCommand(pullAccountCmd(App))
	pullCmd.AddCommand(pullAccountsCmd(App))
	pullCmd.AddCommand(pullCheckinCmd(App))
	pullCmd.AddCommand(pullCheckinsCmd(App))
	pullCmd.AddCommand(pullRouteCmd(App))
	pullCmd.AddCommand(pullRoutesCmd(App))
	pullCmd.AddCommand(pullProfileCmd(App))
	pullCmd.AddCommand(PullAllCmd(App))

	return pullCmd
}

func pullAccountCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [id]",
		Short: "Pull a single account from BadgerMaps",
		Long:  `Pull a single account from the BadgerMaps API and store it in the local database.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid account ID: %s", args[0])
			}

			listener := func(e events.Event) {
				if e.Source != "account" {
					return
				}
				switch e.Type {
				case events.PullComplete:
					fmt.Println(color.GreenString("Successfully pulled account %d.", e.Payload.(int)))
				case events.PullError:
					fmt.Println(color.RedString("Error: Failed to pull account %d. The API returned an error.", accountID))
					fmt.Println(color.YellowString("Details: %v", e.Payload.(error)))
				}
			}
			App.Events.Subscribe(events.PullComplete, listener)
			App.Events.Subscribe(events.PullError, listener)

			return pull.PullAccount(App, accountID)
		},
	}
	return cmd
}

func pullAccountsCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Pull all accounts from BadgerMaps",
		Long:  `Pull all accounts from the BadgerMaps API and store them in the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			bar := progressbar.NewOptions(-1,
				progressbar.OptionSetDescription("Pulling accounts..."),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
			defer bar.Close()

			return pull.PullAllAccounts(App, 0, nil)
		},
	}
	return cmd
}

func pullCheckinCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkin [id]",
		Short: "Pull a single checkin from BadgerMaps",
		Long:  `Pull a single checkin from the BadgerMaps API and store it in the local database.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			checkinID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid checkin ID: %s", args[0])
			}
			err = pull.PullCheckin(App, checkinID)
			if err != nil {
				// Provide a cleaner error message to the user
				fmt.Println(color.RedString("Error: Failed to pull check-in %d. The API returned an error.", checkinID))
				fmt.Println(color.YellowString("Details: %v", err))
				return nil // Return nil to prevent Cobra from printing usage
			}
			return nil
		},
	}
	return cmd
}

func pullCheckinsCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins",
		Short: "Pull all checkins from BadgerMaps",
		Long:  `Pull all checkins from the BadgerMaps API and store them in the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var bar *progressbar.ProgressBar

			pullListener := func(e events.Event) {
				if e.Source != "checkins" {
					return
				}
				switch e.Type {
				case events.PullAllStart:
					bar = progressbar.NewOptions(-1,
						progressbar.OptionSetDescription("Pulling checkins..."),
						progressbar.OptionSetWriter(os.Stderr),
						progressbar.OptionSpinnerType(14),
						progressbar.OptionEnableColorCodes(true),
					)
				case events.ResourceIDsFetched:
					count := e.Payload.(int)
					if bar != nil {
						bar.ChangeMax(count)
						bar.Describe(fmt.Sprintf("Found %d accounts to pull checkins from.", count))
					}
				case events.StoreSuccess:
					if bar != nil {
						bar.Add(1)
					}
				case events.PullAllError:
					err := e.Payload.(error)
					if bar != nil {
						bar.Clear()
					}
					log.Printf(color.RedString("An error occurred during pull: %v"), err)
				case events.PullAllComplete:
					if bar != nil {
						bar.Finish()
						fmt.Println(color.GreenString("✔ Pull for %s complete.", e.Source))
					}
				}
			}

			// Subscribe the listener to all relevant events
			App.Events.Subscribe(events.PullAllStart, pullListener)
			App.Events.Subscribe(events.ResourceIDsFetched, pullListener)
			App.Events.Subscribe(events.StoreSuccess, pullListener)
			App.Events.Subscribe(events.PullAllError, pullListener)
			App.Events.Subscribe(events.PullAllComplete, pullListener)

			err := pull.PullAllCheckins(App, nil)
			if bar != nil && !bar.IsFinished() {
				bar.Finish()
			}
			return err
		},
	}
	return cmd
}

func pullRouteCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [id]",
		Short: "Pull a single route from BadgerMaps",
		Long:  `Pull a single route from the BadgerMaps API and store it in the local database.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			routeID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid route ID: %s", args[0])
			}
			err = pull.PullRoute(App, routeID)
			if err != nil {
				// Provide a cleaner error message to the user
				fmt.Println(color.RedString("Error: Failed to pull route %d. The API returned an error.", routeID))
				fmt.Println(color.YellowString("Details: %v", err))
				return nil // Return nil to prevent Cobra from printing usage
			}
			return nil
		},
	}
	return cmd
}

func pullRoutesCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Pull all routes from BadgerMaps",
		Long:  `Pull all routes from the BadgerMaps API and store them in the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			bar := progressbar.NewOptions(-1,
				progressbar.OptionSetDescription("Pulling routes..."),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
			defer bar.Close()

			return pull.PullAllRoutes(App, nil)
		},
	}
	return cmd
}

func pullProfileCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Pull user profile from BadgerMaps",
		Long:  `Pull the user profile from the BadgerMaps API and store it in the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return pull.PullProfile(App, nil)
		},
	}
	return cmd
}
