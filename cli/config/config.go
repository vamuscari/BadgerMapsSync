package config

import (
	"badgermaps/app"
	"badgermaps/events"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// ConfigCmd creates a new config command
func ConfigCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Run the interactive configuration setup",
		Long:  `Run the interactive setup to configure the BadgerMaps CLI application.`,
		Run: func(cmd *cobra.Command, args []string) {
			// The configuration is already loaded by the PersistentPreRun in main.go
			// We just need to run the interactive setup.

			// If a config file was specified and doesn't exist, create it.
			if a.ConfigFile != "" {
				if _, err := os.Stat(a.ConfigFile); os.IsNotExist(err) {
					file, err := os.Create(a.ConfigFile)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
						os.Exit(1)
					}
					file.Close()
				}
			}

			// Suppress logs during interactive setup to avoid cluttering the prompts.
			// Without this, log messages like "INFO [config] Configuration detected: config.yaml"
			// would appear in the middle of user input, making the interactive experience confusing.
			// The InteractiveSetup() function uses direct fmt.Println() calls for important
			// feedback (like database connection status), so users still get the information
			// they need without the timestamp/source prefix clutter.
			previousQuiet := a.State.Quiet
			a.State.Quiet = true

			success := a.InteractiveSetup()

			// Restore previous quiet state
			a.State.Quiet = previousQuiet

			if success {
				a.Events.Dispatch(events.Infof("config", "Setup completed successfully."))
			} else {
				a.Events.Dispatch(events.Errorf("config", "Setup did not complete successfully."))
			}
		},
	}

	return cmd
}
