package app

import (
	"badgermaps/api"
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	API                   api.APIConfig           `yaml:"api"`
	DB                    database.DBConfig       `yaml:"db"`
	MaxConcurrentRequests int                     `yaml:"max_concurrent_requests"`
	Events                map[string][]string `yaml:"events"`
}

type App struct {
	ConfigFile string

	State  *state.State
	Config *Config
	DB     database.DB
	API    *api.APIClient
	Events *EventDispatcher

	MaxConcurrentRequests int
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
		if strings.HasSuffix(path, ".env") {
			// Load from .env file
			// This is a simplified implementation. A more robust solution would handle
			// comments, quoted values, etc.
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					a.setConfigValue(key, value)
				}
			}
		} else {
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
	}

	a.API = api.NewAPIClient(&a.Config.API)

	var dbErr error
	a.DB, dbErr = database.NewDB(&a.Config.DB, a.State)
	if dbErr != nil {
		return dbErr
	}

	// Limit between
	if a.MaxConcurrentRequests < 1 || a.MaxConcurrentRequests > 10 {
		a.MaxConcurrentRequests = 5
	}

	return nil
}

func (a *App) setConfigValue(key, value string) {
	switch key {
	case "API_URL":
		a.Config.API.BaseURL = value
	case "API_KEY":
		a.Config.API.APIKey = value
	case "DB_TYPE":
		a.Config.DB.Type = value
	case "DB_HOST":
		a.Config.DB.Host = value
	case "DB_PORT":
		a.Config.DB.Port, _ = strconv.Atoi(value)
	case "DB_NAME":
		a.Config.DB.Database = value
	case "DB_USER":
		a.Config.DB.Username = value
	case "DB_PASSWORD":
		a.Config.DB.Password = value
	case "DB_SSL_MODE":
		a.Config.DB.SSLMode = value
	case "DB_PATH":
		a.Config.DB.Path = value
	case "MAX_CONCURRENT_REQUESTS":
		a.Config.MaxConcurrentRequests, _ = strconv.Atoi(value)
	}
}

func (a *App) SaveConfig() error {
	if a.ConfigFile == "" {
		return fmt.Errorf("no configuration file loaded, cannot save")
	}

	if strings.HasSuffix(a.ConfigFile, ".env") {
		return a.writeEnvFile(a.ConfigFile)
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

func (a *App) writeEnvFile(path string) error {
	settings := make(map[string]string)
	settings["API_URL"] = a.Config.API.BaseURL
	settings["API_KEY"] = a.Config.API.APIKey
	settings["MAX_CONCURRENT_REQUESTS"] = fmt.Sprintf("%d", a.Config.MaxConcurrentRequests)
	settings["DB_TYPE"] = a.Config.DB.Type
	settings["DB_HOST"] = a.Config.DB.Host
	settings["DB_PORT"] = fmt.Sprintf("%d", a.Config.DB.Port)
	settings["DB_NAME"] = a.Config.DB.Database
	settings["DB_USER"] = a.Config.DB.Username
	settings["DB_PASSWORD"] = a.Config.DB.Password
	settings["DB_SSL_MODE"] = a.Config.DB.SSLMode
	settings["DB_PATH"] = a.Config.DB.Path

	return utils.WriteEnvFile(path, settings)
}

func (a *App) ReloadDB() error {
	if a.DB != nil {
		a.DB.Close()
	}
	var err error
	a.DB, err = database.NewDB(&a.Config.DB, a.State)
	return err
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

func (a *App) AddEventAction(event string, action string) error {
	if a.Config.Events == nil {
		a.Config.Events = make(map[string][]string)
	}
	key := fmt.Sprintf("on_%s", event)
	a.Config.Events[key] = append(a.Config.Events[key], action)
	err := a.SaveConfig()
	if err == nil {
		a.Events.Dispatch(Event{Type: EventCreate, Source: "events", Payload: map[string]string{"event": event, "action": action}})
	}
	return err
}

func (a *App) GetEventActions() map[string][]string {
	return a.Config.Events
}

func (a *App) RemoveEventAction(event string, actionToRemove string) error {
	key := fmt.Sprintf("on_%s", event)
	actions, ok := a.Config.Events[key]
	if !ok {
		return nil
	}
	var newActions []string
	for _, action := range actions {
		if action != actionToRemove {
			newActions = append(newActions, action)
		}
	}
	a.Config.Events[key] = newActions
	err := a.SaveConfig()
	if err == nil {
			a.Events.Dispatch(Event{Type: EventDelete, Source: "events", Payload: map[string]string{"event": event, "action": actionToRemove}})
		}
		return err
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

func (a *App) EnsureConfig() {
	if a.State.NoColor {
		utils.InitColors(a.State)
	}

	path, ok, err := a.GetConfigFilePath()
	if err != nil {
		fmt.Println(utils.Colors.Red("Error getting config file path: %v", err))
		os.Exit(1)
	}

	if ok {
		if a.State.Verbose || a.State.Debug {
			fmt.Println(utils.Colors.Green("Configuration detected: %s", path))
		}
		if err := a.LoadConfig(); err != nil {
			fmt.Println(utils.Colors.Red("Error loading configuration: %v", err))
		} else {
			if (a.State.Verbose || a.State.Debug) && a.API != nil {
				apiKeyStatus := "not set"
				if a.API.APIKey != "" {
					apiKeyStatus = "set"
				}
			dbType := a.DB.GetType()
			fmt.Println(utils.Colors.Cyan("Setup OK: DB_TYPE=%s, API_KEY=%s", dbType, apiKeyStatus))
			}
			return
		}
	}

	if a.State.Verbose || a.State.Debug {
		fmt.Println(utils.Colors.Yellow("No configuration file found (.env or config.yaml)."))
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

	setupOptions := []string{"Quick Setup", "Advanced Setup"}
	setupChoice := utils.PromptChoice(reader, "Select setup type", setupOptions)

	if setupChoice == "Advanced Setup" {
		userSpecifiedFile := (a.State.EnvFile != nil) || (a.State.ConfigFile != nil)

		if !userSpecifiedFile {
			configOptions := []string{"config.yaml (in user config directory)", ".env file (in executable's directory)"}
			configChoice := utils.PromptChoice(reader, "Select configuration file type to save to", configOptions)
			if configChoice == configOptions[1] {
				exe, err := os.Executable()
				if err != nil {
					fmt.Println(utils.Colors.Red("Error getting executable path: %v", err))
					return false
				}
				a.ConfigFile = filepath.Join(filepath.Dir(exe), ".env")
			} else {
				a.ConfigFile = utils.GetConfigDirFile("config.yaml")
			}
		}
	} else {
		a.ConfigFile = utils.GetConfigDirFile("config.yaml")
	}

	// API Settings
	fmt.Println(utils.Colors.Blue("--- API Settings ---"))
	a.Config.API.APIKey = utils.PromptString(reader, "API Key", a.Config.API.APIKey)
	a.Config.API.BaseURL = utils.PromptString(reader, "API URL", a.Config.API.BaseURL)

	// Database Settings
	fmt.Println(utils.Colors.Blue("--- Database Settings ---"))

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
	a.DB, err = database.NewDB(&a.Config.DB, a.State)
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

	if setupChoice == "Advanced Setup" {
		// Server Settings
		fmt.Println(utils.Colors.Blue("--- Server Settings ---"))
		// These settings are not yet in the config struct, so we'll just ignore them for now.
		// a.Config.Server.Host = utils.PromptString(reader, "Server Host", a.Config.Server.Host)
		// a.Config.Server.Port = utils.PromptInt(reader, "Server Port", a.Config.Server.Port)
		// a.Config.Server.WebhookSecret = utils.PromptString(reader, "Webhook Secret", a.Config.Server.WebhookSecret)

		// Advanced Settings
		fmt.Println(utils.Colors.Blue("--- Advanced Settings ---"))
		a.Config.MaxConcurrentRequests = utils.PromptInt(reader, "Max Concurrent Requests", a.Config.MaxConcurrentRequests)
	}

	// Save configuration
	if err := a.SaveConfig(); err != nil {
		fmt.Println(utils.Colors.Red("Error saving config file: %v", err))
		return false
	}

	fmt.Println()
	fmt.Println(utils.Colors.Green("✓ Setup completed successfully!"))
	fmt.Println(utils.Colors.Green("Configuration saved to: %s", a.ConfigFile))

	// Check if the database schema is valid
	if err := a.DB.ValidateSchema(); err == nil {
		fmt.Println(utils.Colors.Yellow("Database schema already exists and is valid."))
		reinitialize := utils.PromptBool(reader, "Do you want to reinitialize the database? (This will delete all existing data)", false)
		if reinitialize {
			fmt.Println(utils.Colors.Yellow("Reinitializing database..."))
			if err := a.DB.DropAllTables(); err != nil {
				fmt.Println(utils.Colors.Red("Error dropping tables: %v", err))
				return false
			}
			if err := a.DB.EnforceSchema(); err != nil {
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
			if err := a.DB.EnforceSchema(); err != nil {
				fmt.Println(utils.Colors.Red("Error enforcing schema: %v", err))
				return false
			}
			fmt.Println(utils.Colors.Green("Database schema created/updated successfully."))
		}
	}

	return true
}
