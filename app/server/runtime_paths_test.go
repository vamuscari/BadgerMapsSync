package server

import (
	"badgermaps/app/state"
	"badgermaps/utils"
	"path/filepath"
	"reflect"
	"testing"
)

func TestServerCommandArgsIncludesConfigPathWhenAvailable(t *testing.T) {
	s := state.NewState()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	*s.ConfigFile = "  " + configPath + "  "

	got := serverCommandArgs(s)
	want := []string{"server", "--config", configPath}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected args %v, got %v", want, got)
	}
}

func TestPIDFilePathFallsBackWhenUnset(t *testing.T) {
	s := state.NewState()

	got := pidFilePath(s)
	if got != ".badgermaps.pid" {
		t.Fatalf("expected fallback PID file path, got %q", got)
	}
}

func TestSchedulerConfigDirUsesConfigDirOrUserDefault(t *testing.T) {
	s := state.NewState()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	*s.ConfigFile = configPath

	if got, want := schedulerConfigDir(s), filepath.Dir(configPath); got != want {
		t.Fatalf("expected scheduler config dir %q, got %q", want, got)
	}

	*s.ConfigFile = ""
	if got, want := schedulerConfigDir(s), utils.GetUserDefaultConfigDir(); got != want {
		t.Fatalf("expected fallback scheduler config dir %q, got %q", want, got)
	}
}
