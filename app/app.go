package app

import (
	"badgermaps/api"
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/utils"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type App struct {
	ConfigFile string

	State  *state.State
	DB     database.DB
	API    *api.APIClient
	Events *EventDispatcher

	MaxConcurrentRequests int
}

func NewApp() *App {
	a := &App{
		State: state.NewState(),
	}
	a.Events = NewEventDispatcher(a)
	return a
}

func (a *App) LoadConfig() error {
	path, ok, err := a.GetConfigFilePath()
	if err != nil {
		return err
	}

	if ok {
		a.ConfigFile = path
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

// WriteConfig saves the current viper configuration to the loaded config file.
func (a *App) WriteConfig() error {
	if a.ConfigFile == "" {
		return fmt.Errorf("no configuration file loaded, cannot save")
	}

	return viper.WriteConfigAs(a.ConfigFile)
}

func (a *App) AddEventAction(event string, action string) error {
	key := fmt.Sprintf("events.on_%s", event)
	actions := viper.GetStringSlice(key)
	actions = append(actions, action)
	viper.Set(key, actions)
	err := a.WriteConfig()
	if err == nil {
		a.Events.Dispatch(Event{Type: EventCreate, Source: "events", Payload: map[string]string{"event": event, "action": action}})
	}
	return err
}

func (a *App) GetEventActions() map[string][]string {
	eventActions := make(map[string][]string)
	settings := viper.AllSettings()
	events, ok := settings["events"].(map[string]interface{})
	if !ok {
		return eventActions
	}

	for key, val := range events {
		if strings.HasPrefix(key, "on_") {
			eventName := strings.TrimPrefix(key, "on_")
			actions, ok := val.([]interface{})
			if !ok {
				continue
			}
			var actionStrings []string
			for _, act := range actions {
				actionStrings = append(actionStrings, fmt.Sprintf("%v", act))
			}
			eventActions[eventName] = actionStrings
		}
	}
	return eventActions
}

func (a *App) RemoveEventAction(event string, actionToRemove string) error {
	key := fmt.Sprintf("events.on_%s", event)
	actions := viper.GetStringSlice(key)
	var newActions []string
	for _, action := range actions {
		if action != actionToRemove {
			newActions = append(newActions, action)
		}
	}
	viper.Set(key, newActions)
	err := a.WriteConfig()
	if err == nil {
		a.Events.Dispatch(Event{Type: EventDelete, Source: "events", Payload: map[string]string{"event": event, "action": actionToRemove}})
	}
	return err
}
