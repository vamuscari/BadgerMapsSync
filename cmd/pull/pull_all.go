package pull

import (
	"badgermaps/app"
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

	pullListener := func(e app.Event) {
		switch e.Type {
		case app.PullAllStart:
			bar = progressbar.NewOptions(-1,
				progressbar.OptionSetDescription(fmt.Sprintf("Starting pull for %s...", e.Source)),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
		case app.ResourceIDsFetched:
			count := e.Payload.(int)
			if bar != nil {
				bar.ChangeMax(count)
				bar.Describe(fmt.Sprintf("Found %d %s to pull.", count, e.Source))
			}
		case app.StoreSuccess:
			if bar != nil {
				bar.Add(1)
			}
		case app.PullAllError:
			err := e.Payload.(error)
			if bar != nil {
				bar.Clear()
			}
			log.Printf(color.RedString("An error occurred during pull: %v"), err)
		case app.PullAllComplete:
			if bar != nil {
				bar.Finish()
				fmt.Println(color.GreenString("✔ Pull for %s complete.", e.Source))
			}
		}
	}

	// Subscribe the listener to all relevant events
	a.Events.Subscribe(app.PullAllStart, pullListener)
	a.Events.Subscribe(app.ResourceIDsFetched, pullListener)
	a.Events.Subscribe(app.StoreSuccess, pullListener)
	a.Events.Subscribe(app.PullAllError, pullListener)
	a.Events.Subscribe(app.PullAllComplete, pullListener)

	// --- Execute Pull Operations ---
	log.Println(color.CyanString("Starting data pull from BadgerMaps API..."))

	logWrapper := func(message string) {
		log.Println(message)
	}

	if err := app.PullAllAccounts(a, top, logWrapper); err != nil {
		log.Fatalf(color.RedString("Failed to pull accounts: %v", err))
	}

	// Note: PullAllCheckins and PullAllRoutes are not yet instrumented with events.
	// To see progress for them, they would need to be updated similarly to PullAllAccounts.
	if err := app.PullAllCheckins(a, logWrapper); err != nil {
		log.Fatalf(color.RedString("Failed to pull checkins: %v", err))
	}

	if err := app.PullAllRoutes(a, logWrapper); err != nil {
		log.Fatalf(color.RedString("Failed to pull routes: %v", err))
	}

	log.Println(color.GreenString("✔ All data pulled successfully!"))
}
