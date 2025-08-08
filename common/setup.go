package common

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// IsSetupComplete checks if the CLI has been set up by verifying if the config file exists
func IsSetupComplete() bool {
	configFile := GetConfigFilePath()
	_, err := os.Stat(configFile)
	return err == nil
}

// PromptForSetup asks the user if they want to run setup
// Returns true if the user wants to run setup, false otherwise
func PromptForSetup() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(color.YellowString("BadgerMaps CLI is not set up. Would you like to run setup now? [y/N]: "))
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// VerifySetupOrExit checks if setup is complete and prompts the user to run setup if not
// If the user declines, the program exits
// Returns true if setup is complete or the user wants to run setup, false otherwise
func VerifySetupOrExit() bool {
	if IsSetupComplete() || CheckEnvFileExists() {
		return true
	}

	if PromptForSetup() {
		return true
	}

	fmt.Println(color.YellowString("Setup is required to use this command. Exiting."))
	os.Exit(0)
	return false
}
