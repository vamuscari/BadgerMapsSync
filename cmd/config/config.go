package config

import (
	"badgermapscli/app"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ConfigCmd creates a new config command
func ConfigCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Run the interactive configuration setup",
		Long:  `Run the interactive setup to configure the BadgerMaps CLI application.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(color.CyanString("Starting interactive setup..."))
			if app.InteractiveSetup(App) {
				fmt.Println(color.GreenString("Setup completed successfully."))
			} else {
				fmt.Println(color.RedString("Setup did not complete successfully."))
			}
		},
	}
	return cmd
}
