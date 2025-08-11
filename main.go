package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"badgermapscli/cmd/help"
	"badgermapscli/cmd/pull"
	"badgermapscli/cmd/push"
	"badgermapscli/cmd/search"
	"badgermapscli/cmd/server"
	"badgermapscli/cmd/setup"
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
	cobra.EnableCommandSorting = false

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

	viper.G

	// Create commands
	pushCmd := push.NewPushCmd()
	pullCmd := pull.PullCmd()
	searchCmd := search.NewSearchCmd()
	serverCmd := server.NewServerCmd()
	testCmd := test.TestCmd()
	utilsCmd := utils.NewUtilsCmd()
	setupCmd := setup.NewSetupCmd()
	helpCmd := help.NewHelpCmd(rootCmd)
	versionCmd := version.NewVersionCmd()

	// Apply setup verification to commands that need it
	wrapCommandWithSetupCheck(pushCmd)
	wrapCommandWithSetupCheck(pullCmd)
	wrapCommandWithSetupCheck(serverCmd)
	wrapCommandWithSetupCheck(testCmd)
	wrapCommandWithSetupCheck(searchCmd)
	wrapCommandWithSetupCheck(helpCmd)

	// Add commands to root
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(utilsCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(helpCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Use OS-specific config directory
		configDir := common.GetConfigDir()
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		// Also search in ~/.badgermaps/ for backward compatibility (Unix systems)
		if runtime.GOOS != "windows" {
			home, err := os.UserHomeDir()
			if err == nil {
				viper.AddConfigPath(filepath.Join(home, ".badgermaps"))
			}
		}

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
	} else {
		// If .env not found, try the default config files
		if err := viper.ReadInConfig(); err == nil {
			if verbose {
				fmt.Println("Using config file:", viper.ConfigFileUsed())
			}
		} else if verbose {
			fmt.Println("No config file found, using defaults and environment variables")
		}
	}

	// Initialize colors and error utilities after viper is configured
	common.InitColors()
	common.InitErrorUtil()
}

// wrapCommandWithSetupCheck adds setup verification to a command and all its subcommands
// It checks if the command is one that requires setup (not setup, version, or utils)
// If setup is required but not completed, it prompts the user to run setup
func wrapCommandWithSetupCheck(cmd *cobra.Command) {
	// First wrap the command itself
	wrapSingleCommandWithSetupCheck(cmd)

	// Then recursively wrap all subcommands
	for _, subCmd := range cmd.Commands() {
		wrapCommandWithSetupCheck(subCmd)
	}
}

// wrapSingleCommandWithSetupCheck adds setup verification to a single command
func wrapSingleCommandWithSetupCheck(cmd *cobra.Command) {
	originalRun := cmd.Run
	originalRunE := cmd.RunE

	cmdName := cmd.Name()
	// Skip setup verification for these commands
	if cmdName == "setup" || cmdName == "version" || strings.HasPrefix(cmdName, "utils") {
		return
	}

	if originalRun != nil {
		cmd.Run = func(cmd *cobra.Command, args []string) {
			common.VerifySetupOrExit()
			originalRun(cmd, args)
		}
	}

	if originalRunE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			common.VerifySetupOrExit()
			return originalRunE(cmd, args)
		}
	}
}

func main() {
	Execute()
}
