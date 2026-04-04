package gui

import (
	"badgermaps/app"
	"badgermaps/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configSaveLocationUserID   = "user"
	configSaveLocationGlobalID = "global"

	legacyConfigSaveLocationLocalID = "local"

	legacyConfigSaveLocationLocalLabel  = "Local (./config.yaml)"
	legacyConfigSaveLocationGlobalLabel = "Global (~/.config/badgermaps/config.yaml)"
)

func configSaveLocationOptions() []string {
	return []string{
		configSaveLocationLabel(configSaveLocationUserID),
		configSaveLocationLabel(configSaveLocationGlobalID),
	}
}

func configSaveLocationPath(location string) (string, error) {
	switch normalizeConfigSaveLocation(location) {
	case configSaveLocationUserID:
		return userConfigFilePath()
	case configSaveLocationGlobalID:
		return globalConfigFilePath()
	default:
		return "", fmt.Errorf("unknown config save location: %q", location)
	}
}

func normalizeConfigSaveLocation(location string) string {
	trimmed := strings.TrimSpace(location)
	if trimmed == "" {
		return ""
	}

	switch trimmed {
	case configSaveLocationUserID:
		return configSaveLocationUserID
	case configSaveLocationGlobalID:
		return configSaveLocationGlobalID
	case legacyConfigSaveLocationLocalID, legacyConfigSaveLocationLocalLabel:
		// Backward compatibility: prior "local" selection wrote next to the executable.
		return configSaveLocationGlobalID
	case legacyConfigSaveLocationGlobalLabel:
		// Backward compatibility: prior "global" selection targeted the user config directory.
		return configSaveLocationUserID
	}

	userLabel := configSaveLocationLabel(configSaveLocationUserID)
	globalLabel := configSaveLocationLabel(configSaveLocationGlobalID)
	if trimmed == userLabel || strings.HasPrefix(trimmed, "User Config (") {
		return configSaveLocationUserID
	}
	if trimmed == globalLabel || strings.HasPrefix(trimmed, "Global (") {
		return configSaveLocationGlobalID
	}
	if strings.HasPrefix(trimmed, "Local (") {
		return configSaveLocationGlobalID
	}

	return ""
}

func configSaveLocationLabel(locationID string) string {
	switch locationID {
	case configSaveLocationUserID:
		path, err := userConfigFilePath()
		if err != nil {
			return "User Config (user config directory)"
		}
		return fmt.Sprintf("User Config (%s)", path)
	case configSaveLocationGlobalID:
		path, err := globalConfigFilePath()
		if err != nil {
			return "Global (config.yaml beside executable)"
		}
		return fmt.Sprintf("Global (%s)", path)
	default:
		return ""
	}
}

func globalConfigFilePath() (string, error) {
	executablePath, err := os.Executable()
	if err != nil || strings.TrimSpace(executablePath) == "" {
		return filepath.Abs("config.yaml")
	}
	return filepath.Abs(filepath.Join(filepath.Dir(executablePath), "config.yaml"))
}

func userConfigFilePath() (string, error) {
	return filepath.Abs(utils.GetConfigDirFile("config.yaml"))
}

func currentConfigPath(a *app.App) string {
	if a == nil {
		return ""
	}

	if current := strings.TrimSpace(a.ConfigFile); current != "" {
		return current
	}

	if path, ok, err := a.GetConfigFilePath(); err == nil && ok {
		return strings.TrimSpace(path)
	}

	return ""
}

func detectConfigSaveLocation(a *app.App) string {
	currentPath := currentConfigPath(a)

	if currentPath == "" {
		return configSaveLocationLabel(configSaveLocationUserID)
	}

	currentAbs, err := filepath.Abs(currentPath)
	if err != nil {
		return configSaveLocationLabel(configSaveLocationUserID)
	}
	userAbs, err := userConfigFilePath()
	if err == nil && samePath(currentAbs, userAbs) {
		return configSaveLocationLabel(configSaveLocationUserID)
	}
	globalAbs, err := globalConfigFilePath()
	if err == nil && samePath(currentAbs, globalAbs) {
		return configSaveLocationLabel(configSaveLocationGlobalID)
	}

	return ""
}

func ensureConfigSavePath(a *app.App, location string) error {
	if a == nil {
		return fmt.Errorf("application context is unavailable")
	}

	trimmedLocation := strings.TrimSpace(location)
	if trimmedLocation == "" {
		if strings.TrimSpace(a.ConfigFile) != "" {
			return nil
		}
		trimmedLocation = detectConfigSaveLocation(a)
	}

	targetPath, err := configSaveLocationPath(trimmedLocation)
	if err != nil {
		return err
	}
	a.SetConfigFilePath(targetPath)
	return nil
}

func samePath(left, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
