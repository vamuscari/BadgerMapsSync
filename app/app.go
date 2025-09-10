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

	State          *state.State
	DB             database.DB
	API            *api.APIClient
	AdvancedConfig *AdvancedConfig
}

func NewApplication() *App {
	return &App{
		State:          state.NewState(),
		AdvancedConfig: &AdvancedConfig{MaxConcurrentRequests: 10},
	}
}

func (a *App) LoadConfiguration(cmd *cobra.Command) error {
	path, ok, err := a.GetConfigFilePath(cmd)
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

	var apiClient api.APIClient
	if err := viper.Unmarshal(&apiClient); err != nil {
		return fmt.Errorf("error unmarshalling API config: %w", err)
	}
	a.API = api.NewAPIClient(apiClient.APIKey, apiClient.BaseURL)

	var dbErr error
	a.DB, dbErr = database.LoadDatabaseSettings(a.State)
	if dbErr != nil {
		return dbErr
	}

	var advancedConfig AdvancedConfig
	if err := viper.Unmarshal(&advancedConfig); err != nil {
		return fmt.Errorf("error unmarshalling advanced config: %w", err)
	}
	if advancedConfig.MaxConcurrentRequests == 0 {
		advancedConfig.MaxConcurrentRequests = 10
	}
	a.AdvancedConfig = &advancedConfig

	return nil
}

func (a *App) VerifySetupOrExit(cmd *cobra.Command) bool {
	if a.State.NoColor {
		color.NoColor = true
	}

	path, ok, err := a.GetConfigFilePath(cmd)
	if err != nil {
		fmt.Println(color.RedString("Error getting config file path: %v", err))
		os.Exit(1)
	}

	if ok {
		if a.State.Verbose || a.State.Debug {
			fmt.Println(color.GreenString("Configuration detected: %s", path))
		}
		if err := a.LoadConfiguration(cmd); err != nil {
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
		if InteractiveSetup(a, cmd) {
			if err := a.LoadConfiguration(cmd); err != nil {
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

func (a *App) GetConfigFilePath(cmd *cobra.Command) (string, bool, error) {
	// Highest precedence: --config flag from App struct
	if a.CfgFile != "" {
		absPath, err := filepath.Abs(a.CfgFile)
		if err != nil {
			return "", false, fmt.Errorf("error getting absolute path for %s: %w", a.CfgFile, err)
		}
		return absPath, true, nil
	}

	// Second precedence: --env flag
	envFile, _ := cmd.Flags().GetString("env")
	if cmd.Flags().Changed("env") {
		if strings.TrimSpace(envFile) == "" {
			exe, err := os.Executable()
			if err != nil {
				return "", false, fmt.Errorf("error getting executable path: %w", err)
			}
			return filepath.Join(filepath.Dir(exe), ".env"), true, nil
		}
		absPath, err := filepath.Abs(envFile)
		if err != nil {
			return "", false, fmt.Errorf("error getting absolute path for %s: %w", envFile, err)
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
