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
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	Timezone    string          `yaml:"timezone,omitempty"`
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
	API                   api.APIConfig                     `yaml:"api"`
	DB                    database.DBConfig                 `yaml:"db"`
	Server                ServerConfig                      `yaml:"server"`
	ThemePreference       string                            `yaml:"theme_preference"`
	MaxConcurrentRequests int                               `yaml:"max_concurrent_requests"`
	CustomCheckins        bool                              `yaml:"custom_checkins"`
	WorkflowProfiles      map[string]server.WorkflowProfile `yaml:"workflow_profiles"`
	WebhookCatchAll       bool                              `yaml:"webhook_catch_all"`
	LogFile               string                            `yaml:"log_file"`
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
	syncQueue      *server.SyncJobCoordinator

	MaxConcurrentRequests int

	syncHistoryRuns map[string]*syncHistoryRun
	syncHistoryMu   sync.Mutex
	syncHistoryOnce bool
	closeOnce       sync.Once
	shuttingDown    atomic.Bool
	coordMu         sync.Mutex
}

func (a *App) Close() {
	a.closeOnce.Do(func() {
		a.shuttingDown.Store(true)

		if a.Events != nil {
			// Let async event handlers finish before DB resources are torn down.
			a.Events.WaitForDrain(2 * time.Second)
		}

		a.StopSyncCoordinator()
		if a.DB != nil {
			a.DB.Close()
		}
		if a.LogListener != nil {
			a.LogListener.Close()
		}
	})
}

func (a *App) GetState() *state.State {
	return a.State
}

func (a *App) IsShuttingDown() bool {
	return a.shuttingDown.Load()
}

func (a *App) GetDB() database.DB {
	return a.DB
}
func (a *App) GetAPI() *api.APIClient {
	return a.API
}

func (a *App) EnsureSyncCoordinator() *server.SyncJobCoordinator {
	a.coordMu.Lock()
	defer a.coordMu.Unlock()

	if a.syncQueue != nil {
		return a.syncQueue
	}
	a.syncQueue = server.NewSyncJobCoordinator(a.State, a.Events)
	return a.syncQueue
}

func (a *App) GetSyncCoordinator() *server.SyncJobCoordinator {
	a.coordMu.Lock()
	defer a.coordMu.Unlock()
	return a.syncQueue
}

func (a *App) StopSyncCoordinator() {
	a.coordMu.Lock()
	queue := a.syncQueue
	a.syncQueue = nil
	a.coordMu.Unlock()

	if queue != nil {
		queue.Stop()
	}
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
				Timezone:    "",
				LogRequests: true,
				Webhooks:    defaultWebhookConfig(),
			},
			ThemePreference:       ThemePreferenceAuto,
			MaxConcurrentRequests: 5,
			CustomCheckins:        false,
			WorkflowProfiles:      server.DefaultWorkflowProfiles(),
		},
	}
	a.State.PIDFile = utils.GetConfigDirFile(".badgermaps.pid")
	a.Events = events.NewEventDispatcher()
	a.Server = server.NewServerManager(a.State)
	a.syncHistoryRuns = make(map[string]*syncHistoryRun)

	return a
}

// SetConfigFilePath updates the active config path and aligns runtime files with it.
func (a *App) SetConfigFilePath(path string) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return
	}
	if absPath, err := filepath.Abs(trimmed); err == nil {
		trimmed = absPath
	}

	a.ConfigFile = trimmed

	if a.State != nil {
		if a.State.ConfigFile != nil {
			*a.State.ConfigFile = trimmed
		}
		a.State.PIDFile = filepath.Join(filepath.Dir(trimmed), ".badgermaps.pid")
	}
}

func (a *App) InitLogging() error {
	logPath := a.State.LogFile
	if logPath == "" {
		logPath = a.Config.LogFile
	}
	if strings.TrimSpace(logPath) == "" {
		logPath = a.defaultLogFilePath()
	}
	a.State.LogFile = logPath
	if strings.TrimSpace(a.Config.LogFile) == "" {
		a.Config.LogFile = logPath
	}

	var err error
	a.LogListener, err = events.NewLogListener(a.State, logPath)
	if err != nil {
		return err
	}

	a.Events.Subscribe("log", a.LogListener.Handle)
	return nil
}

func (a *App) defaultLogFilePath() string {
	configPath := strings.TrimSpace(a.ConfigFile)
	if configPath == "" {
		if detectedPath, ok, err := a.GetConfigFilePath(); err == nil && ok {
			configPath = strings.TrimSpace(detectedPath)
		}
	}
	if configPath != "" {
		return filepath.Join(filepath.Dir(configPath), "badgermaps.log")
	}
	return utils.GetConfigDirFile("badgermaps.log")
}

func (a *App) LoadConfig() error {
	path, ok, err := a.GetConfigFilePath()
	if err != nil {
		return err
	}

	if ok {
		a.SetConfigFilePath(path)
		// Load from YAML file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := validateLegacyConfigContract(data); err != nil {
			return err
		}
		err = yaml.Unmarshal(data, a.Config)
		if err != nil {
			return err
		}
	}
	if err := a.ensureWorkflowProfiles(); err != nil {
		return err
	}
	a.ensureServerWebhookDefaults()
	a.ensureThemePreference()
	a.Config.Server.Timezone = server.NormalizeTimezone(a.Config.Server.Timezone)
	if err := server.ValidateTimezone(a.Config.Server.Timezone); err != nil {
		return err
	}

	// Transfer server config to state
	a.State.ServerHost = a.Config.Server.Host
	a.State.ServerPort = a.Config.Server.Port
	a.State.ServerTimezone = a.Config.Server.Timezone
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

func (a *App) ensureWorkflowProfiles() error {
	current := server.NormalizeWorkflowProfiles(a.Config.WorkflowProfiles)
	defaults := server.DefaultWorkflowProfiles()
	if current == nil {
		current = map[string]server.WorkflowProfile{}
	}

	for name, profile := range defaults {
		if _, exists := current[name]; !exists {
			current[name] = profile
		}
	}

	if err := server.ValidateWorkflowProfiles(current); err != nil {
		return fmt.Errorf("workflow_profiles validation failed: %w", err)
	}
	a.Config.WorkflowProfiles = current
	return nil
}

func validateLegacyConfigContract(data []byte) error {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse config for legacy validation: %w", err)
	}
	if raw == nil {
		return nil
	}

	if value, exists := raw["event_actions"]; exists && !yamlValueEmpty(value) {
		return fmt.Errorf("legacy config key 'event_actions' is no longer supported; manually rewrite actions into workflow_profiles and scheduled job steps")
	}
	if value, exists := raw["cron_jobs"]; exists && !yamlValueEmpty(value) {
		return fmt.Errorf("legacy config key 'cron_jobs' is no longer supported; manually rewrite cron jobs into scheduled jobs with explicit workflow steps")
	}
	return nil
}

func yamlValueEmpty(value interface{}) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []interface{}:
		return len(typed) == 0
	case map[string]interface{}:
		return len(typed) == 0
	default:
		return false
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

func (a *App) ReloadDB() error {
	if a.DB != nil {
		a.DB.Close()
	}
	var err error
	a.DB, err = database.NewDB(&a.Config.DB)
	if err != nil {
		return err
	}
	if a.DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	if err := a.DB.Connect(); err != nil {
		a.DB.Close()
		a.DB = nil
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := a.DB.TestConnection(); err != nil {
		a.DB.Close()
		a.DB = nil
		return fmt.Errorf("failed to test database connection: %w", err)
	}
	return nil
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
	localConfigPath := filepath.Join(".", "config.yaml")
	if utils.CheckIfFileExists(localConfigPath) {
		absPath, err := filepath.Abs(localConfigPath)
		if err != nil {
			return "", false, fmt.Errorf("error getting absolute path for %s: %w", localConfigPath, err)
		}
		return absPath, true, nil
	}
	// 2. Check user config directory
	userConfigPath := utils.GetConfigDirFile("config.yaml")
	if utils.CheckIfFileExists(userConfigPath) {
		absPath, err := filepath.Abs(userConfigPath)
		if err != nil {
			return "", false, fmt.Errorf("error getting absolute path for %s: %w", userConfigPath, err)
		}
		return absPath, true, nil
	}

	return "", false, nil
}

func (a *App) AddEventAction(event, source string, actionConfig action.ActionConfig) error {
	return fmt.Errorf("event_actions is retired; add action steps under a workflow profile or scheduled job steps")
}
func (a *App) UpdateEventAction(eventName string, actionIndex int, actionConfig action.ActionConfig) error {
	return fmt.Errorf("event_actions is retired; edit action steps under a workflow profile or scheduled job steps")
}

func (a *App) RemoveEventAction(eventName string, actionIndex int) error {
	return fmt.Errorf("event_actions is retired; edit action steps under a workflow profile or scheduled job steps")
}

func (a *App) SetEventActionEnabled(eventName string, actionIndex int, enabled bool) error {
	return fmt.Errorf("event_actions is retired; enable or disable action steps within workflow profiles or scheduled jobs")
}

func (a *App) ListScheduledJobs() ([]*server.ScheduledJob, error) {
	jobsMap, err := server.LoadScheduledJobs(a.State)
	if err != nil {
		return nil, err
	}

	jobs := make([]*server.ScheduledJob, 0, len(jobsMap))
	for id, job := range jobsMap {
		if job == nil {
			continue
		}
		jobCopy := *job
		jobCopy.ID = id
		jobs = append(jobs, &jobCopy)
	}

	sort.SliceStable(jobs, func(i, j int) bool {
		leftName := strings.TrimSpace(jobs[i].Name)
		rightName := strings.TrimSpace(jobs[j].Name)
		if leftName == rightName {
			return jobs[i].ID < jobs[j].ID
		}
		return strings.ToLower(leftName) < strings.ToLower(rightName)
	})

	return jobs, nil
}

func (a *App) ServerTimezoneLocation() *time.Location {
	loc, err := server.ResolveTimezoneLocation(a.Config.Server.Timezone)
	if err != nil || loc == nil {
		return time.Local
	}
	return loc
}

func (a *App) DisplayTimezoneName() string {
	loc := a.ServerTimezoneLocation()
	if loc == nil {
		return time.Local.String()
	}
	return loc.String()
}

func (a *App) FormatTimestampInDisplayTimezone(ts time.Time, layout string) string {
	loc := a.ServerTimezoneLocation()
	if loc == nil {
		loc = time.Local
	}

	trimmedLayout := strings.TrimSpace(layout)
	if trimmedLayout == "" {
		trimmedLayout = time.RFC3339
	}

	formatted := ts.In(loc).Format(trimmedLayout)
	return fmt.Sprintf("%s [%s]", formatted, loc.String())
}

func (a *App) DateInDisplayTimezone(ts time.Time) string {
	loc := a.ServerTimezoneLocation()
	if loc == nil {
		loc = time.Local
	}
	return ts.In(loc).Format("2006-01-02")
}

func (a *App) EffectiveScheduledJobTimezone(job *server.ScheduledJob) (string, string) {
	if job == nil {
		return time.Local.String(), server.TimezoneSourceLocal
	}
	name, source, _, err := server.ResolveEffectiveTimezone(job.Timezone, a.Config.Server.Timezone)
	if err != nil {
		return time.Local.String(), server.TimezoneSourceLocal
	}
	return name, source
}

func (a *App) ensureScheduledJobEditsAllowed() error {
	if a == nil || a.Server == nil {
		return nil
	}
	if _, running := a.Server.GetServerStatus(); running {
		return fmt.Errorf("cannot modify scheduled jobs while server is running; stop the server first")
	}
	return nil
}

func (a *App) UpsertScheduledJob(job *server.ScheduledJob) error {
	if job == nil {
		return fmt.Errorf("job is required")
	}
	if err := a.ensureScheduledJobEditsAllowed(); err != nil {
		return err
	}

	trimmedSchedule := strings.TrimSpace(job.Schedule)
	if trimmedSchedule == "" {
		return fmt.Errorf("job schedule is required")
	}
	if err := server.TestCronExpression(trimmedSchedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	jobsMap, err := server.LoadScheduledJobs(a.State)
	if err != nil {
		return err
	}

	jobID := strings.TrimSpace(job.ID)
	if jobID == "" {
		jobID = server.GenerateScheduledJobID()
	}

	jobCopy := *job
	jobCopy.ID = jobID
	jobCopy.Schedule = trimmedSchedule
	jobCopy.WorkflowProfile = strings.TrimSpace(jobCopy.WorkflowProfile)
	jobCopy.Timezone = server.NormalizeTimezone(jobCopy.Timezone)
	if err := server.ValidateTimezone(jobCopy.Timezone); err != nil {
		return err
	}
	if strings.TrimSpace(jobCopy.Name) == "" {
		jobCopy.Name = jobID
	}
	if err := server.ValidateScheduledJobDefinition(&jobCopy, a.Config.WorkflowProfiles); err != nil {
		return err
	}

	jobsMap[jobID] = &jobCopy
	if err := server.SaveScheduledJobs(a.State, jobsMap); err != nil {
		return err
	}

	a.Events.Dispatch(events.Infof("jobs", "Saved scheduled job '%s'", jobCopy.Name))
	return nil
}

func (a *App) DeleteScheduledJob(jobID string) error {
	trimmedID := strings.TrimSpace(jobID)
	if trimmedID == "" {
		return fmt.Errorf("job id is required")
	}
	if err := a.ensureScheduledJobEditsAllowed(); err != nil {
		return err
	}

	jobsMap, err := server.LoadScheduledJobs(a.State)
	if err != nil {
		return err
	}
	if _, exists := jobsMap[trimmedID]; !exists {
		return fmt.Errorf("job not found: %s", trimmedID)
	}

	delete(jobsMap, trimmedID)
	if err := server.SaveScheduledJobs(a.State, jobsMap); err != nil {
		return err
	}
	a.Events.Dispatch(events.Infof("jobs", "Deleted scheduled job '%s'", trimmedID))
	return nil
}

func (a *App) SetScheduledJobEnabled(jobID string, enabled bool) error {
	trimmedID := strings.TrimSpace(jobID)
	if trimmedID == "" {
		return fmt.Errorf("job id is required")
	}
	if err := a.ensureScheduledJobEditsAllowed(); err != nil {
		return err
	}

	jobsMap, err := server.LoadScheduledJobs(a.State)
	if err != nil {
		return err
	}
	job, exists := jobsMap[trimmedID]
	if !exists || job == nil {
		return fmt.Errorf("job not found: %s", trimmedID)
	}

	job.Enabled = enabled
	jobsMap[trimmedID] = job
	if err := server.SaveScheduledJobs(a.State, jobsMap); err != nil {
		return err
	}

	stateLabel := "resumed"
	if !enabled {
		stateLabel = "paused"
	}
	a.Events.Dispatch(events.Infof("jobs", "Job '%s' %s", trimmedID, stateLabel))
	return nil
}

func (a *App) ExecuteAction(actionConfig action.ActionConfig) error {
	return a.ExecuteActionWithContext(actionConfig, nil)
}

func (a *App) ExecuteActionSyncWithContext(actionConfig action.ActionConfig, execCtx *action.ExecutionContext) error {
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

	baseExecutor := a.ActionExecutor
	if baseExecutor == nil {
		baseExecutor = action.NewExecutor(a.DB, a.API)
		a.ActionExecutor = baseExecutor
	}
	executorWithContext := baseExecutor.WithContext(execCtx)
	if err := actionInstance.Execute(executorWithContext); err != nil {
		a.Events.Dispatch(events.Errorf(logSource, "action '%s' failed: %v", actionConfig.Type, err))
		return err
	}
	a.Events.Dispatch(events.Debugf(logSource, "Action '%s' completed successfully", actionConfig.Type))
	return nil
}

func (a *App) ExecuteActionWithContext(actionConfig action.ActionConfig, execCtx *action.ExecutionContext) error {
	a.Events.Dispatch(events.Debugf(resolveActionLogSource(execCtx), "Executing action type '%s'", actionConfig.Type))
	go func() { // run in a goroutine to not block the GUI
		_ = a.ExecuteActionSyncWithContext(actionConfig, execCtx)
	}()
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
		a.SetConfigFilePath(utils.GetConfigDirFile("config.yaml"))
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
	a.Config.CustomCheckins = utils.PromptBool(reader, "Enable Custom Checkins API", a.Config.CustomCheckins)

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
