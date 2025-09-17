package app

import (
	"badgermaps/api"
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/events"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	API                   api.APIConfig              `yaml:"api"`
	DB                    database.DBConfig          `yaml:"db"`
	MaxConcurrentRequests int                        `yaml:"max_concurrent_requests"`
	Actions               map[string][]events.ActionConfig `yaml:"actions"`
}

type App struct {
	ConfigFile string

	State  *state.State
	Config *Config
	DB     database.DB
	API    *api.APIClient
	Events *events.EventDispatcher

	MaxConcurrentRequests int
}

func (a *App) GetState() *state.State {
	return a.State
}

func (a *App) GetConfig() *events.AppConfig {
	return &events.AppConfig{
		Events: a.Config.Actions,
	}
}

func (a *App) GetDB() database.DB {
	return a.DB
}

func (a *App) GetAPI() *api.APIClient {
	return a.API
}

func NewApp() *App {
	a := &App{
		State: state.NewState(),
		Config: &Config{
			API: api.APIConfig{
				BaseURL: api.DefaultApiBaseURL,
			},
			DB: database.DBConfig{
				Type: "sqlite3",
				Path: utils.GetConfigDirFile("badgermaps.db"),
			},
			MaxConcurrentRequests: 5,
		},
	}
	a.State.PIDFile = utils.GetConfigDirFile(".badgermaps.pid")
	a.Events = events.NewEventDispatcher(a)

	// Register the log listener
	logListener := events.NewLogListener(a)
	a.Events.Subscribe(events.LogEvent, logListener.Handle)

	return a
}

func (a *App) LoadConfig() error {
	path, ok, err := a.GetConfigFilePath()
	if err != nil {
		return err
	}

	if ok {
		a.ConfigFile = path
		// Load from YAML file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(data, a.Config)
		if err != nil {
			return err
		}
	}

	a.API = api.NewAPIClient(&a.Config.API)

	var dbErr error
	a.DB, dbErr = database.NewDB(&a.Config.DB)
	if dbErr != nil {
		return dbErr
	}

	// Limit between
	if a.MaxConcurrentRequests < 1 || a.MaxConcurrentRequests > 10 {
		a.MaxConcurrentRequests = 5
	}

	return nil
}

func (a *App) SaveConfig() error {
	if a.ConfigFile == "" {
		return fmt.Errorf("no configuration file loaded, cannot save")
	}
	return a.writeYamlFile(a.ConfigFile)
}

func (a *App) writeYamlFile(path string) error {
	data, err := yaml.Marshal(a.Config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (a *App) ReloadDB() error {
	if a.DB != nil {
		a.DB.Close()
	}
	var err error
	a.DB, err = database.NewDB(&a.Config.DB)
	return err
}

func (a *App) GetConfigFilePath() (string, bool, error) {
	// Second precedence: --config flag
	if a.State.ConfigFile != nil && *a.State.ConfigFile != "" {
		absPath, err := filepath.Abs(*a.State.ConfigFile)
		if err != nil {
			return "", false, fmt.Errorf("error getting absolute path for %s: %w", *a.State.ConfigFile, err)
		}
		return absPath, true, nil
	}

	// Auto-detection logic
	// 1. Check local config.yaml
	if utils.CheckIfFileExists(filepath.Join(".", "config.yaml")) {
		return filepath.Join(".", "config.yaml"), true, nil
	}
	// 2. Check user config directory
	userConfigPath := utils.GetConfigDirFile("config.yaml")
	if utils.CheckIfFileExists(userConfigPath) {
		return userConfigPath, true, nil
	}

	return "", false, nil
}

func (a *App) AddEventAction(event, source, actionString string) error {
	if a.Config.Actions == nil {
		a.Config.Actions = make(map[string][]events.ActionConfig)
	}

	// This is a temporary solution to keep the GUI working.
	// A better solution would be to have a dedicated UI for creating actions.
	parts := strings.SplitN(actionString, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid action format: %s", actionString)
	}
	actionConfig := events.ActionConfig{
		Type: parts[0],
		Args: map[string]interface{}{
			"command":  parts[1], // Assuming exec action for now
			"function": parts[1], // Assuming db action for now
			"endpoint": parts[1], // Assuming api action for now
		},
	}

	key := fmt.Sprintf("on_%s", event)
	if source != "" {
		key = fmt.Sprintf("%s_%s", key, source)
	}
	a.Config.Actions[key] = append(a.Config.Actions[key], actionConfig)
	err := a.SaveConfig()
	if err == nil {
		a.Events.Dispatch(events.Event{Type: events.EventCreate, Source: "events", Payload: map[string]string{"event": event, "action": actionString}})
	}
	return err
}

func (a *App) GetEventActions() map[string][]events.ActionConfig {
	return a.Config.Actions
}

func (a *App) RemoveEventAction(event, source, actionToRemove string) error {
	key := fmt.Sprintf("on_%s", event)
	if source != "" {
		key = fmt.Sprintf("%s_%s", key, source)
	}
	actions, ok := a.Config.Actions[key]
	if !ok {
		return nil
	}
	var newActions []events.ActionConfig
	for _, _ = range actions {
		// This is a temporary solution. We need a better way to identify actions to remove.
		// For now, we just don't support removing actions from the GUI.
		// This will be improved in a future update.
		// if action != actionToRemove {
		// 	newActions = append(newActions, action)
		// }
	}
	a.Config.Actions[key] = newActions
	err := a.SaveConfig()
	if err == nil {
		a.Events.Dispatch(events.Event{Type: events.EventDelete, Source: "events", Payload: map[string]string{"event": event, "action": actionToRemove}})
	}
	return nil
}

// TriggerEventAction executes a specific action string (e.g., "db:my_function", "exec:ls").
func (a *App) TriggerEventAction(action string) error {
	parts := strings.SplitN(action, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid action format: %s", action)
	}
	actionType := parts[0]
	actionValue := parts[1]

	switch actionType {
	case "db":
		go func(functionName string) {
			if err := a.DB.RunFunction(functionName); err != nil {
				fmt.Printf("Error executing db function '%s': %v\n", functionName, err)
			}
		}(actionValue)
	case "api":
		go func(endpoint string) {
			if _, err := a.API.GetRaw(endpoint); err != nil {
				fmt.Printf("Error executing api action '%s': %v\n", endpoint, err)
			}
		}(actionValue)
	case "exec":
		go func(command string) {
			cmd := exec.Command("sh", "-c", command)
			if err := cmd.Run(); err != nil {
				fmt.Printf("Error executing command '%s': %v\n", command, err)
			}
		}(actionValue)
	default:
		return fmt.Errorf("unknown action type: %s", actionType)
	}
	return nil
}

func (a *App) EnsureConfig(isGui bool) {
	if a.State.NoColor {
		utils.InitColors(a.State)
	}

	path, ok, err := a.GetConfigFilePath()
	if err != nil {
		fmt.Println(utils.Colors.Red("Error getting config file path: %v", err))
		os.Exit(1)
	}

	if ok {
		a.Events.Dispatch(events.Infof("config", "Configuration detected: %s", path))
		if err := a.LoadConfig(); err != nil {
			a.Events.Dispatch(events.Errorf("config", "Error loading configuration: %v", err))
		} else {
			if (a.State.Verbose || a.State.Debug) && a.API != nil {
				apiKeyStatus := "not set"
				if a.API.APIKey != "" {
					apiKeyStatus = "set"
				}
				dbType := a.DB.GetType()
				a.Events.Dispatch(events.Debugf("config", "Setup OK: DB_TYPE=%s, API_KEY=%s", dbType, apiKeyStatus))
			}
			return
		}
	}

	// If running in GUI mode and no config is found, just load the defaults and continue.
	if isGui {
		if a.State.Verbose || a.State.Debug {
			fmt.Println(utils.Colors.Yellow("No configuration file found. Loading default settings for GUI."))
		}
		// LoadConfig with no path will initialize with defaults
		if err := a.LoadConfig(); err != nil {
			fmt.Println(utils.Colors.Red("Error loading default configuration: %v", err))
			// In GUI mode, we might still want to continue with a broken config to allow fixing it.
		}
		return
	}

	if a.State.Verbose || a.State.Debug {
		fmt.Println(utils.Colors.Yellow("No configuration file found (config.yaml)."))
	}

	if a.State.NoInput {
		fmt.Println(utils.Colors.Red("No configuration file found and interactive prompts are disabled. Exiting."))
		os.Exit(1)
	}

	if promptForSetup() {
		if a.InteractiveSetup() {
			if err := a.LoadConfig(); err != nil {
				fmt.Println(utils.Colors.Red("Error reloading configuration after setup: %v", err))
				os.Exit(1)
			}
			return
		}
	}

	fmt.Println(utils.Colors.Yellow("Setup is required to use this command. Exiting."))
	os.Exit(0)
}

func promptForSetup() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Would you like to run the setup wizard? (y/N) ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)
	return strings.ToLower(response) == "y"
}

func (a *App) InteractiveSetup() bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(utils.Colors.Blue("=== BadgerMaps CLI Setup ==="))
	fmt.Println(utils.Colors.Yellow("This will guide you through setting up the BadgerMaps CLI."))
	fmt.Println()

	a.ConfigFile = utils.GetConfigDirFile("config.yaml")

	// API Settings
	fmt.Println(utils.Colors.Blue("---" + " API Settings ---"))
	a.Config.API.APIKey = utils.PromptString(reader, "API Key", a.Config.API.APIKey)
	a.Config.API.BaseURL = utils.PromptString(reader, "API URL", a.Config.API.BaseURL)

	// Database Settings
	fmt.Println(utils.Colors.Blue("---" + " Database Settings ---"))

	dbOptions := []string{"sqlite3", "postgres", "mssql"}
	dbType := utils.PromptChoice(reader, "Select database type", dbOptions)
	a.Config.DB.Type = dbType

	switch dbType {
	case "sqlite3":
		a.Config.DB.Path = utils.PromptString(reader, "Database Path", a.Config.DB.Path)
	case "postgres":
		a.Config.DB.Host = utils.PromptString(reader, "Database Host", a.Config.DB.Host)
		a.Config.DB.Port = utils.PromptInt(reader, "Database Port", a.Config.DB.Port)
		a.Config.DB.Database = utils.PromptString(reader, "Database Name", a.Config.DB.Database)
		a.Config.DB.Username = utils.PromptString(reader, "Database Username", a.Config.DB.Username)
		a.Config.DB.Password = utils.PromptPassword(reader, "Database Password", a.Config.DB.Password)
		a.Config.DB.SSLMode = utils.PromptString(reader, "Database SSL Mode", a.Config.DB.SSLMode)
	case "mssql":
		a.Config.DB.Host = utils.PromptString(reader, "Database Host", a.Config.DB.Host)
		a.Config.DB.Port = utils.PromptInt(reader, "Database Port", a.Config.DB.Port)
		a.Config.DB.Database = utils.PromptString(reader, "Database Name", a.Config.DB.Database)
		a.Config.DB.Username = utils.PromptString(reader, "Database Username", a.Config.DB.Username)
		a.Config.DB.Password = utils.PromptPassword(reader, "Database Password", a.Config.DB.Password)
	}

	// Test the new connection before proceeding
	fmt.Println()
	fmt.Println(utils.Colors.Cyan("Testing database connection..."))
	var err error
	a.DB, err = database.NewDB(&a.Config.DB)
	if err != nil {
		fmt.Println(utils.Colors.Red("Failed to load database settings: %v", err))
		return false
	}
	if err := a.DB.TestConnection(); err != nil {
		fmt.Println(utils.Colors.Red("Database connection failed: %v", err))
		fmt.Println(utils.Colors.Yellow("Please check your database settings and try again."))
		return false
	}
	fmt.Println(utils.Colors.Green("Database connection successful."))

	// Advanced Settings
	fmt.Println(utils.Colors.Blue("---" + " Advanced Settings ---"))
	a.Config.MaxConcurrentRequests = utils.PromptInt(reader, "Max Concurrent Requests", a.Config.MaxConcurrentRequests)

	// Save configuration
	if err := a.SaveConfig(); err != nil {
		fmt.Println(utils.Colors.Red("Error saving config file: %v", err))
		return false
	}

	fmt.Println()
	fmt.Println(utils.Colors.Green("✓ Setup completed successfully!"))
	fmt.Println(utils.Colors.Green("Configuration saved to: %s", a.ConfigFile))

	// Check if the database schema is valid
	if err := a.DB.ValidateSchema(a.State); err == nil {
		fmt.Println(utils.Colors.Yellow("Database schema already exists and is valid."))
		reinitialize := utils.PromptBool(reader, "Do you want to reinitialize the database? (This will delete all existing data)", false)
		if reinitialize {
			fmt.Println(utils.Colors.Yellow("Reinitializing database..."))
			if err := a.DB.DropAllTables(); err != nil {
				fmt.Println(utils.Colors.Red("Error dropping tables: %v", err))
				return false
			}
			if err := a.DB.EnforceSchema(a.State); err != nil {
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
			if err := a.DB.EnforceSchema(a.State); err != nil {
				fmt.Println(utils.Colors.Red("Error enforcing schema: %v", err))
				return false
			}
			fmt.Println(utils.Colors.Green("Database schema created/updated successfully."))
		}
	}

	return true
}
