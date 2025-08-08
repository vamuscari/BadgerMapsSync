package help

import (
	"fmt"
	"strings"

	"badgermapscli/common"

	"github.com/spf13/cobra"
)

// NewHelpCmd creates a new help command
func NewHelpCmd(rootCmd *cobra.Command) *cobra.Command {
	var verbose bool

	helpCmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Display help information",
		Long:  `Display help information for BadgerMaps CLI commands.`,
		Run: func(cmd *cobra.Command, args []string) {
			colors := common.Colors

			if len(args) == 0 {
				// Show general help
				fmt.Println(colors.Blue("BadgerMaps CLI Help"))
				fmt.Println(colors.Blue("================="))
				fmt.Println()
				fmt.Println("Usage:")
				fmt.Println("  badgermaps [command] [flags]")
				fmt.Println()
				fmt.Println("Available Commands:")

				// Get all commands from root command
				commands := rootCmd.Commands()

				// Calculate the longest command name for alignment
				maxLen := 0
				for _, c := range commands {
					if !c.Hidden && c.Name() != "help" {
						if len(c.Name()) > maxLen {
							maxLen = len(c.Name())
						}
					}
				}

				// Print commands
				for _, c := range commands {
					if !c.Hidden && c.Name() != "help" {
						padding := strings.Repeat(" ", maxLen-len(c.Name())+2)
						fmt.Printf("  %s%s%s\n", colors.Cyan(c.Name()), padding, c.Short)
					}
				}

				fmt.Println()
				fmt.Println("Global Flags:")
				fmt.Println("  -h, --help      Help for this command")
				fmt.Println("  -v, --verbose   Enable verbose output")
				fmt.Println("  -q, --quiet     Suppress all non-essential output")
				fmt.Println("      --debug     Enable debug mode with maximum verbosity")
				fmt.Println("      --no-color  Disable colored output")

				fmt.Println()
				fmt.Println("Use \"badgermaps help [command]\" for more information about a command.")
			} else {
				// Show help for specific command
				cmdName := args[0]

				// Find the command
				var targetCmd *cobra.Command
				for _, c := range rootCmd.Commands() {
					if c.Name() == cmdName {
						targetCmd = c
						break
					}
				}

				if targetCmd == nil {
					fmt.Printf("%s\n", colors.Red("Error: unknown command \"%s\"", cmdName))
					fmt.Println("Run \"badgermaps help\" for usage.")
					return
				}

				// Display command help
				fmt.Printf("%s\n", colors.Blue("Help for command: "+cmdName))
				fmt.Println(colors.Blue(strings.Repeat("=", 14+len(cmdName))))
				fmt.Println()

				fmt.Println("Usage:")
				fmt.Printf("  badgermaps %s [flags]\n", targetCmd.Use)

				if targetCmd.Long != "" {
					fmt.Println()
					fmt.Println("Description:")
					fmt.Println("  " + targetCmd.Long)
				}

				// Show examples if verbose
				if verbose && len(targetCmd.Example) > 0 {
					fmt.Println()
					fmt.Println("Examples:")
					fmt.Println(targetCmd.Example)
				}

				// Show flags
				if len(targetCmd.Flags().FlagUsages()) > 0 {
					fmt.Println()
					fmt.Println("Flags:")
					fmt.Print(targetCmd.Flags().FlagUsages())
				}

				// Show subcommands if any
				if len(targetCmd.Commands()) > 0 {
					fmt.Println()
					fmt.Println("Subcommands:")

					// Calculate the longest subcommand name for alignment
					maxLen := 0
					for _, c := range targetCmd.Commands() {
						if !c.Hidden {
							if len(c.Name()) > maxLen {
								maxLen = len(c.Name())
							}
						}
					}

					// Print subcommands
					for _, c := range targetCmd.Commands() {
						if !c.Hidden {
							padding := strings.Repeat(" ", maxLen-len(c.Name())+2)
							fmt.Printf("  %s%s%s\n", colors.Cyan(c.Name()), padding, c.Short)
						}
					}
				}
			}
		},
	}

	// Add flags
	helpCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show more detailed help information")

	return helpCmd
}
