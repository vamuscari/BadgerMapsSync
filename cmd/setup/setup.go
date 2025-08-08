package setup

import (
	"badgermapscli/api"
	"badgermapscli/common"
	"badgermapscli/database"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewSetupCmd creates a new setup command
func NewSetupCmd() *cobra.Command {
	var (
		useEnv bool
	)

	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup BadgerMaps CLI configuration",
		Long:  `Setup BadgerMaps CLI configuration including API URL, API key, and database settings.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Check if configuration already exists
			configExists := false
			existingConfig := make(map[string]string)

			// Check for existing configuration based on storage type
			if useEnv {
				// Check if .env file exists
				if !common.CheckEnvFileExists() {
					fmt.Println(color.YellowString("No .env file found. Creating one with default values..."))
					if err := common.CreateEnvFile(false); err != nil {
						fmt.Println(color.RedString("Failed to create .env file: %v", err))
						os.Exit(1)
					}
				} else {
					configExists = true
					// Read existing values from viper (which reads from .env)
					existingConfig["API_URL"] = viper.GetString("API_URL")
					existingConfig["API_KEY"] = viper.GetString("API_KEY")
					existingConfig["DATABASE_TYPE"] = viper.GetString("DATABASE_TYPE")
					existingConfig["DATABASE_NAME"] = viper.GetString("DATABASE_NAME")
					existingConfig["DATABASE_HOST"] = viper.GetString("DATABASE_HOST")
					existingConfig["DATABASE_PORT"] = viper.GetString("DATABASE_PORT")
					existingConfig["DATABASE_USERNAME"] = viper.GetString("DATABASE_USERNAME")
					existingConfig["DATABASE_PASSWORD"] = viper.GetString("DATABASE_PASSWORD")
				}
			} else {
				// Check if config file exists
				configFile := common.GetConfigFilePath()
				if _, err := os.Stat(configFile); err == nil {
					configExists = true
					// Read existing values from viper (which reads from config.yaml)
					existingConfig["API_URL"] = viper.GetString("API_URL")
					existingConfig["API_KEY"] = viper.GetString("API_KEY")
					existingConfig["DATABASE_TYPE"] = viper.GetString("DATABASE_TYPE")
					existingConfig["DATABASE_NAME"] = viper.GetString("DATABASE_NAME")
					existingConfig["DATABASE_HOST"] = viper.GetString("DATABASE_HOST")
					existingConfig["DATABASE_PORT"] = viper.GetString("DATABASE_PORT")
					existingConfig["DATABASE_USERNAME"] = viper.GetString("DATABASE_USERNAME")
					existingConfig["DATABASE_PASSWORD"] = viper.GetString("DATABASE_PASSWORD")
				}
			}

			// Display existing configuration if it exists
			if configExists {
				fmt.Println(color.CyanString("Existing configuration found. You can update it or keep existing values by pressing Enter."))
				fmt.Println(color.CyanString("Current configuration:"))

				// Display API URL
				if existingConfig["API_URL"] != "" {
					fmt.Printf("  API URL: %s\n", existingConfig["API_URL"])
				} else {
					fmt.Println("  API URL: not set")
				}

				// Display API Key (masked)
				if existingConfig["API_KEY"] != "" {
					fmt.Println("  API Key: [exists]")
				} else {
					fmt.Println("  API Key: not set")
				}

				// Display Database Type
				if existingConfig["DATABASE_TYPE"] != "" {
					fmt.Printf("  Database Type: %s\n", existingConfig["DATABASE_TYPE"])
				} else {
					fmt.Println("  Database Type: not set")
				}

				// Display Database Name
				if existingConfig["DATABASE_NAME"] != "" {
					fmt.Printf("  Database Name: %s\n", existingConfig["DATABASE_NAME"])
				} else {
					fmt.Println("  Database Name: not set")
				}

				// Display Database Host (if applicable)
				if existingConfig["DATABASE_HOST"] != "" {
					fmt.Printf("  Database Host: %s\n", existingConfig["DATABASE_HOST"])
				}

				// Display Database Port (if applicable)
				if existingConfig["DATABASE_PORT"] != "" {
					fmt.Printf("  Database Port: %s\n", existingConfig["DATABASE_PORT"])
				}

				// Display Database Username (if applicable)
				if existingConfig["DATABASE_USERNAME"] != "" {
					fmt.Printf("  Database Username: %s\n", existingConfig["DATABASE_USERNAME"])
				}

				// Display Database Password (masked, if applicable)
				if existingConfig["DATABASE_PASSWORD"] != "" {
					fmt.Println("  Database Password: [exists]")
				}

				fmt.Println()
			}

			// Prompt for API URL with existing value as default if available
			defaultAPIURL := "https://badgerapis.badgermapping.com/api/2"
			if configExists && existingConfig["API_URL"] != "" {
				defaultAPIURL = existingConfig["API_URL"]
			}
			apiURL := promptWithDefault("Enter API URL", defaultAPIURL)

			// Prompt for API key
			apiKey := ""
			if configExists && existingConfig["API_KEY"] != "" {
				apiKey = promptWithDefault("Enter API key (leave empty to keep existing)", "")
				if apiKey == "" {
					apiKey = existingConfig["API_KEY"]
				}
			} else {
				apiKey = promptForAPIKey()
			}

			// Prompt for database type with existing value as default if available
			dbType := ""
			if configExists && existingConfig["DATABASE_TYPE"] != "" {
				dbType = promptForDatabaseTypeWithDefault(existingConfig["DATABASE_TYPE"])
			} else {
				dbType = promptForDatabaseType()
			}

			// Prompt for database-specific configurations
			dbConfig := promptForDatabaseConfigWithExisting(dbType, existingConfig)

			// Store the configuration
			if err := storeConfiguration(apiURL, apiKey, dbConfig, useEnv); err != nil {
				fmt.Println(color.RedString("Failed to store configuration: %v", err))
				os.Exit(1)
			}

			// Validate the API key
			if err := validateAPIKey(apiKey, apiURL); err != nil {
				fmt.Println(color.YellowString("Warning: API key validation failed: %v", err))
				fmt.Println("Configuration has been saved, but the API key may be invalid.")
			} else {
				fmt.Println(color.GreenString("API key validation successful"))
			}

			fmt.Println(color.GreenString("Setup completed successfully"))
		},
	}

	// Add flags
	setupCmd.Flags().BoolVar(&useEnv, "env", false, "Store configuration in .env file instead of the OS-specific config file")

	return setupCmd
}

// promptWithDefault prompts the user for input with a default value
func promptWithDefault(promptText, defaultValue string) string {
	fmt.Printf("%s [%s]: ", promptText, defaultValue)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

// promptForAPIKey prompts the user for their API key
func promptForAPIKey() string {
	fmt.Print("Enter your BadgerMaps API key: ")
	reader := bufio.NewReader(os.Stdin)
	apiKey, _ := reader.ReadString('\n')
	return strings.TrimSpace(apiKey)
}

// promptForDatabaseType prompts the user for their database type
func promptForDatabaseType() string {
	fmt.Println("Select database type:")
	fmt.Println("1. SQLite (default)")
	fmt.Println("2. PostgreSQL")
	fmt.Println("3. Microsoft SQL Server")

	fmt.Print("Enter your choice [1]: ")
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "2":
		return "postgres"
	case "3":
		return "mssql"
	default:
		return "sqlite3"
	}
}

// promptForDatabaseTypeWithDefault prompts the user for their database type with a default value
func promptForDatabaseTypeWithDefault(defaultType string) string {
	// Determine the default choice number based on the default type
	defaultChoice := "1" // sqlite3
	if defaultType == "postgres" {
		defaultChoice = "2"
	} else if defaultType == "mssql" {
		defaultChoice = "3"
	}

	fmt.Println("Select database type:")
	fmt.Println("1. SQLite")
	fmt.Println("2. PostgreSQL")
	fmt.Println("3. Microsoft SQL Server")

	fmt.Printf("Enter your choice [%s]: ", defaultChoice)
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "" {
		choice = defaultChoice
	}

	switch choice {
	case "2":
		return "postgres"
	case "3":
		return "mssql"
	default:
		return "sqlite3"
	}
}

// promptForDatabaseConfig prompts for database-specific configurations
func promptForDatabaseConfig(dbType string) *database.Config {
	dbConfig := &database.Config{
		DatabaseType: dbType,
	}

	switch dbType {
	case "sqlite3":
		// For SQLite, just prompt for the database file path
		defaultDBPath := "badgermaps.db"
		dbPath := promptWithDefault("Enter database file path", defaultDBPath)
		dbConfig.Database = dbPath

	case "postgres":
		// For PostgreSQL, prompt for host, port, database name, username, and password
		dbConfig.Host = promptWithDefault("Enter database host", "localhost")
		dbConfig.Port = promptWithDefault("Enter database port", "5432")
		dbConfig.Database = promptWithDefault("Enter database name", "badgermaps")
		dbConfig.Username = promptWithDefault("Enter database username", "postgres")
		dbConfig.Password = promptForPassword("Enter database password")

	case "mssql":
		// For MSSQL, prompt for host, port, database name, username, and password
		dbConfig.Host = promptWithDefault("Enter database host", "localhost")
		dbConfig.Port = promptWithDefault("Enter database port", "1433")
		dbConfig.Database = promptWithDefault("Enter database name", "badgermaps")
		dbConfig.Username = promptWithDefault("Enter database username", "sa")
		dbConfig.Password = promptForPassword("Enter database password")
	}

	return dbConfig
}

// promptForDatabaseConfigWithExisting prompts for database-specific configurations with existing values
func promptForDatabaseConfigWithExisting(dbType string, existingConfig map[string]string) *database.Config {
	dbConfig := &database.Config{
		DatabaseType: dbType,
	}

	switch dbType {
	case "sqlite3":
		// For SQLite, just prompt for the database file path
		defaultDBPath := "badgermaps.db"
		if existingConfig["DATABASE_NAME"] != "" {
			defaultDBPath = existingConfig["DATABASE_NAME"]
		}
		dbPath := promptWithDefault("Enter database file path", defaultDBPath)
		dbConfig.Database = dbPath

	case "postgres":
		// For PostgreSQL, prompt for host, port, database name, username, and password
		defaultHost := "localhost"
		if existingConfig["DATABASE_HOST"] != "" {
			defaultHost = existingConfig["DATABASE_HOST"]
		}
		dbConfig.Host = promptWithDefault("Enter database host", defaultHost)

		defaultPort := "5432"
		if existingConfig["DATABASE_PORT"] != "" {
			defaultPort = existingConfig["DATABASE_PORT"]
		}
		dbConfig.Port = promptWithDefault("Enter database port", defaultPort)

		defaultDBName := "badgermaps"
		if existingConfig["DATABASE_NAME"] != "" {
			defaultDBName = existingConfig["DATABASE_NAME"]
		}
		dbConfig.Database = promptWithDefault("Enter database name", defaultDBName)

		defaultUsername := "postgres"
		if existingConfig["DATABASE_USERNAME"] != "" {
			defaultUsername = existingConfig["DATABASE_USERNAME"]
		}
		dbConfig.Username = promptWithDefault("Enter database username", defaultUsername)

		// For password, check if one exists already
		if existingConfig["DATABASE_PASSWORD"] != "" {
			password := promptWithDefault("Enter database password (leave empty to keep existing)", "")
			if password == "" {
				dbConfig.Password = existingConfig["DATABASE_PASSWORD"]
			} else {
				dbConfig.Password = password
			}
		} else {
			dbConfig.Password = promptForPassword("Enter database password")
		}

	case "mssql":
		// For MSSQL, prompt for host, port, database name, username, and password
		defaultHost := "localhost"
		if existingConfig["DATABASE_HOST"] != "" {
			defaultHost = existingConfig["DATABASE_HOST"]
		}
		dbConfig.Host = promptWithDefault("Enter database host", defaultHost)

		defaultPort := "1433"
		if existingConfig["DATABASE_PORT"] != "" {
			defaultPort = existingConfig["DATABASE_PORT"]
		}
		dbConfig.Port = promptWithDefault("Enter database port", defaultPort)

		defaultDBName := "badgermaps"
		if existingConfig["DATABASE_NAME"] != "" {
			defaultDBName = existingConfig["DATABASE_NAME"]
		}
		dbConfig.Database = promptWithDefault("Enter database name", defaultDBName)

		defaultUsername := "sa"
		if existingConfig["DATABASE_USERNAME"] != "" {
			defaultUsername = existingConfig["DATABASE_USERNAME"]
		}
		dbConfig.Username = promptWithDefault("Enter database username", defaultUsername)

		// For password, check if one exists already
		if existingConfig["DATABASE_PASSWORD"] != "" {
			password := promptWithDefault("Enter database password (leave empty to keep existing)", "")
			if password == "" {
				dbConfig.Password = existingConfig["DATABASE_PASSWORD"]
			} else {
				dbConfig.Password = password
			}
		} else {
			dbConfig.Password = promptForPassword("Enter database password")
		}
	}

	return dbConfig
}

// promptForPassword prompts for a password without echoing
func promptForPassword(promptText string) string {
	fmt.Print(promptText + ": ")
	reader := bufio.NewReader(os.Stdin)
	password, _ := reader.ReadString('\n')
	return strings.TrimSpace(password)
}

// validateAPIKey validates the API key by making a test request
func validateAPIKey(apiKey, apiURL string) error {
	// Create API client with custom URL
	apiClient := api.NewAPIClientWithURL(apiKey, apiURL)

	// Test API connection
	return apiClient.TestAPIConnection()
}

// storeConfiguration stores the configuration
func storeConfiguration(apiURL, apiKey string, dbConfig *database.Config, useEnv bool) error {
	// Set in viper
	viper.Set("API_KEY", apiKey)
	viper.Set("API_URL", apiURL)
	viper.Set("DATABASE_TYPE", dbConfig.DatabaseType)
	viper.Set("DATABASE_NAME", dbConfig.Database)
	viper.Set("DATABASE_HOST", dbConfig.Host)
	viper.Set("DATABASE_PORT", dbConfig.Port)
	viper.Set("DATABASE_USERNAME", dbConfig.Username)
	viper.Set("DATABASE_PASSWORD", dbConfig.Password)

	// Always store in config.yaml file unless useEnv is true
	if !useEnv {
		err := storeInConfigFile(apiURL, apiKey, dbConfig)
		if err != nil {
			return err
		}
		fmt.Println(color.GreenString("Configuration stored in config file"))
	} else {
		// Store in .env file
		err := storeInEnvFile(apiURL, apiKey, dbConfig)
		if err != nil {
			return err
		}
		fmt.Println(color.GreenString("Configuration stored in environment file"))
	}

	return nil
}

// storeInConfigFile stores the configuration in the config.yaml file
func storeInConfigFile(apiURL, apiKey string, dbConfig *database.Config) error {
	// Ensure config directory exists
	if err := common.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Get config file path
	configFile := common.GetConfigFilePath()

	// Create a map for the config
	config := map[string]string{
		"API_KEY":           apiKey,
		"API_URL":           apiURL,
		"DATABASE_TYPE":     dbConfig.DatabaseType,
		"DATABASE_NAME":     dbConfig.Database,
		"DATABASE_HOST":     dbConfig.Host,
		"DATABASE_PORT":     dbConfig.Port,
		"DATABASE_USERNAME": dbConfig.Username,
		"DATABASE_PASSWORD": dbConfig.Password,
	}

	// Convert to YAML format
	yamlData := "# BadgerMaps CLI Configuration\n"
	for key, value := range config {
		if value != "" {
			yamlData += fmt.Sprintf("%s: %s\n", key, value)
		}
	}

	// Write to file
	if err := os.WriteFile(configFile, []byte(yamlData), 0644); err != nil {
		return fmt.Errorf("failed to write to config file: %w", err)
	}

	fmt.Println("Configuration stored in config file:", configFile)
	return nil
}

// storeInEnvFile stores the configuration in the .env file
func storeInEnvFile(apiURL, apiKey string, dbConfig *database.Config) error {
	// Check if .env file exists
	if !common.CheckEnvFileExists() {
		return fmt.Errorf("no .env file found. Use 'badgermaps utils create-env' to create one")
	}

	// Read the current .env file
	envContent, err := os.ReadFile(".env")
	if err != nil {
		return fmt.Errorf("failed to read .env file: %w", err)
	}

	// Parse the current .env file
	lines := strings.Split(string(envContent), "\n")
	envMap := make(map[string]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			envMap[key] = parts[1]
		}
	}

	// Update with new values
	envMap["API_KEY"] = apiKey
	envMap["API_URL"] = apiURL
	envMap["DATABASE_TYPE"] = dbConfig.DatabaseType
	envMap["DATABASE_NAME"] = dbConfig.Database
	envMap["DATABASE_HOST"] = dbConfig.Host
	envMap["DATABASE_PORT"] = dbConfig.Port
	envMap["DATABASE_USERNAME"] = dbConfig.Username
	envMap["DATABASE_PASSWORD"] = dbConfig.Password

	// Rebuild the .env file content
	newContent := "# BadgerMaps CLI Configuration\n\n"
	newContent += "# API Configuration\n"
	newContent += fmt.Sprintf("API_KEY=%s\n", envMap["API_KEY"])
	newContent += fmt.Sprintf("API_URL=%s\n", envMap["API_URL"])
	newContent += "\n# Database Configuration\n"
	newContent += fmt.Sprintf("DATABASE_TYPE=%s\n", envMap["DATABASE_TYPE"])
	newContent += fmt.Sprintf("DATABASE_NAME=%s\n", envMap["DATABASE_NAME"])
	newContent += fmt.Sprintf("DATABASE_HOST=%s\n", envMap["DATABASE_HOST"])
	newContent += fmt.Sprintf("DATABASE_PORT=%s\n", envMap["DATABASE_PORT"])
	newContent += fmt.Sprintf("DATABASE_USERNAME=%s\n", envMap["DATABASE_USERNAME"])
	newContent += fmt.Sprintf("DATABASE_PASSWORD=%s\n", envMap["DATABASE_PASSWORD"])

	// Preserve other settings
	newContent += "\n# Other Configuration\n"
	for key, value := range envMap {
		if key != "API_KEY" && key != "API_URL" &&
			key != "DATABASE_TYPE" && key != "DATABASE_NAME" &&
			key != "DATABASE_HOST" && key != "DATABASE_PORT" &&
			key != "DATABASE_USERNAME" && key != "DATABASE_PASSWORD" {
			newContent += fmt.Sprintf("%s=%s\n", key, value)
		}
	}

	// Write the updated content back to the .env file
	if err := os.WriteFile(".env", []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write to .env file: %w", err)
	}

	return nil
}
