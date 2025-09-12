package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
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

// WriteInterfaceToYAMLFile marshals the given interface into YAML and writes it to the specified file.
func WriteInterfaceToYAMLFile(filePath string, data interface{}) error {
	// Marshal the data into YAML format
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to YAML: %w", err)
	}

	// Write the YAML data to the file
	err = os.WriteFile(filePath, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write YAML to file %s: %w", filePath, err)
	}

	return nil
}

// WriteLines writes a slice of strings to a file, one per line.
func WriteLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

// WriteEnvFile writes a map of key-value pairs to a .env file.
func WriteEnvFile(path string, data map[string]string) error {
	var lines []string
	for key, value := range data {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}
	return WriteLines(lines, path)
}
