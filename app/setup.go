package app

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// InteractiveSetup guides the user through setting up the configuration
// It creates both the config.yaml and .env files with user-provided values
func InteractiveSetup(app *Application) bool {
	var QuickSetup bool = true
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(Colors.Blue("=== BadgerMaps CLI Setup ==="))
	fmt.Println(Colors.Yellow("This will guide you through setting up the BadgerMaps CLI."))
	fmt.Println(Colors.Yellow("Press Enter to accept the default value shown in [brackets]."))
	fmt.Println()

	// Create a new config instance
	config := defaultConfig(app)

	// Load existing configuration if it exists
	path, ok := GetConfigFilePath()
	if ok {
		viper.SetConfigFile(path)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Println(Colors.Red("Error reading config file: %v", err))
			return false
		}
		config = LoadConfig(app)
	}

	// QuickSetup vs AdvancedSetup
	QuickSetup = promptBool(reader, "Quick Setup?", QuickSetup)

	// Ensure config directory exists
	if err := EnsureDirExists(GetConfigDir()); err != nil {
		fmt.Println(Colors.Red("Error creating config directory: %v", err))
		return false
	}

	// Ensure cache directory exists
	if err := EnsureDirExists(GetCacheDir()); err != nil {
		fmt.Println(Colors.Red("Error creating cache directory: %v", err))
		return false
	}

	// API Settings
	fmt.Println(Colors.Blue("--- API Settings ---"))

	// API Key
	apiKey := promptString(reader, "API Key", config.APIKey)
	viper.Set("API_KEY", apiKey)

	// API URL
	config.APIURL = "https://badgerapis.badgermapping.com/api/2"
	if QuickSetup {
		defaultAPIURL := "https://badgerapis.badgermapping.com/api/2"
		if config.APIURL == "" {
			config.APIURL = defaultAPIURL
		}
		config.APIURL = promptString(reader, "API URL", config.APIURL)
	}
	viper.Set("API_URL", config.APIURL)

	if !QuickSetup {
		// Server Settings
		fmt.Println()
		fmt.Println(Colors.Blue("--- Server Settings ---"))

		// Server Host
		defaultServerHost := "localhost"
		if config.ServerHost == "" {
			config.ServerHost = defaultServerHost
		}
		serverHost := promptString(reader, "Server Host", config.ServerHost)
		viper.Set("SERVER_HOST", serverHost)

		// Server Port
		defaultServerPort := 8080
		if config.ServerPort == 0 {
			config.ServerPort = defaultServerPort
		}
		serverPort := promptInt(reader, "Server Port", config.ServerPort)
		viper.Set("SERVER_PORT", serverPort)

		// Server TLS Enable
		serverTLS := promptBool(reader, "Enable TLS", config.ServerTLSEnable)
		viper.Set("SERVER_TLS_ENABLED", serverTLS)

		// Rate Limiting Settings
		fmt.Println()
		fmt.Println(Colors.Blue("--- Rate Limiting Settings ---"))

		// Rate Limit Requests
		defaultRateLimitRequests := 100
		if config.RateLimitRequests == 0 {
			config.RateLimitRequests = defaultRateLimitRequests
		}
		rateLimitRequests := promptInt(reader, "Rate Limit Requests", config.RateLimitRequests)
		viper.Set("RATE_LIMIT_REQUESTS", rateLimitRequests)

		// Rate Limit Period
		defaultRateLimitPeriod := 60
		if config.RateLimitPeriod == 0 {
			config.RateLimitPeriod = defaultRateLimitPeriod
		}
		rateLimitPeriod := promptInt(reader, "Rate Limit Period (seconds)", config.RateLimitPeriod)
		viper.Set("RATE_LIMIT_PERIOD", rateLimitPeriod)

		// Parallel Processing Settings
		fmt.Println()
		fmt.Println(Colors.Blue("--- Parallel Processing Settings ---"))

		// Max Parallel Processes
		defaultMaxParallelProcesses := 5
		if config.MaxParallelProcesses == 0 {
			config.MaxParallelProcesses = defaultMaxParallelProcesses
		}
		maxParallelProcesses := promptInt(reader, "Max Parallel Processes", config.MaxParallelProcesses)
		viper.Set("MAX_PARALLEL_PROCESSES", maxParallelProcesses)

	}

	// Save configuration
	configFile := GetConfigDirFile("config.yaml")
	if !QuickSetup {
		configFile = promptChoice(reader, "Pick Config Save Location", []string{configFile, ".env"})
	}

	if err := viper.SafeWriteConfigAs(configFile); err != nil {
		fmt.Println(Colors.Red("Error saving config file: %v", err))
		return false
	}

	fmt.Println()
	fmt.Println(Colors.Green("✓ Setup completed successfully!"))
	fmt.Println(Colors.Green("Configuration saved to: %s", configFile))

	return true
}
