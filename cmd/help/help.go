package help

import (
	"fmt"
	"strings"

	"badgermapscli/app"

	"github.com/spf13/cobra"
)

// HelpCmd creates a new help command
func HelpCmd(rootCmd *cobra.Command, App *app.Application) *cobra.Command {
	var verbose bool

	helpCmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Display help information",
		Long:  `Display help information for BadgerMaps CLI commands.`,
		Run: func(cmd *cobra.Command, args []string) {
			// No debug output needed
			colors := app.Colors

			if len(args) == 0 {
				// Show general help
				fmt.Println(colors.Blue("BadgerMaps CLI Help - Custom Help Command"))
				fmt.Println(colors.Blue("==================================="))
				fmt.Println()
				fmt.Println(colors.Yellow("Usage:"))
				fmt.Println("  " + colors.Green("badgermaps") + " [command] [flags]")
				fmt.Println()
				fmt.Println(colors.Yellow("Available Commands:"))

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
				fmt.Println(colors.Yellow("Global Flags:"))
				fmt.Println("  " + colors.Cyan("-h, --help") + "      Help for this command")
				fmt.Println("  " + colors.Cyan("-v, --verbose") + "   Enable verbose output")
				fmt.Println("  " + colors.Cyan("-q, --quiet") + "     Suppress all non-essential output")
				fmt.Println("      " + colors.Cyan("--debug") + "     Enable debug mode with maximum verbosity")
				fmt.Println("      " + colors.Cyan("--no-color") + "  Disable colored output")

				fmt.Println()
				fmt.Println("Use " + colors.Green("\"badgermaps help [command]\"") + " for more information about a command.")
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
				fmt.Println(colors.Yellow("Usage:"))
				fmt.Printf("  %s %s [flags]\n", colors.Green("badgermaps"), colors.Cyan(targetCmd.Use))

				if targetCmd.Long != "" {
					fmt.Println()
					fmt.Println(colors.Yellow("Description:"))
					fmt.Println("  " + targetCmd.Long)
				}

				// Show examples if verbose
				if verbose && len(targetCmd.Example) > 0 {
					fmt.Println()
					fmt.Println(colors.Yellow("Examples:"))
					fmt.Println(colors.Green(targetCmd.Example))
				}

				// Show flags
				if len(targetCmd.Flags().FlagUsages()) > 0 {
					fmt.Println()
					fmt.Println(colors.Yellow("Flags:"))
					// We can't directly color the flag usages as they come pre-formatted
					// But we can print them as is
					fmt.Print(targetCmd.Flags().FlagUsages())
				}

				// Show subcommands if any
				if len(targetCmd.Commands()) > 0 {
					fmt.Println()
					fmt.Println(colors.Yellow("Subcommands:"))

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
