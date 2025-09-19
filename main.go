package main

import (
	"fmt"
	"os"

	"badgermaps/app"
	"badgermaps/cli/action"
	"badgermaps/cli/config"
	"badgermaps/cli/pull"
	"badgermaps/cli/push"
	"badgermaps/cli/server"
	"badgermaps/cli/test"
	"badgermaps/cli/version"
	"badgermaps/database"
	"badgermaps/events"
	"badgermaps/gui"
	"badgermaps/utils"

	_ "embed"

	"fyne.io/fyne/v2"
	"github.com/spf13/cobra"
)

//go:embed assets/icon.png
var iconBytes []byte
var AppIcon = &fyne.StaticResource{StaticName: "icon.png", StaticContent: iconBytes}

var (
	// Global application instance
	App *app.App
)

var guiFlag bool

// createRootCmd configures and returns the root cobra command
func createRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "badgermaps",
		Short: "BadgerMaps CLI - Command line interface for BadgerMaps",
		Long: `BadgerMaps CLI is a command line interface for interacting with the BadgerMaps API.
It allows you to push and pull data, run in server mode, and perform various utility operations.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Don't load config for version, help, or gui commands
			if cmd.Name() == "version" || cmd.Name() == "help" || (cmd.Name() == "badgermaps" && guiFlag) {
				return
			}
			App.EnsureConfig(false)
		},
		Run: func(cmd *cobra.Command, args []string) {
			if guiFlag && gui.Enabled {
				App.EnsureConfig(true) // Load config specifically for GUI
				App.State.IsGui = true
				gui.Run(App, AppIcon)
			} else {
				cmd.Help()
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			if App.DB != nil {
				err := database.LogCommand(App.DB, cmd.Name(), args, true, "")
				if err != nil {
					fmt.Printf("Error logging command: %v\n", err)
				}
			}
		},
	}

	// Create and add commands
	pullCmd := pull.PullCmd(App)
	pushCmd := push.PushCmd(App)
	serverCmd := server.ServerCmd(App)
	testCmd := test.TestCmd(App)
	configCmd := config.ConfigCmd(App)
	versionCmd := version.VersionCmd()
	actionCmd := action.ActionCmd(App)

	rootCmd.AddCommand(pushCmd, pullCmd, serverCmd, testCmd, configCmd, versionCmd, actionCmd)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&App.State.Verbose, "verbose", "v", false, "Enable verbose output with additional details")
	rootCmd.PersistentFlags().BoolVarP(&App.State.Quiet, "quiet", "q", false, "Suppress all non-essential output")
	rootCmd.PersistentFlags().BoolVar(&App.State.Debug, "debug", false, "Enable debug mode with maximum verbosity")
	rootCmd.PersistentFlags().BoolVar(&App.State.NoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVar(&App.State.NoInput, "no-input", false, "Disable interactive prompts")
	rootCmd.PersistentFlags().StringVar(App.State.ConfigFile, "config", "", "Config file (default is $HOME/.badgermaps.yaml)")
	rootCmd.Flags().BoolVar(&guiFlag, "gui", false, "Launch the graphical user interface")

	return rootCmd
}

func main() {
	// Initialize the core application
	App = app.NewApp()

	utils.InitColors(App.State)
	if App.DB != nil {
		defer App.DB.Close()
	}

	// If no command is specified, and the GUI is available, launch it.
	if len(os.Args) == 1 && gui.Enabled {
		App.State.IsGui = true
		gui.Run(App, AppIcon)
		return // Exit after GUI closes
	}

	// Otherwise, run the command-line interface
	rootCmd := createRootCmd()
	if err := rootCmd.Execute(); err != nil {
		App.Events.Dispatch(events.Errorf("main", "command execution failed: %v", err))
		os.Exit(1)
	}
}
