package app

import (
	"badgermapscli/utils"
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// InteractiveSetup guides the user through setting up the configuration
// It creates both the config.yaml and .env files with user-provided values
func InteractiveSetup(app *State) bool {
	var QuickSetup bool = true
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(utils.Colors.Blue("=== BadgerMaps CLI Setup ==="))
	fmt.Println(utils.Colors.Yellow("This will guide you through setting up the BadgerMaps CLI."))
	fmt.Println(utils.Colors.Yellow("Press Enter to accept the default value shown in [brackets]."))
	fmt.Println()

	// Create a new config instance
	config := defaultConfig(app)

	// Load existing configuration if it exists
	path, ok := GetConfigFilePath()
	if ok {
		viper.SetConfigFile(path)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Println(utils.Colors.Red("Error reading config file: %v", err))
			return false
		}
		config = LoadConfig(app)
	}

	// QuickSetup vs AdvancedSetup
	QuickSetup = utils.PromptBool(reader, "Quick Setup?", QuickSetup)

	// Ensure config directory exists
	if err := utils.EnsureDirExists(utils.GetConfigDir()); err != nil {
		fmt.Println(utils.Colors.Red("Error creating config directory: %v", err))
		return false
	}

	// API Settings
	fmt.Println(utils.Colors.Blue("--- API Settings ---"))

	// API Key
	apiKey := utils.PromptString(reader, "API Key", config.APIKey)
	viper.Set("API_KEY", apiKey)

	// API URL
	config.APIURL = "https://badgerapis.badgermapping.com/api/2"
	if QuickSetup {
		defaultAPIURL := "https://badgerapis.badgermapping.com/api/2"
		if config.APIURL == "" {
			config.APIURL = defaultAPIURL
		}
		config.APIURL = utils.PromptString(reader, "API URL", config.APIURL)
	}
	viper.Set("API_URL", config.APIURL)

	if !QuickSetup {
		// Server Settings
		fmt.Println()
		fmt.Println(utils.Colors.Blue("--- Server Settings ---"))

		// Server Host
		defaultServerHost := "localhost"
		if config.ServerHost == "" {
			config.ServerHost = defaultServerHost
		}
		serverHost := utils.PromptString(reader, "Server Host", config.ServerHost)
		viper.Set("SERVER_HOST", serverHost)

		// Server Port
		defaultServerPort := 8080
		if config.ServerPort == 0 {
			config.ServerPort = defaultServerPort
		}
		serverPort := utils.PromptInt(reader, "Server Port", config.ServerPort)
		viper.Set("SERVER_PORT", serverPort)

		// Server TLS Enable
		serverTLS := utils.PromptBool(reader, "Enable TLS", config.ServerTLSEnable)
		viper.Set("SERVER_TLS_ENABLED", serverTLS)

		// Rate Limiting Settings
		fmt.Println()
		fmt.Println(utils.Colors.Blue("--- Rate Limiting Settings ---"))

		// Rate Limit Requests
		defaultRateLimitRequests := 100
		if config.RateLimitRequests == 0 {
			config.RateLimitRequests = defaultRateLimitRequests
		}
		rateLimitRequests := utils.PromptInt(reader, "Rate Limit Requests", config.RateLimitRequests)
		viper.Set("RATE_LIMIT_REQUESTS", rateLimitRequests)

		// Rate Limit Period
		defaultRateLimitPeriod := 60
		if config.RateLimitPeriod == 0 {
			config.RateLimitPeriod = defaultRateLimitPeriod
		}
		rateLimitPeriod := utils.PromptInt(reader, "Rate Limit Period (seconds)", config.RateLimitPeriod)
		viper.Set("RATE_LIMIT_PERIOD", rateLimitPeriod)

		// Parallel Processing Settings
		fmt.Println()
		fmt.Println(utils.Colors.Blue("--- Parallel Processing Settings ---"))

		// Max Parallel Processes
		defaultMaxParallelProcesses := 5
		if config.MaxParallelProcesses == 0 {
			config.MaxParallelProcesses = defaultMaxParallelProcesses
		}
		maxParallelProcesses := utils.PromptInt(reader, "Max Parallel Processes", config.MaxParallelProcesses)
		viper.Set("MAX_PARALLEL_PROCESSES", maxParallelProcesses)

	}

	// Save configuration
	configFile := utils.GetConfigDirFile("config.yaml")
	if !QuickSetup {
		configFile = utils.PromptChoice(reader, "Pick Config Save Location", []string{configFile, ".env"})
	}

	if err := viper.SafeWriteConfigAs(configFile); err != nil {
		fmt.Println(utils.Colors.Red("Error saving config file: %v", err))
		return false
	}

	fmt.Println()
	fmt.Println(utils.Colors.Green("✓ Setup completed successfully!"))
	fmt.Println(utils.Colors.Green("Configuration saved to: %s", configFile))

	return true
}
