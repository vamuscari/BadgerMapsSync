package autocomplete

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"badgermapscli/common"

	"github.com/spf13/cobra"
)

// NewAutocompleteCmd creates a new autocomplete command
func NewAutocompleteCmd(rootCmd *cobra.Command) *cobra.Command {
	var force bool
	var shell string

	autocompleteCmd := &cobra.Command{
		Use:   "autocomplete",
		Short: "Generate shell autocompletion scripts",
		Long: `Generate shell autocompletion scripts for BadgerMaps CLI.
Supported shells: bash, zsh, fish, powershell.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If no arguments, show help
			if len(args) == 0 {
				cmd.Help()
				return
			}

			// Handle install command
			if args[0] == "install" {
				installAutocomplete(rootCmd, force, shell)
				return
			}

			// Handle uninstall command
			if args[0] == "uninstall" {
				uninstallAutocomplete(force, shell)
				return
			}

			// Generate completion script for the specified shell
			generateCompletionScript(rootCmd, args[0])
		},
	}

	// Add flags
	autocompleteCmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	autocompleteCmd.Flags().StringVar(&shell, "shell", "", "Specify shell type (bash, zsh, fish, powershell)")

	// Add subcommands
	autocompleteCmd.AddCommand(newBashCmd(rootCmd))
	autocompleteCmd.AddCommand(newZshCmd(rootCmd))
	autocompleteCmd.AddCommand(newFishCmd(rootCmd))
	autocompleteCmd.AddCommand(newPowerShellCmd(rootCmd))
	autocompleteCmd.AddCommand(newInstallCmd(rootCmd))
	autocompleteCmd.AddCommand(newUninstallCmd())

	return autocompleteCmd
}

// newBashCmd creates a command to generate bash completion script
func newBashCmd(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Long:  `Generate bash completion script for BadgerMaps CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenBashCompletion(os.Stdout)
		},
	}
}

// newZshCmd creates a command to generate zsh completion script
func newZshCmd(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long:  `Generate zsh completion script for BadgerMaps CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenZshCompletion(os.Stdout)
		},
	}
}

// newFishCmd creates a command to generate fish completion script
func newFishCmd(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long:  `Generate fish completion script for BadgerMaps CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenFishCompletion(os.Stdout, true)
		},
	}
}

// newPowerShellCmd creates a command to generate PowerShell completion script
func newPowerShellCmd(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell completion script",
		Long:  `Generate PowerShell completion script for BadgerMaps CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenPowerShellCompletion(os.Stdout)
		},
	}
}

// newInstallCmd creates a command to install completion script
func newInstallCmd(rootCmd *cobra.Command) *cobra.Command {
	var force bool
	var shell string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install shell completion script",
		Long:  `Install shell completion script for BadgerMaps CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			installAutocomplete(rootCmd, force, shell)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	cmd.Flags().StringVar(&shell, "shell", "", "Specify shell type (bash, zsh, fish, powershell)")

	return cmd
}

// newUninstallCmd creates a command to uninstall completion script
func newUninstallCmd() *cobra.Command {
	var force bool
	var shell string

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall shell completion script",
		Long:  `Uninstall shell completion script for BadgerMaps CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			uninstallAutocomplete(force, shell)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	cmd.Flags().StringVar(&shell, "shell", "", "Specify shell type (bash, zsh, fish, powershell)")

	return cmd
}

// generateCompletionScript generates a completion script for the specified shell
func generateCompletionScript(rootCmd *cobra.Command, shellType string) {
	colors := common.Colors

	switch strings.ToLower(shellType) {
	case "bash":
		rootCmd.GenBashCompletion(os.Stdout)
	case "zsh":
		rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		rootCmd.GenFishCompletion(os.Stdout, true)
	case "powershell":
		rootCmd.GenPowerShellCompletion(os.Stdout)
	default:
		fmt.Println(colors.Red("Error: unsupported shell type: %s", shellType))
		fmt.Println("Supported shells: bash, zsh, fish, powershell")
		os.Exit(1)
	}
}

// installAutocomplete installs the completion script for the current shell
func installAutocomplete(rootCmd *cobra.Command, force bool, shellType string) {
	colors := common.Colors

	// Determine shell type if not specified
	if shellType == "" {
		shellType = detectShell()
		if shellType == "" {
			fmt.Println(colors.Red("Error: could not detect shell type"))
			fmt.Println("Please specify shell type with --shell flag")
			os.Exit(1)
		}
	}

	// Get installation path
	installPath := getCompletionInstallPath(shellType)
	if installPath == "" {
		fmt.Println(colors.Red("Error: unsupported shell type: %s", shellType))
		fmt.Println("Supported shells: bash, zsh, fish, powershell")
		os.Exit(1)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(installPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if !force {
			fmt.Printf("Directory %s does not exist. Create it? [y/N] ", dir)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println(colors.Yellow("Installation cancelled"))
				return
			}
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Println(colors.Red("Error creating directory: %v", err))
			os.Exit(1)
		}
	}

	// Confirm installation
	if !force {
		fmt.Printf("Install completion script to %s? [y/N] ", installPath)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println(colors.Yellow("Installation cancelled"))
			return
		}
	}

	// Create temporary file for completion script
	tempFile, err := os.CreateTemp("", "badgermaps-completion-")
	if err != nil {
		fmt.Println(colors.Red("Error creating temporary file: %v", err))
		os.Exit(1)
	}
	defer os.Remove(tempFile.Name())

	// Generate completion script
	switch strings.ToLower(shellType) {
	case "bash":
		rootCmd.GenBashCompletion(tempFile)
	case "zsh":
		rootCmd.GenZshCompletion(tempFile)
	case "fish":
		rootCmd.GenFishCompletion(tempFile, true)
	case "powershell":
		rootCmd.GenPowerShellCompletion(tempFile)
	}

	tempFile.Close()

	// Copy temporary file to installation path
	err = copyFile(tempFile.Name(), installPath)
	if err != nil {
		fmt.Println(colors.Red("Error installing completion script: %v", err))
		os.Exit(1)
	}

	fmt.Println(colors.Green("Completion script installed successfully to %s", installPath))

	// Print instructions
	printShellInstructions(shellType, installPath)
}

// uninstallAutocomplete uninstalls the completion script
func uninstallAutocomplete(force bool, shellType string) {
	colors := common.Colors

	// Determine shell type if not specified
	if shellType == "" {
		shellType = detectShell()
		if shellType == "" {
			fmt.Println(colors.Red("Error: could not detect shell type"))
			fmt.Println("Please specify shell type with --shell flag")
			os.Exit(1)
		}
	}

	// Get installation path
	installPath := getCompletionInstallPath(shellType)
	if installPath == "" {
		fmt.Println(colors.Red("Error: unsupported shell type: %s", shellType))
		fmt.Println("Supported shells: bash, zsh, fish, powershell")
		os.Exit(1)
	}

	// Check if file exists
	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		fmt.Println(colors.Yellow("Completion script not found at %s", installPath))
		return
	}

	// Confirm uninstallation
	if !force {
		fmt.Printf("Uninstall completion script from %s? [y/N] ", installPath)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println(colors.Yellow("Uninstallation cancelled"))
			return
		}
	}

	// Remove file
	err := os.Remove(installPath)
	if err != nil {
		fmt.Println(colors.Red("Error removing completion script: %v", err))
		os.Exit(1)
	}

	fmt.Println(colors.Green("Completion script uninstalled successfully from %s", installPath))
}

// detectShell attempts to detect the current shell
func detectShell() string {
	// Check SHELL environment variable
	shell := os.Getenv("SHELL")
	if shell != "" {
		if strings.Contains(shell, "bash") {
			return "bash"
		} else if strings.Contains(shell, "zsh") {
			return "zsh"
		} else if strings.Contains(shell, "fish") {
			return "fish"
		}
	}

	// Check if we're in PowerShell
	if os.Getenv("PSModulePath") != "" {
		return "powershell"
	}

	// Default to bash on Unix-like systems, powershell on Windows
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "bash"
}

// getCompletionInstallPath returns the installation path for the completion script
func getCompletionInstallPath(shellType string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch strings.ToLower(shellType) {
	case "bash":
		// Try to find bash completion directory
		for _, dir := range []string{
			"/etc/bash_completion.d",
			"/usr/local/etc/bash_completion.d",
			filepath.Join(home, ".local/share/bash-completion/completions"),
		} {
			if _, err := os.Stat(dir); err == nil {
				return filepath.Join(dir, "badgermaps")
			}
		}
		// Fallback to home directory
		return filepath.Join(home, ".bash_completion.d", "badgermaps")
	case "zsh":
		// Try to find zsh completion directory
		for _, dir := range []string{
			filepath.Join(home, ".zsh/completion"),
			filepath.Join(home, ".zsh/completions"),
			filepath.Join(home, ".oh-my-zsh/completions"),
		} {
			if _, err := os.Stat(dir); err == nil {
				return filepath.Join(dir, "_badgermaps")
			}
		}
		// Fallback to home directory
		return filepath.Join(home, ".zsh/completion", "_badgermaps")
	case "fish":
		return filepath.Join(home, ".config/fish/completions", "badgermaps.fish")
	case "powershell":
		// Get PowerShell modules directory
		out, err := exec.Command("powershell", "-Command", "echo $env:PSModulePath").Output()
		if err == nil {
			paths := strings.Split(strings.TrimSpace(string(out)), string(os.PathListSeparator))
			if len(paths) > 0 {
				return filepath.Join(paths[0], "badgermaps.ps1")
			}
		}
		// Fallback to documents directory
		return filepath.Join(home, "Documents", "WindowsPowerShell", "badgermaps.ps1")
	}

	return ""
}

// printShellInstructions prints instructions for using the completion script
func printShellInstructions(shellType, installPath string) {
	colors := common.Colors
	fmt.Println()
	fmt.Println(colors.Blue("To enable completions in your current shell session:"))

	switch strings.ToLower(shellType) {
	case "bash":
		fmt.Println("  source " + installPath)
		fmt.Println()
		fmt.Println(colors.Blue("To enable completions for all sessions:"))
		fmt.Println("  Add the following line to your ~/.bashrc file:")
		fmt.Println("  source " + installPath)
	case "zsh":
		fmt.Println("  source " + installPath)
		fmt.Println()
		fmt.Println(colors.Blue("To enable completions for all sessions:"))
		fmt.Println("  Add the following line to your ~/.zshrc file:")
		fmt.Println("  source " + installPath)
	case "fish":
		fmt.Println("  source " + installPath)
		fmt.Println()
		fmt.Println(colors.Blue("Fish completions are automatically loaded from:"))
		fmt.Println("  " + installPath)
	case "powershell":
		fmt.Println("  . " + installPath)
		fmt.Println()
		fmt.Println(colors.Blue("To enable completions for all sessions:"))
		fmt.Println("  Add the following line to your PowerShell profile:")
		fmt.Println("  . " + installPath)
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return nil
}
