package server

import (
	"badgermaps/app/state"
	"badgermaps/utils"
	"path/filepath"
	"strings"
)

func resolveConfigPathFromState(s *state.State) string {
	if s == nil || s.ConfigFile == nil {
		return ""
	}
	return strings.TrimSpace(*s.ConfigFile)
}

func serverCommandArgs(s *state.State) []string {
	args := []string{"server"}
	if configPath := resolveConfigPathFromState(s); configPath != "" {
		args = append(args, "--config", configPath)
	}
	return args
}

func pidFilePath(s *state.State) string {
	if s != nil {
		if p := strings.TrimSpace(s.PIDFile); p != "" {
			return p
		}
	}
	return ".badgermaps.pid"
}

func schedulerConfigDir(s *state.State) string {
	if configPath := resolveConfigPathFromState(s); configPath != "" {
		return filepath.Dir(strings.TrimSpace(configPath))
	}
	return utils.GetUserDefaultConfigDir()
}

func runtimeActivityPath(s *state.State) string {
	return filepath.Join(schedulerConfigDir(s), "server_activity.json")
}
