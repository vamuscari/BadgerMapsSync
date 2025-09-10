package app

import (
	"badgermaps/api"
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type App struct {
	CfgFile          string
	LoadedConfigFile string

	State  *state.State
	DB     database.DB
	API    *api.APIClient
	Events *EventDispatcher

	MaxConcurrentRequests int
}

func NewApp() *App {
	return &App{
		State:  state.NewState(),
		Events: NewEventDispatcher(),
	}
}

func (a *App) LoadConfig() error {
	path, ok, err := a.GetConfigFilePath()
	if err != nil {
		return err
	}

	if ok {
		a.LoadedConfigFile = path
		viper.SetConfigFile(path)
		if err := viper.ReadInConfig(); err != nil {
			// Ignore if file is empty, it will be written to later
			if _, ok := err.(viper.ConfigParseError); !ok {
				return fmt.Errorf("error reading config file: %w", err)
			}
		}
	}

	a.API = api.NewAPIClient()

	var dbErr error
	a.DB, dbErr = database.LoadDatabaseSettings(a.State)
	if dbErr != nil {
		return dbErr
	}

	a.MaxConcurrentRequests = viper.GetInt("MAX_CONCURRENT_REQUESTS")

	// Limit between
	if a.MaxConcurrentRequests < 1 || a.MaxConcurrentRequests > 10 {
		a.MaxConcurrentRequests = 5
	}

	return nil
}

func (a *App) SaveConfig() {
	a.API.SaveConfig()
	a.DB.SaveConfig()
	viper.Set("MAX_CONCURRENT_REQUESTS", a.MaxConcurrentRequests)

}

func (a *App) VerifySetupOrExit(cmd *cobra.Command) bool {
	if a.State.NoColor {
		color.NoColor = true
	}

	path, ok, err := a.GetConfigFilePath()
	if err != nil {
		fmt.Println(color.RedString("Error getting config file path: %v", err))
		os.Exit(1)
	}

	if ok {
		if a.State.Verbose || a.State.Debug {
			fmt.Println(color.GreenString("Configuration detected: %s", path))
		}
		if err := a.LoadConfig(); err != nil {
			fmt.Println(color.RedString("Error loading configuration: %v", err))
		} else {
			if (a.State.Verbose || a.State.Debug) && a.API != nil {
				apiKeyStatus := "not set"
				if a.API.APIKey != "" {
					apiKeyStatus = "set"
				}
				dbType := a.DB.GetType()
				fmt.Println(color.CyanString("Setup OK: DB_TYPE=%s, API_KEY=%s", dbType, apiKeyStatus))
			}
			return true
		}
	}

	if a.State.Verbose || a.State.Debug {
		fmt.Println(color.YellowString("No configuration file found (.env or config.yaml)."))
	}

	if promptForSetup() {
		if InteractiveSetup(a) {
			if err := a.LoadConfig(); err != nil {
				fmt.Println(color.RedString("Error reloading configuration after setup: %v", err))
				os.Exit(1)
			}
			return true
		}
	}

	fmt.Println(color.YellowString("Setup is required to use this command. Exiting."))
	os.Exit(0)
	return false
}

func (a *App) GetConfigFilePath() (string, bool, error) {

	// Highest precedence: --env flag
	if a.State.EnvFile != nil && *a.State.EnvFile != "" {
		absPath, err := filepath.Abs(*a.State.EnvFile)
		if err != nil {
			return "", false, fmt.Errorf("error getting absolute path for %s: %w", *a.State.EnvFile, err)
		}
		return absPath, true, nil
	}

	// Second precedence: --config flag
	if a.State.ConfigFile != nil && *a.State.ConfigFile != "" {
		absPath, err := filepath.Abs(*a.State.ConfigFile)
		if err != nil {
			return "", false, fmt.Errorf("error getting absolute path for %s: %w", *a.State.ConfigFile, err)
		}
		return absPath, true, nil
	}

	// Auto-detection logic: .env takes precedence
	// 1. Check local .env
	if utils.CheckIfFileExists(".env") {
		return ".env", true, nil
	}
	// 2. Check user config directory
	userConfigPath := utils.GetConfigDirFile("config.yaml")
	if utils.CheckIfFileExists(userConfigPath) {
		return userConfigPath, true, nil
	}
	// 3. Check local config.yaml
	if utils.CheckIfFileExists(filepath.Join(".", "config.yaml")) {
		return filepath.Join(".", "config.yaml"), true, nil
	}

	return "", false, nil
}

func promptForSetup() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(color.YellowString("BadgerMaps CLI is not set up. Would you like to run setup now? [y/N]: "))
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// WriteConfig saves the current viper configuration to the loaded config file.
func (a *App) WriteConfig() error {
	if a.LoadedConfigFile == "" {
		return fmt.Errorf("no configuration file loaded, cannot save")
	}

	return viper.WriteConfigAs(a.LoadedConfigFile)
}
