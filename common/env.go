package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// CreateEnvFile creates a new .env file with default configuration values
// If the file already exists, it will not be overwritten unless force is true
func CreateEnvFile(force bool) error {
	// Check if .env file already exists
	envPath := ".env"
	if _, err := os.Stat(envPath); err == nil && !force {
		return fmt.Errorf(".env file already exists. Use --force to overwrite")
	}

	// Default configuration values
	defaultConfig := []string{
		"# BadgerMaps CLI Configuration",
		"",
		"# API Configuration",
		"API_KEY=",
		"API_URL=https://badgerapis.badgermapping.com/api/2",
		"",
		"# Database Configuration",
		"DATABASE_TYPE=sqlite3",
		"DATABASE_NAME=badgermaps.db",
		"DATABASE_HOST=",
		"DATABASE_PORT=",
		"DATABASE_USERNAME=",
		"DATABASE_PASSWORD=",
		"",
		"# Rate Limiting",
		"RATE_LIMIT_REQUESTS=100",
		"RATE_LIMIT_PERIOD=60",
		"",
		"# Parallel Processing",
		"MAX_PARALLEL_PROCESSES=5",
		"",
		"# Server Configuration",
		"SERVER_HOST=localhost",
		"SERVER_PORT=8080",
		"SERVER_TLS_ENABLED=false",
	}

	// Create the file
	file, err := os.Create(envPath)
	if err != nil {
		return fmt.Errorf("failed to create .env file: %w", err)
	}
	defer file.Close()

	// Write configuration to file
	content := strings.Join(defaultConfig, "\n")
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to .env file: %w", err)
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(envPath)
	if err != nil {
		absPath = envPath
	}

	fmt.Println(color.GreenString("Created .env file at %s", absPath))
	return nil
}

// CheckEnvFileExists checks if the .env file exists
func CheckEnvFileExists() bool {
	_, err := os.Stat(".env")
	return err == nil
}

// LoadEnvFile loads the .env file into viper
// This is a placeholder for future implementation if needed
func LoadEnvFile() error {
	// This would be implemented if we need to manually load .env files
	// Currently, we're relying on viper's functionality
	return nil
}
