package app

import (
	"badgermaps/api"
	"badgermaps/app/action"
	"badgermaps/app/server"
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/events"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	WebhookAccountCreate = "account_create"
	WebhookCheckin       = "checkin"

	ThemePreferenceAuto  = "auto"
	ThemePreferenceLight = "light"
	ThemePreferenceDark  = "dark"
)

type ServerConfig struct {
	Host        string          `yaml:"host"`
	Port        int             `yaml:"port"`
	TLSEnabled  bool            `yaml:"tls_enabled"`
	TLSCert     string          `yaml:"tls_cert"`
	TLSKey      string          `yaml:"tls_key"`
	LogRequests bool            `yaml:"log_requests"`
	Webhooks    map[string]bool `yaml:"webhooks"`
}

func defaultWebhookConfig() map[string]bool {
	return map[string]bool{
		WebhookAccountCreate: true,
		WebhookCheckin:       true,
	}
}

type Config struct {
	API                   api.APIConfig        `yaml:"api"`
	DB                    database.DBConfig    `yaml:"db"`
	Server                ServerConfig         `yaml:"server"`
	ThemePreference       string               `yaml:"theme_preference"`
	MaxConcurrentRequests int                  `yaml:"max_concurrent_requests"`
	EventActions          []action.EventAction `yaml:"event_actions"`
	CronJobs              []server.CronJob     `yaml:"cron_jobs"`
	WebhookCatchAll       bool                 `yaml:"webhook_catch_all"`
	LogFile               string               `yaml:"log_file"`
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
	LogListener    *events.LogListener

	MaxConcurrentRequests int

	syncHistoryRuns map[string]*syncHistoryRun
	syncHistoryMu   sync.Mutex
	syncHistoryOnce bool
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
	if a.LogListener != nil {
		a.LogListener.Close()
	}
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
			Server: ServerConfig{
				Host:        "localhost",
				Port:        8080,
				LogRequests: true,
				Webhooks:    defaultWebhookConfig(),
			},
			ThemePreference:       ThemePreferenceAuto,
			MaxConcurrentRequests: 5,
		},
	}
	a.State.PIDFile = utils.GetConfigDirFile(".badgermaps.pid")
	a.Events = events.NewEventDispatcher()
	a.Server = server.NewServerManager(a.State)
	a.syncHistoryRuns = make(map[string]*syncHistoryRun)

	return a
}

func (a *App) InitLogging() error {
	logPath := a.State.LogFile
	if logPath == "" {
		logPath = a.Config.LogFile
	}

	var err error
	a.LogListener, err = events.NewLogListener(a.State, logPath)
	if err != nil {
		return err
	}

	a.Events.Subscribe("log", a.LogListener.Handle)
	return nil
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
		a.ensureExecActionShellDefaults()
	}
	a.ensureServerWebhookDefaults()
	a.ensureThemePreference()

	// Transfer server config to state
	a.State.ServerHost = a.Config.Server.Host
	a.State.ServerPort = a.Config.Server.Port
	a.State.TLSEnabled = a.Config.Server.TLSEnabled
	a.State.TLSCert = a.Config.Server.TLSCert
	a.State.TLSKey = a.Config.Server.TLSKey
	a.State.ServerLogRequests = a.Config.Server.LogRequests

	a.API = api.NewAPIClient(&a.Config.API)

	var dbErr error
	a.DB, dbErr = database.NewDB(&a.Config.DB)
	if dbErr != nil {
		a.Events.Dispatch(events.Errorf("db", "Failed to initialize database handle: %v", dbErr))
		a.DB = nil
	} else if a.DB != nil {
		if err := a.DB.Connect(); err != nil {
			a.Events.Dispatch(events.Errorf("db", "Failed to connect to database: %v", err))
			a.DB.Close()
			a.DB = nil
		} else {
			a.DB.TestConnection()
		}
	}

	a.ActionExecutor = action.NewExecutor(a.DB, a.API)
	a.Events.Subscribe("*", func(event events.Event) {
		execCtx := &action.ExecutionContext{
			EventType: string(event.Type),
			Source:    event.Source,
			Payload:   event.Payload,
		}
		for _, eventAction := range a.Config.EventActions {
			if eventAction.Event == string(event.Type) && (eventAction.Source == "" || eventAction.Source == event.Source) {
				for _, actionConfig := range eventAction.Run {
					ac := actionConfig
					execCopy := execCtx
					go func(cfg action.ActionConfig, ctx *action.ExecutionContext) {
						if err := a.ExecuteActionWithContext(cfg, ctx); err != nil {
							a.Events.Dispatch(events.Errorf("action", "Error executing action: %v", err))
						}
					}(ac, execCopy)
				}
			}
		}
	})

	// Respect configured concurrency before enforcing the default bounds.
	a.MaxConcurrentRequests = a.Config.MaxConcurrentRequests
	// Limit between
	if a.MaxConcurrentRequests < 1 || a.MaxConcurrentRequests > 10 {
		a.MaxConcurrentRequests = 5
	}

	a.ensureSyncHistoryTracking()

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

func (a *App) ensureExecActionShellDefaults() {
	if a.ConfigFile == "" {
		return
	}

	updated := false
	for i := range a.Config.EventActions {
		for j := range a.Config.EventActions[i].Run {
			actionConfig := &a.Config.EventActions[i].Run[j]
			if actionConfig.Type != "exec" {
				continue
			}
			if actionConfig.Args == nil {
				actionConfig.Args = make(map[string]interface{})
			}
			if _, ok := actionConfig.Args["use_shell"]; !ok {
				actionConfig.Args["use_shell"] = true
				updated = true
			}
		}
	}

	if !updated {
		return
	}

	if err := a.SaveConfig(); err != nil {
		a.Events.Dispatch(events.Warningf("config", "Failed to save config after backfilling exec use_shell flag: %v", err))
	} else {
		a.Events.Dispatch(events.Infof("config", "Configuration updated to backfill exec action 'use_shell' flag."))
	}
}

func (a *App) ensureServerWebhookDefaults() {
	if len(a.Config.Server.Webhooks) == 0 {
		a.Config.Server.Webhooks = defaultWebhookConfig()
		return
	}

	for key, defaultValue := range defaultWebhookConfig() {
		if _, ok := a.Config.Server.Webhooks[key]; !ok {
			a.Config.Server.Webhooks[key] = defaultValue
		}
	}
}

func (a *App) ensureThemePreference() {
	a.Config.ThemePreference = NormalizeThemePreference(a.Config.ThemePreference)
}

func NormalizeThemePreference(pref string) string {
	switch strings.ToLower(strings.TrimSpace(pref)) {
	case ThemePreferenceLight:
		return ThemePreferenceLight
	case ThemePreferenceDark:
		return ThemePreferenceDark
	case ThemePreferenceAuto:
		return ThemePreferenceAuto
	default:
		return ThemePreferenceAuto
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
				a.Events.Dispatch(events.Event{Type: "action.config.updated", Source: "events", Payload: events.ActionConfigUpdatedPayload{}})
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
		a.Events.Dispatch(events.Event{Type: "action.config.created", Source: "events", Payload: events.ActionConfigCreatedPayload{}})
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
				a.Events.Dispatch(events.Event{Type: "action.config.updated", Source: "events", Payload: events.ActionConfigUpdatedPayload{}})
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
				a.Events.Dispatch(events.Event{Type: "action.config.deleted", Source: "events", Payload: events.ActionConfigDeletedPayload{}})
			}
			return err
		}
	}
	return fmt.Errorf("event action not found: %s", eventName)
}

func (a *App) ExecuteAction(actionConfig action.ActionConfig) error {
	return a.ExecuteActionWithContext(actionConfig, nil)
}

func (a *App) ExecuteActionWithContext(actionConfig action.ActionConfig, execCtx *action.ExecutionContext) error {
	logSource := resolveActionLogSource(execCtx)
	actionInstance, err := action.NewActionFromConfig(actionConfig)
	if err != nil {
		a.Events.Dispatch(events.Errorf(logSource, "error creating action: %v", err))
		return err
	}

	if err := actionInstance.Validate(); err != nil {
		a.Events.Dispatch(events.Errorf(logSource, "invalid action configuration: %v", err))
		return err
	}

	a.Events.Dispatch(events.Debugf(logSource, "Executing action type '%s'", actionConfig.Type))

	baseExecutor := a.ActionExecutor
	if baseExecutor == nil {
		baseExecutor = action.NewExecutor(a.DB, a.API)
		a.ActionExecutor = baseExecutor
	}
	executorWithContext := baseExecutor.WithContext(execCtx)
	actionType := actionConfig.Type

	go func(execCopy *action.Executor) { // run in a goroutine to not block the GUI
		if err := actionInstance.Execute(execCopy); err != nil {
			a.Events.Dispatch(events.Errorf(logSource, "action '%s' failed: %v", actionType, err))
		} else {
			a.Events.Dispatch(events.Debugf(logSource, "Action '%s' completed successfully", actionType))
		}
	}(executorWithContext)

	return nil
}

func resolveActionLogSource(ctx *action.ExecutionContext) string {
	if ctx == nil {
		return "manual_run"
	}
	if ctx.EventType != "" {
		return fmt.Sprintf("action.%s", ctx.EventType)
	}
	return "action"
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
		// We can't use the event system yet, so print directly
		fmt.Fprintf(os.Stderr, "Error getting config file path: %v\n", err)
		os.Exit(1)
	}

	if ok {
		if err := a.LoadConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		}
	}

	// Initialize logging now that config and flags are loaded
	if err := a.InitLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logger: %v\n", err)
		os.Exit(1)
	}

	if ok {
		a.Events.Dispatch(events.Infof("config", "Configuration detected: %s", path))
		if (a.State.Verbose || a.State.Debug) && a.API != nil {
			apiKeyStatus := "not set"
			if a.API.APIKey != "" {
				apiKeyStatus = "set"
			}
			dbType := "none"
			if a.DB != nil {
				dbType = a.DB.GetType()
			}
			a.Events.Dispatch(events.Debugf("config", "Setup OK: DB_TYPE=%s, API_KEY=%s", dbType, apiKeyStatus))
		}
		return
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

	fmt.Println(utils.Colors.Blue("=== Badger Maps Sync Setup ==="))
	fmt.Println()
	fmt.Println(utils.Colors.Yellow("This will guide you through setting up the BadgerMapsSync."))

	// Only set the config file path if it's not already set from LoadConfig.
	// This ensures we save to the detected config location rather than always
	// overwriting it with the default user config directory path.
	if a.ConfigFile == "" {
		a.ConfigFile = utils.GetConfigDirFile("config.yaml")
	}

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
		fmt.Println(utils.Colors.Red("✗ Failed to load database settings: %v", err))
		return false
	}
	if err := a.DB.Connect(); err != nil {
		fmt.Println(utils.Colors.Red("✗ Database connection failed: %v", err))
		fmt.Println(utils.Colors.Yellow("Please check your database settings and try again."))
		return false
	}
	if err := a.DB.TestConnection(); err != nil {
		fmt.Println(utils.Colors.Red("✗ Database connection failed: %v", err))
		fmt.Println(utils.Colors.Yellow("Please check your database settings and try again."))
		return false
	}
	fmt.Println(utils.Colors.Green("✓ Database connection successful"))

	// Advanced Settings
	fmt.Println(utils.Colors.Blue("---" + " Advanced Settings ---"))
	a.Config.MaxConcurrentRequests = utils.PromptInt(reader, "Max Concurrent Requests", a.Config.MaxConcurrentRequests)

	// Save configuration
	if err := a.SaveConfig(); err != nil {
		fmt.Println(utils.Colors.Red("✗ Error saving config file: %v", err))
		return false
	}

	fmt.Println()
	fmt.Println(utils.Colors.Green("✓ Configuration saved to: %s", a.ConfigFile))

	// Check if the database schema is valid
	if err := a.DB.ValidateSchema(a.State); err == nil {
		fmt.Println(utils.Colors.Yellow("⚠ Database schema already exists and is valid."))
		reinitialize := utils.PromptBool(reader, "Do you want to reinitialize the database? (This will delete all existing data)", false)
		if reinitialize {
			fmt.Println(utils.Colors.Yellow("Re-initializing database schema and deleting all existing data..."))
			if err := a.DB.ResetSchema(a.State); err != nil {
				fmt.Println(utils.Colors.Red("✗ Error resetting schema: %v", err))
				return false
			}
			fmt.Println(utils.Colors.Green("✓ Database reinitialized successfully"))
		}
	} else {
		// Schema is invalid or does not exist
		fmt.Println(utils.Colors.Yellow("⚠ Database schema is invalid or missing."))
		enforce := utils.PromptBool(reader, "Do you want to create/update the database schema now?", true)
		if enforce {
			fmt.Println(utils.Colors.Cyan("Enforcing schema..."))
			if err := a.DB.EnforceSchema(a.State); err != nil {
				fmt.Println(utils.Colors.Red("✗ Error enforcing schema: %v", err))
				return false
			}
			fmt.Println(utils.Colors.Green("✓ Database schema created/updated successfully"))
		}
	}

	return true
}
