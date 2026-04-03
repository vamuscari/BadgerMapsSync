package pull

import (
	"badgermaps/app"
	"badgermaps/app/pull"
	appserver "badgermaps/app/server"
	"badgermaps/app/syncproxy"
	"badgermaps/events"
	"badgermaps/utils"
	"fmt"
	"log"
	"os"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var bar *progressbar.ProgressBar

func PullAllCmd(a *app.App) *cobra.Command {
	var top int

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Pull all accounts, checkins, and routes from BadgerMaps.",
		Long:  `Pulls all data including accounts, check-ins, and routes from the BadgerMaps API and stores it in the local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			runPullGroup(a, top)
		},
	}

	cmd.Flags().IntVar(&top, "top", 0, "Pull only the top N accounts (for testing).")

	return cmd
}

func runPullGroup(a *app.App, top int) {
	localRun := func() error {
		// Validate prerequisites before attempting local pull.
		if a.API == nil || a.API.APIKey == "" {
			return fmt.Errorf("api key is not configured. please run 'badgermaps config' to set up your API credentials")
		}

		if a.DB == nil {
			return fmt.Errorf("database is not configured. please run 'badgermaps config' to set up your database")
		}

		if !a.DB.IsConnected() {
			return fmt.Errorf("database is not connected. please check your database configuration")
		}

		log.SetOutput(utils.StderrWriter())

		pullListener := func(e events.Event) {
			switch e.Type {
			case "pull.group.start":
				bar = progressbar.NewOptions(-1,
					progressbar.OptionSetDescription(fmt.Sprintf("Starting pull for %s...", e.Source)),
					progressbar.OptionSetWriter(utils.StderrWriter()),
					progressbar.OptionSpinnerType(14),
					progressbar.OptionEnableColorCodes(true),
				)
			case "pull.ids_fetched":
				payload := e.Payload.(events.ResourceIDsFetchedPayload)
				if bar != nil {
					bar.ChangeMax(payload.Count)
					bar.Describe(fmt.Sprintf("Found %d %s to pull.", payload.Count, e.Source))
				}
			case "pull.store.success":
				if bar != nil {
					bar.Add(1)
				}
			case "pull.group.error":
				payload := e.Payload.(events.ErrorPayload)
				if bar != nil {
					bar.Clear()
				}
				a.Events.Dispatch(events.Errorf("pull", "An error occurred during pull: %v", payload.Error))
			case "pull.group.complete":
				if bar != nil {
					bar.Finish()
					a.Events.Dispatch(events.Infof("pull", "✔ Pull for %s complete.", e.Source))
				}
			}
		}

		a.Events.Subscribe("pull.*", pullListener)
		a.Events.Dispatch(events.Infof("pull", "Starting data pull from BadgerMaps API..."))

		if err := pull.PullGroupAccounts(a, top, nil); err != nil {
			return fmt.Errorf("failed to pull accounts: %w", err)
		}

		if err := pull.PullGroupCheckins(a, nil); err != nil {
			return fmt.Errorf("failed to pull checkins: %w", err)
		}

		if err := pull.PullGroupRoutes(a, nil); err != nil {
			return fmt.Errorf("failed to pull routes: %w", err)
		}

		if _, err := pull.PullProfile(a, nil); err != nil {
			return fmt.Errorf("failed to pull user profile: %w", err)
		}

		a.Events.Dispatch(events.Infof("pull", "✔ All data pulled successfully!"))
		return nil
	}

	if err := syncproxy.RunWithServerRouting(a, appserver.SyncModePull, "cli.pull.all", top, localRun); err != nil {
		fmt.Fprintf(utils.StderrWriter(), "Error: %v\n", err)
		os.Exit(1)
	}
}
