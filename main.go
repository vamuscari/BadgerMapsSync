package main

import (
	"fmt"
	"os"

	"badgermaps/app"
	"badgermaps/cmd/config"
	"badgermaps/cmd/pull"
	"badgermaps/cmd/push"
	"badgermaps/cmd/server"
	"badgermaps/cmd/test"
	"badgermaps/cmd/version"
	"badgermaps/gui"

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

// createRootCmd configures and returns the root cobra command
func createRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "badgermaps",
		Short: "BadgerMaps CLI - Command line interface for BadgerMaps",
		Long: `BadgerMaps CLI is a command line interface for interacting with the BadgerMaps API.
It allows you to push and pull data, run in server mode, and perform various utility operations.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Don't load config for version or help commands
			if cmd.Name() == "version" || cmd.Name() == "help" {
				return
			}
			App.VerifySetupOrExit(cmd)
		},
	}

	// Create and add commands
	pullCmd := pull.PullCmd(App)
	pushCmd := push.PushCmd(App)
	serverCmd := server.ServerCmd(App)
	testCmd := test.TestCmd(App)
	configCmd := config.ConfigCmd(App)
	versionCmd := version.VersionCmd()

	rootCmd.AddCommand(pushCmd, pullCmd, serverCmd, testCmd, configCmd, versionCmd)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&App.State.Verbose, "verbose", "v", false, "Enable verbose output with additional details")
	rootCmd.PersistentFlags().BoolVarP(&App.State.Quiet, "quiet", "q", false, "Suppress all non-essential output")
	rootCmd.PersistentFlags().BoolVar(&App.State.Debug, "debug", false, "Enable debug mode with maximum verbosity")
	rootCmd.PersistentFlags().BoolVar(&App.State.NoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().StringVar(App.State.ConfigFile, "config", "", "Config file (default is $HOME/.badgermaps.yaml)")
	rootCmd.PersistentFlags().StringVar(App.State.EnvFile, "env", "", "Env file (default is $PWD/.env).")

	return rootCmd
}

func main() {
	// Initialize the core application
	App = app.NewApplication()
	if App.DB != nil {
		defer App.DB.Close()
	}

	// Check if the app should run in GUI or CLI mode
	// os.Args[0] is the program name, so len > 1 means subcommands were passed
	if len(os.Args) > 1 {
		// CLI Mode
		runCLI()
	} else {
		// GUI Mode
		runGUI()
	}
}

func runCLI() {
	rootCmd := createRootCmd()
	if err := rootCmd.Execute(); err != nil {
		if App.State.Debug {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}
}

func runGUI() {
	// For the GUI, we need to ensure the basic configuration is loaded
	// so the app can function. We can trigger the same logic Cobra uses.
	App.VerifySetupOrExit(nil) // Passing nil as we don't have a command context

	// Launch the Fyne GUI
	gui.Launch(App, AppIcon)
}

