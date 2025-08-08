package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"badgermapscli/common"
	"badgermapscli/database"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewUtilsCmd creates a new utils command
func NewUtilsCmd() *cobra.Command {
	utilsCmd := &cobra.Command{
		Use:   "utils",
		Short: "Utility commands for maintenance",
		Long:  `Utility commands for database maintenance, schema validation, and other operations.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a utility command")
			cmd.Help()
		},
	}

	// Add subcommands
	utilsCmd.AddCommand(newCreateTablesCmd())
	utilsCmd.AddCommand(newValidateSchemaCmd())
	utilsCmd.AddCommand(newDropTablesCmd())
	utilsCmd.AddCommand(newInitDatabaseCmd())
	utilsCmd.AddCommand(newCreateEnvCmd())
	utilsCmd.AddCommand(newInstallAutocompleteCmd())

	return utilsCmd
}

// newCreateTablesCmd creates a command to create database tables
func newCreateTablesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-tables",
		Short: "Create database tables",
		Long:  `Create all necessary database tables if they don't exist.`,
		Run: func(cmd *cobra.Command, args []string) {
			createTables()
		},
	}

	return cmd
}

// newValidateSchemaCmd creates a command to validate database schema
func newValidateSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-schema",
		Short: "Validate database schema",
		Long:  `Validate that all required database tables exist with the correct fields.`,
		Run: func(cmd *cobra.Command, args []string) {
			validateSchema()
		},
	}

	return cmd
}

// newDropTablesCmd creates a command to drop database tables
func newDropTablesCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "drop-tables",
		Short: "Drop database tables",
		Long:  `Drop all database tables. This will delete all data in the database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if !force {
				fmt.Println(color.YellowString("Warning: This will delete all data in the database."))
				fmt.Println("Use --force to confirm.")
				return
			}
			dropTables()
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force dropping tables without confirmation")

	return cmd
}

// newInitDatabaseCmd creates a command to initialize the database
func newInitDatabaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init-database",
		Short: "Initialize database",
		Long:  `Initialize the database with all necessary tables and indexes.`,
		Run: func(cmd *cobra.Command, args []string) {
			initDatabase()
		},
	}

	return cmd
}

// createTables creates all necessary database tables
func createTables() {
	fmt.Println(color.CyanString("Creating database tables..."))

	// Get database configuration from viper
	dbConfig := getDatabaseConfig()

	// Create database client
	client, err := database.NewClient(dbConfig, false)
	if err != nil {
		fmt.Println(color.RedString("Error creating database client: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Create tables
	err = client.InitializeSchema()
	if err != nil {
		fmt.Println(color.RedString("Error creating tables: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Database tables created successfully"))
}

// validateSchema validates the database schema
func validateSchema() {
	fmt.Println(color.CyanString("Validating database schema..."))

	// Get database configuration from viper
	dbConfig := getDatabaseConfig()

	// Create database client
	client, err := database.NewClient(dbConfig, false)
	if err != nil {
		fmt.Println(color.RedString("Error creating database client: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Validate schema
	err = client.ValidateDatabaseSchema()
	if err != nil {
		fmt.Println(color.RedString("Schema validation failed: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Database schema validation successful"))
}

// dropTables drops all database tables
func dropTables() {
	fmt.Println(color.CyanString("Dropping database tables..."))

	// Get database configuration from viper
	dbConfig := getDatabaseConfig()

	// Create database client
	client, err := database.NewClient(dbConfig, false)
	if err != nil {
		fmt.Println(color.RedString("Error creating database client: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Drop tables
	err = client.DropAllTables()
	if err != nil {
		fmt.Println(color.RedString("Error dropping tables: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Database tables dropped successfully"))
}

// initDatabase initializes the database
func initDatabase() {
	fmt.Println(color.CyanString("Initializing database..."))

	// Get database configuration from viper
	dbConfig := getDatabaseConfig()

	// Create database client
	client, err := database.NewClient(dbConfig, false)
	if err != nil {
		fmt.Println(color.RedString("Error creating database client: %v", err))
		os.Exit(1)
	}
	defer client.Close()

	// Initialize schema
	err = client.InitializeSchema()
	if err != nil {
		fmt.Println(color.RedString("Error initializing database: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Database initialized successfully"))
}

// newCreateEnvCmd creates a command to create a new .env file
func newCreateEnvCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "create-env",
		Short: "Create a new environment file",
		Long:  `Create a new .env file with default configuration values.`,
		Run: func(cmd *cobra.Command, args []string) {
			createEnvFile(force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force creation of .env file even if it already exists")

	return cmd
}

// createEnvFile creates a new .env file with default configuration values
func createEnvFile(force bool) {
	fmt.Println(color.CyanString("Creating .env file..."))

	err := common.CreateEnvFile(force)
	if err != nil {
		fmt.Println(color.RedString("Error creating .env file: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Environment file created successfully"))
	fmt.Println("You can now edit the .env file to customize your configuration")
}

// newInstallAutocompleteCmd creates a command to install shell autocompletion
func newInstallAutocompleteCmd() *cobra.Command {
	var (
		force     bool
		shellType string
	)

	cmd := &cobra.Command{
		Use:   "install-autocomplete",
		Short: "Install shell autocompletion",
		Long:  `Install shell autocompletion for bash, zsh, fish, or powershell.`,
		Run: func(cmd *cobra.Command, args []string) {
			installAutocomplete(force, shellType)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force installation without confirmation")
	cmd.Flags().StringVar(&shellType, "shell", "", "Shell type (bash, zsh, fish, powershell)")

	return cmd
}

// installAutocomplete installs the completion script for the specified shell
func installAutocomplete(force bool, shellType string) {
	// Determine shell type if not specified
	if shellType == "" {
		shellType = detectShell()
		if shellType == "" {
			fmt.Println(color.RedString("Error: could not detect shell type"))
			fmt.Println("Please specify shell type with --shell flag")
			os.Exit(1)
		}
	}

	// Get installation path
	installPath := getCompletionInstallPath(shellType)
	if installPath == "" {
		fmt.Println(color.RedString("Error: unsupported shell type: %s", shellType))
		fmt.Println("Supported shells: bash, zsh, fish, powershell")
		os.Exit(1)
	}

	// Create directory if it doesn't exist
	configDir := common.GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Println(color.RedString("Error creating config directory: %v", err))
		os.Exit(1)
	}

	// Path to store the completion script in the config directory
	scriptPath := filepath.Join(configDir, "autocomplete.sh")

	// Generate completion script
	fmt.Println(color.CyanString("Generating completion script..."))
	scriptContent := generateCompletionScript(shellType)

	// Write script to config directory
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	if err != nil {
		fmt.Println(color.RedString("Error writing completion script: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Completion script generated at %s", scriptPath))

	// Check if we need to modify shell config file
	shellConfigPath := getShellConfigPath(shellType)
	if shellConfigPath == "" {
		fmt.Println(color.YellowString("Could not determine shell config file location"))
		fmt.Println("Please manually add the following line to your shell config file:")
		fmt.Printf("source %s\n", scriptPath)
		return
	}

	// Check if the source line already exists in the shell config
	needToModify := true
	if _, err := os.Stat(shellConfigPath); err == nil {
		content, err := os.ReadFile(shellConfigPath)
		if err == nil {
			sourceLine := fmt.Sprintf("source %s", scriptPath)
			if strings.Contains(string(content), sourceLine) {
				needToModify = false
				fmt.Println(color.GreenString("Shell config already contains the source line"))
			}
		}
	}

	// Modify shell config if needed
	if needToModify {
		if !force {
			fmt.Printf("Add source line to %s? [y/N] ", shellConfigPath)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println(color.YellowString("Shell config not modified"))
				fmt.Println("Please manually add the following line to your shell config file:")
				fmt.Printf("source %s\n", scriptPath)
				return
			}
		}

		// Append source line to shell config
		f, err := os.OpenFile(shellConfigPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println(color.RedString("Error opening shell config file: %v", err))
			fmt.Println("Please manually add the following line to your shell config file:")
			fmt.Printf("source %s\n", scriptPath)
			return
		}
		defer f.Close()

		sourceLine := fmt.Sprintf("\n# BadgerMaps CLI autocompletion\nsource %s\n", scriptPath)
		if _, err := f.WriteString(sourceLine); err != nil {
			fmt.Println(color.RedString("Error writing to shell config file: %v", err))
			fmt.Println("Please manually add the following line to your shell config file:")
			fmt.Printf("source %s\n", scriptPath)
			return
		}

		fmt.Println(color.GreenString("Added source line to %s", shellConfigPath))
		fmt.Println("Please restart your shell or run the following command to enable autocompletion:")
		fmt.Printf("source %s\n", scriptPath)
	}
}

// detectShell detects the current shell
func detectShell() string {
	// Check SHELL environment variable
	shell := os.Getenv("SHELL")
	if shell != "" {
		// Extract the shell name from the path
		shell = filepath.Base(shell)
		switch shell {
		case "bash":
			return "bash"
		case "zsh":
			return "zsh"
		case "fish":
			return "fish"
		case "pwsh", "powershell":
			return "powershell"
		}
	}

	// Check if we're running in PowerShell
	if os.Getenv("PSModulePath") != "" {
		return "powershell"
	}

	// Default to bash on Unix and PowerShell on Windows
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "bash"
}

// getCompletionInstallPath returns the path where the completion script should be installed
func getCompletionInstallPath(shellType string) string {
	switch strings.ToLower(shellType) {
	case "bash":
		// For bash, use ~/.bash_completion or /etc/bash_completion.d/
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, ".bash_completion")
		}
		return "/etc/bash_completion.d/badgermaps"
	case "zsh":
		// For zsh, use ~/.zsh/completion/ or /usr/local/share/zsh/site-functions/
		home, err := os.UserHomeDir()
		if err == nil {
			zshCompletionDir := filepath.Join(home, ".zsh", "completion")
			if err := os.MkdirAll(zshCompletionDir, 0755); err == nil {
				return filepath.Join(zshCompletionDir, "_badgermaps")
			}
		}
		return "/usr/local/share/zsh/site-functions/_badgermaps"
	case "fish":
		// For fish, use ~/.config/fish/completions/
		home, err := os.UserHomeDir()
		if err == nil {
			fishCompletionDir := filepath.Join(home, ".config", "fish", "completions")
			if err := os.MkdirAll(fishCompletionDir, 0755); err == nil {
				return filepath.Join(fishCompletionDir, "badgermaps.fish")
			}
		}
		return "/usr/local/share/fish/completions/badgermaps.fish"
	case "powershell":
		// For PowerShell, use Documents\WindowsPowerShell\
		home, err := os.UserHomeDir()
		if err == nil {
			if runtime.GOOS == "windows" {
				return filepath.Join(home, "Documents", "WindowsPowerShell", "badgermaps.ps1")
			}
			return filepath.Join(home, ".config", "powershell", "badgermaps.ps1")
		}
	}
	return ""
}

// getShellConfigPath returns the path to the shell configuration file
func getShellConfigPath(shellType string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch strings.ToLower(shellType) {
	case "bash":
		// Check for .bashrc first, then .bash_profile
		bashrc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(bashrc); err == nil {
			return bashrc
		}
		return filepath.Join(home, ".bash_profile")
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	case "powershell":
		if runtime.GOOS == "windows" {
			return filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
		}
		return filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
	}
	return ""
}

// generateCompletionScript generates the completion script for the specified shell
func generateCompletionScript(shellType string) string {
	// Create a temporary file to write the completion script
	tempFile, err := os.CreateTemp("", "badgermaps-completion-")
	if err != nil {
		fmt.Println(color.RedString("Error creating temporary file: %v", err))
		os.Exit(1)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Create a temporary root command for generating completions
	rootCmd := &cobra.Command{Use: "badgermaps"}

	// Add some common subcommands to make the completion useful
	rootCmd.AddCommand(&cobra.Command{Use: "pull"})
	rootCmd.AddCommand(&cobra.Command{Use: "push"})
	rootCmd.AddCommand(&cobra.Command{Use: "setup"})
	rootCmd.AddCommand(&cobra.Command{Use: "utils"})
	rootCmd.AddCommand(&cobra.Command{Use: "version"})
	rootCmd.AddCommand(&cobra.Command{Use: "help"})
	rootCmd.AddCommand(&cobra.Command{Use: "search"})
	rootCmd.AddCommand(&cobra.Command{Use: "server"})
	rootCmd.AddCommand(&cobra.Command{Use: "test"})

	// Generate completion script based on shell type
	switch strings.ToLower(shellType) {
	case "bash":
		rootCmd.GenBashCompletion(tempFile)
	case "zsh":
		rootCmd.GenZshCompletion(tempFile)
	case "fish":
		rootCmd.GenFishCompletion(tempFile, true)
	case "powershell":
		rootCmd.GenPowerShellCompletion(tempFile)
	default:
		fmt.Println(color.RedString("Unsupported shell type: %s", shellType))
		fmt.Println("Supported shells: bash, zsh, fish, powershell")
		os.Exit(1)
	}

	// Read the generated script
	tempFile.Close()
	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		fmt.Println(color.RedString("Error reading completion script: %v", err))
		os.Exit(1)
	}

	return string(content)
}

// getDatabaseConfig gets the database configuration from viper
func getDatabaseConfig() *database.Config {
	dbType := viper.GetString("DATABASE_TYPE")
	if dbType == "" {
		dbType = "sqlite3" // Default to SQLite
	}

	dbConfig := &database.Config{
		DatabaseType: dbType,
		Host:         viper.GetString("DATABASE_HOST"),
		Port:         viper.GetString("DATABASE_PORT"),
		Database:     viper.GetString("DATABASE_NAME"),
		Username:     viper.GetString("DATABASE_USERNAME"),
		Password:     viper.GetString("DATABASE_PASSWORD"),
	}

	// Set default database name for SQLite if not provided
	if dbType == "sqlite3" && dbConfig.Database == "" {
		// Create default directory if it doesn't exist
		defaultDir := "./config/badgermaps"
		if err := os.MkdirAll(defaultDir, 0755); err != nil {
			fmt.Println(color.YellowString("Warning: Could not create default SQLite directory: %v", err))
			// Fall back to current directory if we can't create the default
			dbConfig.Database = "badgermaps.db"
		} else {
			dbConfig.Database = defaultDir + "/badgermaps.db"
		}
	}

	return dbConfig
}
