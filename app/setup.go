package app

import (
	"badgermaps/database"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"github.com/spf13/viper"
	"os"
)

// InteractiveSetup guides the user through setting up the configuration
func InteractiveSetup(app *State) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(utils.Colors.Blue("=== BadgerMaps CLI Setup ==="))
	fmt.Println(utils.Colors.Yellow("This will guide you through setting up the BadgerMaps CLI."))
	fmt.Println(utils.Colors.Yellow("Press Enter to accept the default value shown in [brackets]."))
	fmt.Println()

	// API Settings
	fmt.Println(utils.Colors.Blue("--- API Settings ---"))
	apiKey := utils.PromptString(reader, "API Key", viper.GetString("API_KEY"))
	viper.Set("API_KEY", apiKey)
	apiURL := utils.PromptString(reader, "API URL", viper.GetString("API_URL"))
	viper.Set("API_URL", apiURL)

	// Database Settings
	fmt.Println(utils.Colors.Blue("--- Database Settings ---"))
	// Try to load the current configuration first
	currentDB, err := database.LoadDatabaseSettings()
	if err != nil {
		fmt.Println(utils.Colors.Yellow("Could not load current database configuration: %v", err))
	}

	// Prompt the user to select a database type
	dbType := utils.PromptChoice(reader, "Select database type", []string{"sqlite3", "postgres", "mssql"})
	viper.Set("DB_TYPE", dbType)

	// If the selected database type is different from the current one,
	// we need to create a new database configuration
	if currentDB == nil || currentDB.GetType() != dbType {
		newDB, err := database.LoadDatabaseSettings()
		if err != nil {
			fmt.Println(utils.Colors.Red("Error creating new database configuration: %v", err))
			return false
		}
		app.DB = newDB
	}

	// Run the promptDatabaseSettings() to configure the database
	app.DB.PromptDatabaseSettings()

	// After run SetDatabaseSettings to save the DB
	if err := app.DB.SetDatabaseSettings(); err != nil {
		fmt.Println(utils.Colors.Red("Error saving database settings: %v", err))
		return false
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

	return true
}
