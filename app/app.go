package app

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

type Application struct {
	Verbose bool
	Quiet   bool
	Debug   bool
	NoColor bool
	CfgFile string

	Config *Config
}

func NewApplication() *Application {
	return &Application{
		Verbose: false,
		Quiet:   false,
		Debug:   false,
	}
}

// VerifySetupOrExit checks if setup is complete and prompts the user to run setup if not
// If the user declines, the program exits
// Returns true if setup is complete or the user wants to run setup, false otherwise
func (a *Application) VerifySetupOrExit() bool {
	_, ok := GetConfigFilePath()
	if ok {
		return true
	}

	if promptForSetup() {
		if InteractiveSetup(a) {
			return true
		}
	}

	fmt.Println(color.YellowString("Setup is required to use this command. Exiting."))
	os.Exit(0)
	return false
}
