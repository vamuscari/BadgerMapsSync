package main

import (
	"fmt"
	"os"

	"badgermapscli/app"
	"badgermapscli/cmd/config"
	"badgermapscli/cmd/pull"
	"badgermapscli/cmd/push"
	"badgermapscli/cmd/server"
	"badgermapscli/cmd/test"
	"badgermapscli/cmd/version"

	"github.com/spf13/cobra"
)

var (
	// Global configuration
	App *app.State
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
	configCmd := config.ConfigCmd(App)
	pullCmd := pull.PullCmd(App)
	pushCmd := push.PushCmd(App)
	serverCmd := server.ServerCmd(App)
	testCmd := test.TestCmd(App)
	versionCmd := version.VersionCmd()

	cobra.EnableCommandSorting = false

	// Add commands to root
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(versionCmd)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&App.Verbose, "verbose", "v", false, "Enable verbose output with additional details")
	rootCmd.PersistentFlags().BoolVarP(&App.Quiet, "quiet", "q", false, "Suppress all non-essential output")
	rootCmd.PersistentFlags().BoolVar(&App.Debug, "debug", false, "Enable debug mode with maximum verbosity")
	rootCmd.PersistentFlags().BoolVar(&App.NoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().StringVar(&App.CfgFile, "config", "", "Config file (default is $HOME/.badgermaps.yaml)")
}

func main() {
	App = app.NewApplication()
	bind()
	// Check if no arguments were provided
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
