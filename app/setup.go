package app

import (
	"badgermaps/api"
	"badgermaps/database"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// InteractiveSetup guides the user through setting up the configuration
func InteractiveSetup(a *App) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(utils.Colors.Blue("=== BadgerMaps CLI Setup ==="))
	fmt.Println(utils.Colors.Yellow("This will guide you through setting up the BadgerMaps CLI."))
	fmt.Println()

	setupOptions := []string{"Quick Setup", "Advanced Setup"}
	setupChoice := utils.PromptChoice(reader, "Select setup type", setupOptions)

	if setupChoice == "Advanced Setup" {
		userSpecifiedFile := (a.State.EnvFile != nil) || (a.State.ConfigFile != nil)

		if !userSpecifiedFile {
			configOptions := []string{"config.yaml (in user config directory)", ".env file (in executable's directory)"}
			configChoice := utils.PromptChoice(reader, "Select configuration file type to save to", configOptions)
			if configChoice == configOptions[1] {
				exe, err := os.Executable()
				if err != nil {
					fmt.Println(utils.Colors.Red("Error getting executable path: %v", err))
					return false
				}
				a.LoadedConfigFile = filepath.Join(filepath.Dir(exe), ".env")
			} else {
				a.LoadedConfigFile = utils.GetConfigDirFile("config.yaml")
			}
		}
	}

	// API Settings
	fmt.Println(utils.Colors.Blue("--- API Settings ---"))
	apiKey := utils.PromptString(reader, "API Key", viper.GetString("API_KEY"))
	viper.Set("API_KEY", apiKey)
	apiURL := utils.PromptString(reader, "API URL", api.DefaultApiBaseURL)
	viper.Set("API_URL", apiURL)

	// Database Settings
	fmt.Println(utils.Colors.Blue("--- Database Settings ---"))

	currentDB, err := database.LoadDatabaseSettings(a.State)
	if err != nil {
		fmt.Println(utils.Colors.Yellow("Could not load current database configuration: %v", err))
	}

	dbOptions := []string{"sqlite3", "postgres", "mssql"}
	if currentDB != nil {
		for i, dbType := range dbOptions {
			if dbType == currentDB.GetType() {
				dbOptions = append(dbOptions[:i], dbOptions[i+1:]...)
				dbOptions = append([]string{currentDB.GetType()}, dbOptions...)
				break
			}
		}
	}
	dbType := utils.PromptChoice(reader, "Select database type", dbOptions)
	viper.Set("DB_TYPE", dbType)

	a.DB.PromptDatabaseSettings()
	if err := a.DB.SaveConfig(); err != nil {
		fmt.Println(utils.Colors.Red("Error saving database settings: %v", err))
		return false
	}

	if err != nil {
		fmt.Println(utils.Colors.Red("Error reloading database with new settings: %v", err))
		return false
	}

	// Test the new connection before proceeding
	fmt.Println()
	fmt.Println(utils.Colors.Cyan("Testing database connection..."))
	if err := a.DB.TestConnection(); err != nil {
		fmt.Println(utils.Colors.Red("Database connection failed: %v", err))
		fmt.Println(utils.Colors.Yellow("Please check your database settings and try again."))
		return false
	}
	fmt.Println(utils.Colors.Green("Database connection successful."))

	if setupChoice == "Advanced Setup" {
		// Server Settings
		fmt.Println(utils.Colors.Blue("--- Server Settings ---"))
		serverHost := utils.PromptString(reader, "Server Host", viper.GetString("SERVER_HOST"))
		viper.Set("SERVER_HOST", serverHost)
		serverPort := utils.PromptInt(reader, "Server Port", viper.GetInt("SERVER_PORT"))
		viper.Set("SERVER_PORT", serverPort)
		webhookSecret := utils.PromptString(reader, "Webhook Secret", viper.GetString("WEBHOOK_SECRET"))
		viper.Set("WEBHOOK_SECRET", webhookSecret)

		// Advanced Settings
		fmt.Println(utils.Colors.Blue("--- Advanced Settings ---"))
		maxConcurrent := utils.PromptInt(reader, "Max Concurrent Requests", viper.GetInt("MAX_CONCURRENT_REQUESTS"))
		viper.Set("MAX_CONCURRENT_REQUESTS", maxConcurrent)
	}

	// Save configuration
	configFile := a.LoadedConfigFile
	if configFile == "" {
		configFile = utils.GetConfigDirFile("config.yaml")
	}

	configType := "yaml"
	if filepath.Ext(configFile) == ".env" || filepath.Base(configFile) == ".env" {
		configType = "env"
	}
	viper.SetConfigType(configType)

	if err := viper.WriteConfigAs(configFile); err != nil {
		fmt.Println(utils.Colors.Red("Error saving config file: %v", err))
		return false
	}

	fmt.Println()
	fmt.Println(utils.Colors.Green("✓ Setup completed successfully!"))
	fmt.Println(utils.Colors.Green("Configuration saved to: %s", configFile))

	// Check if the database schema is valid
	if err := a.DB.ValidateSchema(); err == nil {
		fmt.Println(utils.Colors.Yellow("Database schema already exists and is valid."))
		reinitialize := utils.PromptBool(reader, "Do you want to reinitialize the database? (This will delete all existing data)", false)
		if reinitialize {
			fmt.Println(utils.Colors.Yellow("Reinitializing database..."))
			if err := a.DB.DropAllTables(); err != nil {
				fmt.Println(utils.Colors.Red("Error dropping tables: %v", err))
				return false
			}
			if err := a.DB.EnforceSchema(); err != nil {
				fmt.Println(utils.Colors.Red("Error enforcing schema: %v", err))
				return false
			}
			fmt.Println(utils.Colors.Green("Database reinitialized successfully."))
		}
	} else {
		// Schema is invalid or does not exist
		fmt.Println(utils.Colors.Yellow("Database schema is invalid or missing."))
		enforce := utils.PromptBool(reader, "Do you want to create/update the database schema now?", true)
		if enforce {
			fmt.Println(utils.Colors.Yellow("Enforcing schema..."))
			if err := a.DB.EnforceSchema(); err != nil {
				fmt.Println(utils.Colors.Red("Error enforcing schema: %v", err))
				return false
			}
			fmt.Println(utils.Colors.Green("Database schema created/updated successfully."))
		}
	}

	return true
}
