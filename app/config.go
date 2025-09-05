package app

import (
	"badgermaps/database"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

// Config represents the global configuration for the application
type Config struct {
	App *State
	// API settings
	APIKey string
	APIURL string

	// Database settings
	DBType    string
	DBConnStr string

	// Rate limiting
	RateLimitRequests int
	RateLimitPeriod   int

	// Parallel processing
	MaxParallelProcesses int

	// Server settings
	ServerHost       string
	ServerPort       int
	ServerTLSEnable  bool
	ServerTLSCert    string
	ServerTLSKey     string
	WebhookSecret    string
}

func defaultConfig(application *State) *Config {

	config := &Config{
		App:                  application,
		DBType:               "sqlite3",
		DBConnStr:            "badgermaps.db",
		APIKey:               "",
		APIURL:               "https://badgerapis.badgermapping.com/api/2",
		RateLimitRequests:    100,
		RateLimitPeriod:      60,
		MaxParallelProcesses: 10,
		ServerHost:           "localhost",
		ServerPort:           8080,
		ServerTLSEnable:      false,
		ServerTLSCert:        "",
		ServerTLSKey:         "",
		WebhookSecret:        "",
	}

	config.App.DB = &database.SQLiteConfig{
		utils.GetConfigDirFile("badgermaps.db"),
	}

	return config
}

// LoadConfig creates a new Config instance with values from viper
func LoadConfig(application *State) *Config {

	path, ok := GetConfigFilePath()
	if ok {
		viper.SetConfigFile(path)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Println(utils.Colors.Red("Error reading config file: %v", err))
			return nil
		}
	}

	// Create config with all fields
	config := &Config{
		App: application,
		// Database settings
		DBType:    viper.GetString("DB_TYPE"),
		DBConnStr: viper.GetString("DB_CONN_STR"),

		// API settings
		APIKey: viper.GetString("API_KEY"),
		APIURL: viper.GetString("API_URL"),

		// Rate limiting
		RateLimitRequests: viper.GetInt("RATE_LIMIT_REQUESTS"),
		RateLimitPeriod:   viper.GetInt("RATE_LIMIT_PERIOD"),

		// Parallel processing
		MaxParallelProcesses: viper.GetInt("MAX_PARALLEL_PROCESSES"),

		// Server settings
		ServerHost:      viper.GetString("SERVER_HOST"),
		ServerPort:      viper.GetInt("SERVER_PORT"),
		ServerTLSEnable: viper.GetBool("SERVER_TLS_ENABLED"),
		ServerTLSCert:   viper.GetString("SERVER_TLS_CERT"),
		ServerTLSKey:    viper.GetString("SERVER_TLS_KEY"),
		WebhookSecret:   viper.GetString("WEBHOOK_SECRET"),
	}

	var err error
	config.App.DB, err = database.LoadDatabaseSettings(config.DBType)
	if err != nil {
		fmt.Println(err)
		return config
	}

	return config
}

// GetConfigFilePath returns the path to the config
// First checks for an environment file, then checks for a config file, then checks for a config file in the config directory
// If none of these files exist, returns an empty string and a boolean indicating that the file does not exist
func GetConfigFilePath() (string, bool) {
	hasEnv := utils.CheckIfFileExists(".env")
	if hasEnv {
		return ".env", true
	}

	hasConfigFile := utils.CheckIfFileExists(filepath.Join(".", "config.yaml"))
	if hasConfigFile {
		return filepath.Join(".", "config.yaml"), true
	}

	hasConfigFile = utils.CheckIfFileExists(utils.GetConfigDirFile("config.yaml"))
	if hasConfigFile {
		return utils.GetConfigDirFile("config.yaml"), true
	}

	return "", false
}

// PromptForSetup asks the user if they want to run setup
// Returns true if the user wants to run setup, false otherwise
func promptForSetup() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(color.YellowString("BadgerMaps CLI is not set up. Would you like to run setup now? [y/N]: "))
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
