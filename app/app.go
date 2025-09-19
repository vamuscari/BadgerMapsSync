package app

import (
	"badgermaps/api"
	"badgermaps/app/server"
	"badgermaps/app/state"
	"badgermaps/cli/action"
	"badgermaps/database"
	"badgermaps/events"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	API                   api.APIConfig        `yaml:"api"`
	DB                    database.DBConfig    `yaml:"db"`
	MaxConcurrentRequests int                  `yaml:"max_concurrent_requests"`
	EventActions          []action.EventAction `yaml:"event_actions"`
}

type App struct {
	ConfigFile string

	State          *state.State
	Config         *Config
	DB             database.DB
	API            *api.APIClient
	Events         *events.EventDispatcher
	Server         *server.ServerManager
	ActionExecutor *action.Executor

	MaxConcurrentRequests int
}

func (a *App) GetState() *state.State {
	return a.State
}
func (a *App) GetDB() database.DB {
	return a.DB
}
func (a *App) GetAPI() *api.APIClient {
	return a.API
}
func (a *App) RawRequest(method, endpoint string, data map[string]string) ([]byte, error) {
	var body string
	var err error
	switch strings.ToUpper(method) {
	case "GET":
		body, err = a.API.GetRaw(endpoint)
	case "POST":
		body, err = a.API.PostRaw(endpoint, data)
	case "PATCH":
		body, err = a.API.PatchRaw(endpoint, data)
	case "DELETE":
		body, err = a.API.DeleteRaw(endpoint)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}
	if err != nil {
		return nil, err
	}
	return []byte(body), nil
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
	a.Events = events.NewEventDispatcher()
	a.Server = server.NewServerManager(a.State)

	// Register the log listener
	logListener := events.NewLogListener(a.State)
	a.Events.Subscribe(events.LogEvent, logListener.Handle)
	a.Events.Subscribe(events.Debug, logListener.Handle)

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
		a.migrateActionNames()
		a.validateAndCleanActions()
	}

	a.API = api.NewAPIClient(&a.Config.API)

	var dbErr error
	a.DB, dbErr = database.NewDB(&a.Config.DB)
	if dbErr != nil {
		a.Events.Dispatch(events.Errorf("db", "Failed to connect to database: %v", dbErr))
		a.DB = nil // Set DB to nil to indicate connection failure
	} else if a.DB != nil {
		if err := a.DB.Connect(); err == nil {
			a.DB.TestConnection()
		}
	}

	a.ActionExecutor = action.NewExecutor(a.DB, a.API)
	a.Events.Subscribe(events.Wildcard, func(event events.Event) {
		for _, eventAction := range a.Config.EventActions {
			if eventAction.Event == event.Type.String() && (eventAction.Source == "" || eventAction.Source == event.Source) {
				for _, actionConfig := range eventAction.Run {
					go func(ac action.ActionConfig) {
						if err := a.ExecuteAction(ac); err != nil {
							a.Events.Dispatch(events.Errorf("action", "Error executing action: %v", err))
						}
					}(actionConfig)
				}
			}
		}
	})

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

func (a *App) migrateActionNames() {
	actionsModified := false
	for i := range a.Config.EventActions {
		if strings.HasPrefix(a.Config.EventActions[i].Name, "on_") {
			a.Config.EventActions[i].Name = strings.TrimPrefix(a.Config.EventActions[i].Name, "on_")
			actionsModified = true
		}
	}

	if actionsModified {
		if err := a.SaveConfig(); err != nil {
			a.Events.Dispatch(events.Warningf("config", "Failed to save config after migrating action names: %v", err))
		} else {
			a.Events.Dispatch(events.Infof("config", "Configuration updated to remove 'on_' prefix from action names."))
		}
	}
}

func (a *App) validateAndCleanActions() {
	var validEventActions []action.EventAction
	actionsModified := false

	for _, eventAction := range a.Config.EventActions {
		var validRuns []action.ActionConfig
		for _, actionConfig := range eventAction.Run {
			action, err := action.NewActionFromConfig(actionConfig)
			if err != nil {
				a.Events.Dispatch(events.Warningf("config", "Invalid action config for event '%s': %v. Removing.", eventAction.Name, err))
				actionsModified = true
				continue
			}
			if err := action.Validate(); err != nil {
				a.Events.Dispatch(events.Warningf("config", "Invalid action for event '%s': %v. Removing.", eventAction.Name, err))
				actionsModified = true
				continue
			}
			validRuns = append(validRuns, actionConfig)
		}

		if len(validRuns) > 0 {
			eventAction.Run = validRuns
			validEventActions = append(validEventActions, eventAction)
			if len(validRuns) < len(eventAction.Run) {
				actionsModified = true // Some actions were removed from this event
			}
		} else {
			a.Events.Dispatch(events.Warningf("config", "Event action '%s' has no valid actions left. Removing.", eventAction.Name))
			actionsModified = true
		}
	}

	if actionsModified {
		a.Config.EventActions = validEventActions
		if err := a.SaveConfig(); err != nil {
			a.Events.Dispatch(events.Warningf("config", "Failed to save config after removing invalid actions: %v", err))
		} else {
			a.Events.Dispatch(events.Infof("config", "Configuration updated to remove invalid actions."))
		}
	}
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

func (a *App) AddEventAction(event, source string, actionConfig action.ActionConfig) error {
	key := event
	if source != "" {
		key = fmt.Sprintf("%s_%s", key, source)
	}

	// Find existing EventAction
	for i := range a.Config.EventActions {
		if a.Config.EventActions[i].Event == event && a.Config.EventActions[i].Source == source {
			a.Config.EventActions[i].Run = append(a.Config.EventActions[i].Run, actionConfig)
			err := a.SaveConfig()
			if err == nil {
				a.Events.Dispatch(events.Event{Type: events.ActionConfigUpdated, Source: "events"})
			}
			return err
		}
	}

	// Not found, create a new one
	newEventAction := action.EventAction{
		Name:   key,
		Event:  event,
		Source: source,
		Run:    []action.ActionConfig{actionConfig},
	}
	a.Config.EventActions = append(a.Config.EventActions, newEventAction)
	err := a.SaveConfig()
	if err == nil {
		a.Events.Dispatch(events.Event{Type: events.ActionConfigCreated, Source: "events"})
	}
	return err
}
func (a *App) UpdateEventAction(eventName string, actionIndex int, actionConfig action.ActionConfig) error {
	for i := range a.Config.EventActions {
		if a.Config.EventActions[i].Name == eventName {
			if actionIndex < 0 || actionIndex >= len(a.Config.EventActions[i].Run) {
				return fmt.Errorf("invalid action index")
			}
			a.Config.EventActions[i].Run[actionIndex] = actionConfig
			err := a.SaveConfig()
			if err == nil {
				a.Events.Dispatch(events.Event{Type: events.ActionConfigUpdated, Source: "events"})
			}
			return err
		}
	}
	return fmt.Errorf("event action not found: %s", eventName)
}

func (a *App) RemoveEventAction(eventName string, actionIndex int) error {
	for i := range a.Config.EventActions {
		if a.Config.EventActions[i].Name == eventName {
			if actionIndex < 0 || actionIndex >= len(a.Config.EventActions[i].Run) {
				return fmt.Errorf("invalid action index")
			}
			a.Config.EventActions[i].Run = append(a.Config.EventActions[i].Run[:actionIndex], a.Config.EventActions[i].Run[actionIndex+1:]...)

			// If the event has no more actions, remove the event itself
			if len(a.Config.EventActions[i].Run) == 0 {
				a.Config.EventActions = append(a.Config.EventActions[:i], a.Config.EventActions[i+1:]...)
			}

			err := a.SaveConfig()
			if err == nil {
				a.Events.Dispatch(events.Event{Type: events.ActionConfigDeleted, Source: "events"})
			}
			return err
		}
	}
	return fmt.Errorf("event action not found: %s", eventName)
}

func (a *App) ExecuteAction(actionConfig action.ActionConfig) error {
	action, err := action.NewActionFromConfig(actionConfig)
	if err != nil {
		a.Events.Dispatch(events.Errorf("manual_run", "error creating action: %v", err))
		return err
	}

	if err := action.Validate(); err != nil {
		a.Events.Dispatch(events.Errorf("manual_run", "invalid action configuration: %v", err))
		return err
	}

	a.Events.Dispatch(events.Debugf("manual_run", "Executing action type '%s'", actionConfig.Type))

	go func() { // run in a goroutine to not block the GUI
		if err := action.Execute(a.ActionExecutor); err != nil {
			a.Events.Dispatch(events.Errorf("manual_run", "action '%s' failed: %v", actionConfig.Type, err))
		} else {
			a.Events.Dispatch(events.Debugf("manual_run", "Action '%s' completed successfully", actionConfig.Type))
		}
	}()

	return nil
}

// TriggerEventAction executes a specific action string (e.g., "db:my_function", "exec:ls").
func (a *App) TriggerEventAction(actionString string) error {
	parts := strings.SplitN(actionString, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid action format: %s", actionString)
	}
	actionType := parts[0]
	actionValue := parts[1]

	var actionConfig action.ActionConfig
	switch actionType {
	case "db":
		actionConfig = action.ActionConfig{
			Type: "db",
			Args: map[string]interface{}{"command": actionValue},
		}
	case "api":
		actionConfig = action.ActionConfig{
			Type: "api",
			Args: map[string]interface{}{"endpoint": actionValue},
		}
	case "exec":
		actionConfig = action.ActionConfig{
			Type: "exec",
			Args: map[string]interface{}{"command": actionValue},
		}
	default:
		return fmt.Errorf("unknown action type: %s", actionType)
	}
	return a.ExecuteAction(actionConfig)
}

func (a *App) EnsureConfig(isGui bool) {
	if a.State.NoColor {
		utils.InitColors(a.State)
	}

	path, ok, err := a.GetConfigFilePath()
	if err != nil {
		a.Events.Dispatch(events.Errorf("config", "Error getting config file path: %v", err))
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
			a.Events.Dispatch(events.Warningf("config", "No configuration file found. Loading default settings for GUI."))
		}
		// LoadConfig with no path will initialize with defaults
		if err := a.LoadConfig(); err != nil {
			a.Events.Dispatch(events.Errorf("config", "Error loading default configuration: %v", err))
			// In GUI mode, we might still want to continue with a broken config to allow fixing it.
		}
		return
	}

	if a.State.Verbose || a.State.Debug {
		a.Events.Dispatch(events.Warningf("config", "No configuration file found (config.yaml)."))
	}

	if a.State.NoInput {
		a.Events.Dispatch(events.Errorf("config", "No configuration file found and interactive prompts are disabled. Exiting."))
		os.Exit(1)
	}

	if promptForSetup() {
		if a.InteractiveSetup() {
			if err := a.LoadConfig(); err != nil {
				a.Events.Dispatch(events.Errorf("config", "Error reloading configuration after setup: %v", err))
				os.Exit(1)
			}
			return
		}
	}

	a.Events.Dispatch(events.Warningf("config", "Setup is required to use this command. Exiting."))
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
		a.Events.Dispatch(events.Errorf("setup", "Failed to load database settings: %v", err))
		return false
	}
	if err := a.DB.TestConnection(); err != nil {
		a.Events.Dispatch(events.Errorf("setup", "Database connection failed: %v", err))
		a.Events.Dispatch(events.Warningf("setup", "Please check your database settings and try again."))
		return false
	}
	a.Events.Dispatch(events.Infof("setup", "Database connection successful."))

	// Advanced Settings
	fmt.Println(utils.Colors.Blue("---" + " Advanced Settings ---"))
	a.Config.MaxConcurrentRequests = utils.PromptInt(reader, "Max Concurrent Requests", a.Config.MaxConcurrentRequests)

	// Save configuration
	if err := a.SaveConfig(); err != nil {
		a.Events.Dispatch(events.Errorf("setup", "Error saving config file: %v", err))
		return false
	}

	fmt.Println()
	a.Events.Dispatch(events.Infof("setup", "✓ Setup completed successfully!"))
	a.Events.Dispatch(events.Infof("setup", "Configuration saved to: %s", a.ConfigFile))

	// Check if the database schema is valid
	if err := a.DB.ValidateSchema(a.State); err == nil {
		a.Events.Dispatch(events.Warningf("setup", "Database schema already exists and is valid."))
		reinitialize := utils.PromptBool(reader, "Do you want to reinitialize the database? (This will delete all existing data)", false)
		if reinitialize {
			a.Events.Dispatch(events.Warningf("setup", "Reinitializing database..."))
			if err := a.DB.DropAllTables(); err != nil {
				a.Events.Dispatch(events.Errorf("setup", "Error dropping tables: %v", err))
				return false
			}
			if err := a.DB.EnforceSchema(a.State); err != nil {
				a.Events.Dispatch(events.Errorf("setup", "Error enforcing schema: %v", err))
				return false
			}
			a.Events.Dispatch(events.Infof("setup", "Database reinitialized successfully."))
		}
	} else {
		// Schema is invalid or does not exist
		a.Events.Dispatch(events.Warningf("setup", "Database schema is invalid or missing."))
		enforce := utils.PromptBool(reader, "Do you want to create/update the database schema now?", true)
		if enforce {
			a.Events.Dispatch(events.Warningf("setup", "Enforcing schema..."))
			if err := a.DB.EnforceSchema(a.State); err != nil {
				a.Events.Dispatch(events.Errorf("setup", "Error enforcing schema: %v", err))
				return false
			}
			a.Events.Dispatch(events.Infof("setup", "Database schema created/updated successfully."))
		}
	}

	return true
}
