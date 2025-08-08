package version

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Version information - these will be set during build
	Version = "0.1.0"
	Commit  = "development"
	Date    = "unknown"
)

// NewVersionCmd creates a new version command
func NewVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Long:  `Display detailed version information about the BadgerMaps CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			displayVersion()
		},
	}

	return versionCmd
}

// displayVersion shows the version information
func displayVersion() {
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("%s %s\n", bold("BadgerMaps CLI Version:"), cyan(Version))
	fmt.Printf("%s %s\n", bold("Commit:"), Commit)
	fmt.Printf("%s %s\n", bold("Build Date:"), Date)
}
