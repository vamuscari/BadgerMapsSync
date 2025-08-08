package auth

import (
	"badgermapscli/api"
	"badgermapscli/common"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewAuthCmd creates a new auth command
func NewAuthCmd() *cobra.Command {
	var (
		apiKey     string
		storeInEnv bool
	)

	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with the API",
		Long:  `Authenticate with the BadgerMaps API and store your credentials securely.`,
		Run: func(cmd *cobra.Command, args []string) {
			// We no longer automatically create an .env file during authentication
			// Users should use 'badgermaps utils create-env' to create one if needed

			// If no API key is provided, prompt for it
			if apiKey == "" {
				apiKey = promptForAPIKey()
			}

			// Store the API key first, so it's saved even if validation fails
			if err := storeAPIKey(apiKey, storeInEnv); err != nil {
				fmt.Println(color.RedString("Failed to store API key: %v", err))
				os.Exit(1)
			}

			// Validate the API key
			if err := validateAPIKey(apiKey); err != nil {
				fmt.Println(color.RedString("Authentication failed: %v", err))
				fmt.Println(color.YellowString("API key has been stored but is invalid. Please check your API key and try again."))
				os.Exit(1)
			}

			fmt.Println(color.GreenString("Authentication successful"))
		},
	}

	// Add flags
	authCmd.Flags().StringVar(&apiKey, "api-key", "", "BadgerMaps API key")
	authCmd.Flags().BoolVar(&storeInEnv, "store-in-env", false, "Store API key in environment file instead of system keychain")

	// Add subcommands
	authCmd.AddCommand(newAuthValidateCmd())
	authCmd.AddCommand(newAuthStatusCmd())
	authCmd.AddCommand(newAuthClearCmd())

	return authCmd
}

// newAuthValidateCmd creates a command to validate API credentials
func newAuthValidateCmd() *cobra.Command {
	var apiKey string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate API credentials",
		Long:  `Validate your BadgerMaps API credentials without storing them.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If no API key is provided, try to get it from stored credentials
			if apiKey == "" {
				var err error
				apiKey, err = getStoredAPIKey()
				if err != nil {
					fmt.Println(color.YellowString("No stored API key found"))
					apiKey = promptForAPIKey()
				}
			}

			// Validate the API key
			if err := validateAPIKey(apiKey); err != nil {
				fmt.Println(color.RedString("Validation failed: %v", err))
				os.Exit(1)
			}

			fmt.Println(color.GreenString("API key is valid"))
		},
	}

	// Add flags
	cmd.Flags().StringVar(&apiKey, "api-key", "", "BadgerMaps API key")

	return cmd
}

// newAuthStatusCmd creates a command to check authentication status
func newAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		Long:  `Check if you are authenticated with the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Try to get stored API key
			apiKey, err := getStoredAPIKey()
			if err != nil {
				fmt.Println(color.YellowString("Not authenticated"))
				fmt.Println("Use 'badgermaps auth' to authenticate")
				return
			}

			// Validate the API key
			if err := validateAPIKey(apiKey); err != nil {
				fmt.Println(color.YellowString("Authenticated, but API key is invalid"))
				fmt.Println("Use 'badgermaps auth' to update your API key")
				return
			}

			fmt.Println(color.GreenString("Authenticated"))
			fmt.Println("API key is valid")
		},
	}

	return cmd
}

// newAuthClearCmd creates a command to clear stored credentials
func newAuthClearCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear stored credentials",
		Long:  `Clear your stored BadgerMaps API credentials.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Clear stored API key
			if err := clearAPIKey(); err != nil {
				fmt.Println(color.RedString("Failed to clear API key: %v", err))
				os.Exit(1)
			}

			fmt.Println(color.GreenString("API key cleared"))
		},
	}

	return cmd
}

// promptForAPIKey prompts the user for their API key
func promptForAPIKey() string {
	fmt.Print("Enter your BadgerMaps API key: ")
	reader := bufio.NewReader(os.Stdin)
	apiKey, _ := reader.ReadString('\n')
	return strings.TrimSpace(apiKey)
}

// validateAPIKey validates the API key by making a test request
func validateAPIKey(apiKey string) error {
	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Test API connection
	return apiClient.TestAPIConnection()
}

// storeAPIKey stores the API key securely
func storeAPIKey(apiKey string, storeInEnv bool) error {
	// Always store in config.yaml file
	err := storeAPIKeyInConfig(apiKey)
	if err != nil {
		return err
	}

	// If storeInEnv is true, also store in .env file
	if storeInEnv {
		if err := storeAPIKeyInEnv(apiKey); err != nil {
			fmt.Println(color.YellowString("Warning: Failed to store API key in .env file: %v", err))
			// Continue anyway as we've already stored in config.yaml
		}
	}

	return nil
}

// storeAPIKeyInEnv stores the API key in the .env file
func storeAPIKeyInEnv(apiKey string) error {
	// Set in viper
	viper.Set("API_KEY", apiKey)

	// Check if .env file exists
	if !common.CheckEnvFileExists() {
		return fmt.Errorf("no .env file found. Use 'badgermaps utils create-env' to create one")
	}

	// Read the current .env file
	envContent, err := os.ReadFile(".env")
	if err != nil {
		return fmt.Errorf("failed to read .env file: %w", err)
	}

	// Replace or add the API_KEY line
	lines := strings.Split(string(envContent), "\n")
	apiKeyFound := false
	for i, line := range lines {
		if strings.HasPrefix(line, "API_KEY=") {
			lines[i] = "API_KEY=" + apiKey
			apiKeyFound = true
			break
		}
	}

	// If API_KEY line not found, add it
	if !apiKeyFound {
		lines = append(lines, "API_KEY="+apiKey)
	}

	// Write the updated content back to the .env file
	updatedContent := strings.Join(lines, "\n")
	if err := os.WriteFile(".env", []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write to .env file: %w", err)
	}

	fmt.Println("API key stored in environment file")
	return nil
}

// getStoredAPIKey gets the stored API key
func getStoredAPIKey() (string, error) {
	// Try to get from viper
	apiKey := viper.GetString("API_KEY")
	if apiKey != "" {
		return apiKey, nil
	}

	// Try to get from config file
	configFile := common.GetConfigFilePath()
	if _, err := os.Stat(configFile); err == nil {
		content, err := os.ReadFile(configFile)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "API_KEY:") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						apiKey = strings.TrimSpace(parts[1])
						if apiKey != "" {
							return apiKey, nil
						}
					}
				}
			}
		}
	}

	// Check ~/.badgermaps/config.yaml for backward compatibility
	if runtime.GOOS != "windows" {
		home, err := os.UserHomeDir()
		if err == nil {
			configFile = filepath.Join(home, ".badgermaps", "config.yaml")
			if _, err := os.Stat(configFile); err == nil {
				content, err := os.ReadFile(configFile)
				if err == nil {
					lines := strings.Split(string(content), "\n")
					for _, line := range lines {
						if strings.HasPrefix(line, "API_KEY:") {
							parts := strings.SplitN(line, ":", 2)
							if len(parts) == 2 {
								apiKey = strings.TrimSpace(parts[1])
								if apiKey != "" {
									return apiKey, nil
								}
							}
						}
					}
				}
			}
		}
	}

	// Try to get from .env file
	if common.CheckEnvFileExists() {
		content, err := os.ReadFile(".env")
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "API_KEY=") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						apiKey = strings.TrimSpace(parts[1])
						if apiKey != "" {
							return apiKey, nil
						}
					}
				}
			}
		}
	}

	// Not found
	return "", fmt.Errorf("no API key found")
}

// storeAPIKeyInConfig stores the API key in the config.yaml file
func storeAPIKeyInConfig(apiKey string) error {
	// Set in viper
	viper.Set("API_KEY", apiKey)

	// Ensure config directory exists
	if err := common.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Get config file path
	configFile := common.GetConfigFilePath()

	// Create a map for the config
	config := map[string]string{
		"API_KEY": apiKey,
	}

	// Convert to YAML format
	yamlData := "# BadgerMaps CLI Configuration\n"
	for key, value := range config {
		yamlData += fmt.Sprintf("%s: %s\n", key, value)
	}

	// Write to file
	if err := os.WriteFile(configFile, []byte(yamlData), 0644); err != nil {
		return fmt.Errorf("failed to write to config file: %w", err)
	}

	fmt.Println("API key stored in config file:", configFile)
	return nil
}

// clearAPIKey clears the stored API key
func clearAPIKey() error {
	// Clear from viper
	viper.Set("API_KEY", "")

	// Try to clear from config file
	configFile := common.GetConfigFilePath()
	if _, err := os.Stat(configFile); err == nil {
		// Config file exists, update it
		yamlData := "# BadgerMaps CLI Configuration\n"
		yamlData += "API_KEY: \n"
		os.WriteFile(configFile, []byte(yamlData), 0644)
		fmt.Println("API key cleared from config file")
	}

	// Also try to clear from .env file
	if common.CheckEnvFileExists() {
		// Read the current .env file
		envContent, err := os.ReadFile(".env")
		if err == nil {
			// Replace the API_KEY line
			lines := strings.Split(string(envContent), "\n")
			for i, line := range lines {
				if strings.HasPrefix(line, "API_KEY=") {
					lines[i] = "API_KEY="
					break
				}
			}
			// Write the updated content back to the .env file
			updatedContent := strings.Join(lines, "\n")
			os.WriteFile(".env", []byte(updatedContent), 0644)
			fmt.Println("API key cleared from environment file")
		}
	}

	return nil
}

// readPassword reads a password from the terminal
// Note: In a real implementation, we would use golang.org/x/term.ReadPassword
// to read the password without echoing it to the terminal
func readPassword() (string, error) {
	fmt.Print("Enter your password: ")
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(password), nil
}
