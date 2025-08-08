package main

import (
	"fmt"
	"os"

	"badgermapscli/cmd/auth"
	"badgermapscli/cmd/autocomplete"
	"badgermapscli/cmd/help"
	"badgermapscli/cmd/pull"
	"badgermapscli/cmd/push"
	"badgermapscli/cmd/search"
	"badgermapscli/cmd/server"
	"badgermapscli/cmd/test"
	"badgermapscli/cmd/utils"
	"badgermapscli/cmd/version"
	"badgermapscli/common"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Global flags
	verbose bool
	quiet   bool
	debug   bool
	noColor bool
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "badgermaps",
	Short: "BadgerMaps CLI - Command line interface for BadgerMaps",
	Long: `BadgerMaps CLI is a command line interface for interacting with the BadgerMaps API.
It allows you to push and pull data, run in server mode, and perform various utility operations.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output with additional details")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all non-essential output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug mode with maximum verbosity")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default is $HOME/.badgermaps.yaml)")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))

	// Add commands
	rootCmd.AddCommand(version.NewVersionCmd())
	rootCmd.AddCommand(push.NewPushCmd())
	rootCmd.AddCommand(pull.PullCmd())
	rootCmd.AddCommand(server.NewServerCmd())
	rootCmd.AddCommand(test.TestCmd())
	rootCmd.AddCommand(utils.NewUtilsCmd())
	rootCmd.AddCommand(auth.NewAuthCmd())
	rootCmd.AddCommand(search.NewSearchCmd())
	rootCmd.AddCommand(autocomplete.NewAutocompleteCmd(rootCmd))
	rootCmd.AddCommand(help.NewHelpCmd(rootCmd))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".badgermaps" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".badgermaps")

		// Also look for .env file in current directory
		viper.AddConfigPath(".")
		viper.SetConfigType("env")
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("BADGERMAPS")
	viper.AutomaticEnv()

	// Try to read the .env file first
	if _, err := os.Stat(".env"); err == nil {
		viper.SetConfigFile(".env")
		if err := viper.ReadInConfig(); err == nil {
			if verbose {
				fmt.Println("Using config file:", viper.ConfigFileUsed())
			}
		}
	} else if err := viper.ReadInConfig(); err == nil {
		// If .env not found, try the default config file
		if verbose {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}

	// Initialize colors and error utilities after viper is configured
	common.InitColors()
	common.InitErrorUtil()
}

func main() {
	Execute()
}
