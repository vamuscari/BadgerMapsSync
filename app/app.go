package app

import (
	"badgermaps/database"
	"fmt"
	"os"

	"github.com/fatih/color"
)

type State struct {
	Verbose bool
	Quiet   bool
	Debug   bool
	NoColor bool
	CfgFile string

	Config *Config
	DB     database.DB
}

func NewApplication() *State {
	return &State{
		Verbose: false,
		Quiet:   false,
		Debug:   false,
	}
}

// VerifySetupOrExit checks if setup is complete and prompts the user to run setup if not
// If the user declines, the program exits
// Returns true if setup is complete or the user wants to run setup, false otherwise
func (a *State) VerifySetupOrExit() bool {
	// Respect NoColor setting as early as possible
	if a.NoColor {
		color.NoColor = true
	}

	path, ok := GetConfigFilePath()
	if ok {
		if a.Verbose || a.Debug {
			fmt.Println(color.GreenString("Configuration detected: %s", path))
		}
		// Attempt to load configuration if not already loaded
		if a.Config == nil {
			a.Config = LoadConfig(a)
			if (a.Verbose || a.Debug) && a.Config != nil {
				apiKeyStatus := "not set"
				if a.Config.APIKey != "" {
					apiKeyStatus = "set"
				}
				dbType := a.Config.DBType
				if a.DB != nil && dbType == "" {
					dbType = a.DB.GetType()
				}
				fmt.Println(color.CyanString("Setup OK: DB_TYPE=%s, API_KEY=%s", dbType, apiKeyStatus))
			}
		}
		return true
	}

	if a.Verbose || a.Debug {
		fmt.Println(color.YellowString("No configuration file found (.env or config.yaml)."))
	}

	if promptForSetup() {
		if a.Verbose || a.Debug {
			fmt.Println(color.CyanString("Starting interactive setup..."))
		}
		if InteractiveSetup(a) {
			if a.Verbose || a.Debug {
				fmt.Println(color.GreenString("Setup completed successfully."))
			}
			return true
		}
		if a.Verbose || a.Debug {
			fmt.Println(color.RedString("Setup did not complete successfully."))
		}
	}

	fmt.Println(color.YellowString("Setup is required to use this command. Exiting."))
	os.Exit(0)
	return false
}
