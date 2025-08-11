package common

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

// GetConfigDir returns the appropriate directory for configuration files
// based on the current operating system:
// - Linux/Darwin: ~/.config/badgermaps/
// - Windows: %LocalAppData%\badgermaps\
func GetConfigDir() string {
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

// GetCacheDir returns the appropriate directory for cache files
// based on the current operating system:
// - Linux/Darwin: ~/.cache/badgermaps/
// - Windows: %TEMP%\badgermaps\
// This can be overridden by setting the CACHE_DIR configuration value.
func GetCacheDir() string {
	// Check if a custom cache directory is specified in the configuration
	if customCacheDir := viper.GetString("CACHE_DIR"); customCacheDir != "" {
		return customCacheDir
	}

	// Otherwise, use OS-specific defaults
	switch runtime.GOOS {
	case "windows":
		// On Windows, use %TEMP%\badgermaps\
		temp := os.Getenv("TEMP")
		if temp == "" {
			// Fallback if TEMP is not set
			temp = os.Getenv("TMP")
			if temp == "" {
				// Further fallback
				home := os.Getenv("USERPROFILE")
				if home != "" {
					temp = filepath.Join(home, "AppData", "Local", "Temp")
				} else {
					// Last resort
					return filepath.Join(".", "temp")
				}
			}
		}
		return filepath.Join(temp, "badgermaps")
	default:
		// On Linux/Darwin, use ~/.cache/badgermaps/
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if home can't be determined
			return "."
		}
		return filepath.Join(home, ".cache", "badgermaps")
	}
}

// EnsureConfigDir ensures the configuration directory exists
func EnsureConfigDir() error {
	configDir := GetConfigDir()
	return os.MkdirAll(configDir, 0755)
}

// EnsureCacheDir ensures the cache directory exists
func EnsureCacheDir() error {
	cacheDir := GetCacheDir()
	return os.MkdirAll(cacheDir, 0755)
}

// GetConfigFilePath returns the full path to the config file
func GetConfigFilePath() string {
	return filepath.Join(GetConfigDir(), "config.yaml")
}

// GetCacheFilePath returns the full path to a cache file with the given name
func GetCacheFilePath(filename string) string {
	return filepath.Join(GetCacheDir(), filename)
}
