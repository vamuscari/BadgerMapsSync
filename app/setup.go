package app

import (
	"badgermaps/database"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// InteractiveSetup guides the user through setting up the configuration
func InteractiveSetup(App *App) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(utils.Colors.Blue("=== BadgerMaps CLI Setup ==="))
	fmt.Println(utils.Colors.Yellow("This will guide you through setting up the BadgerMaps CLI."))
	fmt.Println()

	setupOptions := []string{"Quick Setup", "Advanced Setup"}
	setupChoice := utils.PromptChoice(reader, "Select setup type", setupOptions)

	// API Settings
	fmt.Println(utils.Colors.Blue("--- API Settings ---"))
	apiKey := utils.PromptString(reader, "API Key", viper.GetString("API_KEY"))
	viper.Set("API_KEY", apiKey)
	apiURL := utils.PromptString(reader, "API URL", viper.GetString("API_URL"))
	viper.Set("API_URL", apiURL)

	// Database Settings
	fmt.Println(utils.Colors.Blue("--- Database Settings ---"))
	currentDB, err := database.LoadDatabaseSettings(App.State)
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

	if currentDB == nil || currentDB.GetType() != dbType {
		newDB, err := database.LoadDatabaseSettings(App.State)
		if err != nil {
			fmt.Println(utils.Colors.Red("Error creating new database configuration: %v", err))
			return false
		}
		App.DB = newDB
	}

	App.DB.PromptDatabaseSettings()
	if err := App.DB.SetDatabaseSettings(); err != nil {
		fmt.Println(utils.Colors.Red("Error saving database settings: %v", err))
		return false
	}

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
	configFile := utils.GetConfigDirFile("config.yaml")
	if err := viper.WriteConfigAs(configFile); err != nil {
		fmt.Println(utils.Colors.Red("Error saving config file: %v", err))
		return false
	}

	fmt.Println()
	fmt.Println(utils.Colors.Green("✓ Setup completed successfully!"))
	fmt.Println(utils.Colors.Green("Configuration saved to: %s", configFile))

	// Check if the database schema is valid
	if err := App.DB.ValidateSchema(); err == nil {
		fmt.Println(utils.Colors.Yellow("Database schema already exists and is valid."))
		reinitialize := utils.PromptBool(reader, "Do you want to reinitialize the database? (This will delete all existing data)", false)
		if reinitialize {
			fmt.Println(utils.Colors.Yellow("Reinitializing database..."))
			if err := App.DB.DropAllTables(); err != nil {
				fmt.Println(utils.Colors.Red("Error dropping tables: %v", err))
				return false
			}
			if err := App.DB.EnforceSchema(); err != nil {
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
			if err := App.DB.EnforceSchema(); err != nil {
				fmt.Println(utils.Colors.Red("Error enforcing schema: %v", err))
				return false
			}
			fmt.Println(utils.Colors.Green("Database schema created/updated successfully."))
		}
	}

	return true
}
