package app

import (
	"badgermaps/api"
	"badgermaps/database"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

type State struct {
	Verbose bool
	Quiet   bool
	Debug   bool
	NoColor bool
	CfgFile string

	DB  database.DB
	API *api.APIClient
}

func NewApplication() *State {
	return &State{
		Verbose: false,
		Quiet:   false,
		Debug:   false,
	}
}

func (a *State) LoadConfiguration() error {
	path, ok := GetConfigFilePath()
	if ok {
		viper.SetConfigFile(path)
		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	var apiClient api.APIClient
	if err := viper.Unmarshal(&apiClient); err != nil {
		return fmt.Errorf("error unmarshalling API config: %w", err)
	}
	a.API = api.NewAPIClient(apiClient.APIKey, apiClient.BaseURL)

	var err error
	a.DB, err = database.LoadDatabaseSettings()
	if err != nil {
		return err
	}

	return nil
}

func (a *State) VerifySetupOrExit() bool {
	if a.NoColor {
		color.NoColor = true
	}

	path, ok := GetConfigFilePath()
	if ok {
		if a.Verbose || a.Debug {
			fmt.Println(color.GreenString("Configuration detected: %s", path))
		}
		if err := a.LoadConfiguration(); err != nil {
			fmt.Println(color.RedString("Error loading configuration: %v", err))
		} else {
			if (a.Verbose || a.Debug) && a.API != nil {
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

	if a.Verbose || a.Debug {
		fmt.Println(color.YellowString("No configuration file found (.env or config.yaml)."))
	}

	if promptForSetup() {
		if InteractiveSetup(a) {
			if err := a.LoadConfiguration(); err != nil {
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

func GetConfigFilePath() (string, bool) {
	if utils.CheckIfFileExists(".env") {
		return ".env", true
	}
	if utils.CheckIfFileExists(filepath.Join(".", "config.yaml")) {
		return filepath.Join(".", "config.yaml"), true
	}
	if utils.CheckIfFileExists(utils.GetConfigDirFile("config.yaml")) {
		return utils.GetConfigDirFile("config.yaml"), true
	}
	return "", false
}

func promptForSetup() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(color.YellowString("BadgerMaps CLI is not set up. Would you like to run setup now? [y/N]: "))
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
