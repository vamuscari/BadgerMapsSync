package pull

import (
	"badgermaps/app"
	"badgermaps/app/pull"
	"badgermaps/events"
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
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
			runPullAll(a, top)
		},
	}

	cmd.Flags().IntVar(&top, "top", 0, "Pull only the top N accounts (for testing).")

	return cmd
}

func runPullAll(a *app.App, top int) {
	log.SetOutput(os.Stderr) // Configure logger to write to stderr

	pullListener := func(e events.Event) {
		switch e.Type {
		case events.PullAllStart:
			bar = progressbar.NewOptions(-1,
				progressbar.OptionSetDescription(fmt.Sprintf("Starting pull for %s...", e.Source)),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
		case events.ResourceIDsFetched:
			count := e.Payload.(int)
			if bar != nil {
				bar.ChangeMax(count)
				bar.Describe(fmt.Sprintf("Found %d %s to pull.", count, e.Source))
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
	a.Events.Subscribe(events.PullAllStart, pullListener)
	a.Events.Subscribe(events.ResourceIDsFetched, pullListener)
	a.Events.Subscribe(events.StoreSuccess, pullListener)
	a.Events.Subscribe(events.PullAllError, pullListener)
	a.Events.Subscribe(events.PullAllComplete, pullListener)

	// --- Execute Pull Operations ---
	a.Events.Dispatch(events.Infof("pull", "Starting data pull from BadgerMaps API..."))

	if err := pull.PullAllAccounts(a, top, nil); err != nil {
		a.Events.Dispatch(events.Errorf("pull", "Failed to pull accounts: %v", err))
		os.Exit(1)
	}

	if err := pull.PullAllCheckins(a, nil); err != nil {
		a.Events.Dispatch(events.Errorf("pull", "Failed to pull checkins: %v", err))
		os.Exit(1)
	}

	if err := pull.PullAllRoutes(a, nil); err != nil {
		a.Events.Dispatch(events.Errorf("pull", "Failed to pull routes: %v", err))
		os.Exit(1)
	}

	if err := pull.PullProfile(a, nil); err != nil {
		a.Events.Dispatch(events.Errorf("pull", "Failed to pull user profile: %v", err))
		os.Exit(1)
	}

	a.Events.Dispatch(events.Infof("pull", "✔ All data pulled successfully!"))
}
