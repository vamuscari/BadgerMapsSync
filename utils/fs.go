package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetUserDefaultConfigDir returns the appropriate directory for user configuration files
// based on the current operating system:
// - Linux/Darwin: ~/.config/badgermaps/
// - Windows: %LocalAppData%\badgermaps\
func GetUserDefaultConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		// On Windows, use %LocalAppData%\badgermaps\
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			// Fallback if LOCALAPPDATA is not set
			home := os.Getenv("USERPROFILE")
			if home != "" {
				localAppData = filepath.Join(home, "AppData", "Local")
			}
		}
		return filepath.Join(localAppData, "badgermaps")
	default:
		// On Linux/Darwin, use ~/.config/badgermaps/
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if home can't be determined
			return "."
		}
		return filepath.Join(home, ".config", "badgermaps")
	}
}

// GetConfigDirFile returns the path to a file in the configuration directory
func GetConfigDirFile(filename string) string {
	return filepath.Join(GetUserDefaultConfigDir(), filename)
}

// CheckIfDirExist checks if a directory exists
func CheckIfDirExist(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// CheckIfFileExists checks if a file exists
func CheckIfFileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func EnsureFileExists(filename string) error {
	err := EnsureDirExists(filepath.Dir(filename))
	if err != nil {
		return err
	}
	if !CheckIfFileExists(filename) {
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		err = file.Chmod(660)
		if err != nil {
			return err
		}
	}

	return nil
}

// EnsureDirExists ensures a directory exists, creating it if necessary
func EnsureDirExists(path string) error {
	if CheckIfDirExist(path) {
		return nil
	}
	return os.MkdirAll(path, 0755)
}
