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

	"github.com/spf13/cobra"
)

var (
	// Global configuration
	App *app.App
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "badgermaps",
	Short: "BadgerMaps CLI - Command line interface for BadgerMaps",
	Long: `BadgerMaps CLI is a command line interface for interacting with the BadgerMaps API.
It allows you to push and pull data, run in server mode, and perform various utility operations.`,
}

func bind() {
	// Create commands with the configuration
	pullCmd := pull.PullCmd(App)
	pushCmd := push.PushCmd(App)
	serverCmd := server.ServerCmd(App)
	testCmd := test.TestCmd(App)
	configCmd := config.ConfigCmd(App)
	versionCmd := version.VersionCmd()

	cobra.EnableCommandSorting = false

	// Add commands to root
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&App.State.Verbose, "verbose", "v", false, "Enable verbose output with additional details")
	rootCmd.PersistentFlags().BoolVarP(&App.State.Quiet, "quiet", "q", false, "Suppress all non-essential output")
	rootCmd.PersistentFlags().BoolVar(&App.State.Debug, "debug", false, "Enable debug mode with maximum verbosity")
	rootCmd.PersistentFlags().BoolVar(&App.State.NoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().StringVar(&App.CfgFile, "config", "", "Config file (default is $HOME/.badgermaps.yaml)")
}

func main() {
	App = app.NewApplication()
	bind()
	// Check if no arguments were provided
	if err := rootCmd.Execute(); err != nil {
		if App.State.Debug {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}
}
