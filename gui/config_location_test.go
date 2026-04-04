package gui

import (
	"badgermaps/app"
	"path/filepath"
	"testing"
)

func TestDetectConfigSaveLocation(t *testing.T) {
	t.Run("user config", func(t *testing.T) {
		a := app.NewApp()
		userPath, err := userConfigFilePath()
		if err != nil {
			t.Fatalf("failed to resolve user path: %v", err)
		}
		a.SetConfigFilePath(userPath)

		want := configSaveLocationLabel(configSaveLocationUserID)
		if got := detectConfigSaveLocation(a); got != want {
			t.Fatalf("expected user location %q, got %q", want, got)
		}
	})

	t.Run("global config", func(t *testing.T) {
		a := app.NewApp()
		globalPath, err := globalConfigFilePath()
		if err != nil {
			t.Fatalf("failed to resolve global path: %v", err)
		}
		a.SetConfigFilePath(globalPath)

		want := configSaveLocationLabel(configSaveLocationGlobalID)
		if got := detectConfigSaveLocation(a); got != want {
			t.Fatalf("expected global location %q, got %q", want, got)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		a := app.NewApp()
		a.SetConfigFilePath(filepath.Join(t.TempDir(), "custom-config.yaml"))

		if got := detectConfigSaveLocation(a); got != "" {
			t.Fatalf("expected empty location for custom path, got %q", got)
		}
	})
}

func TestEnsureConfigSavePath(t *testing.T) {
	t.Run("preserves current when location unset", func(t *testing.T) {
		a := app.NewApp()
		customPath := filepath.Join(t.TempDir(), "existing-config.yaml")
		a.SetConfigFilePath(customPath)

		if err := ensureConfigSavePath(a, ""); err != nil {
			t.Fatalf("expected no error preserving path: %v", err)
		}
		if a.ConfigFile != customPath {
			t.Fatalf("expected config path %q, got %q", customPath, a.ConfigFile)
		}
	})

	t.Run("sets user path", func(t *testing.T) {
		a := app.NewApp()
		if err := ensureConfigSavePath(a, configSaveLocationUserID); err != nil {
			t.Fatalf("expected no error setting user path: %v", err)
		}
		want, err := userConfigFilePath()
		if err != nil {
			t.Fatalf("failed to resolve user path: %v", err)
		}
		if a.ConfigFile != want {
			t.Fatalf("expected config path %q, got %q", want, a.ConfigFile)
		}
	})

	t.Run("sets global path", func(t *testing.T) {
		a := app.NewApp()
		if err := ensureConfigSavePath(a, configSaveLocationGlobalID); err != nil {
			t.Fatalf("expected no error setting global path: %v", err)
		}
		want, err := globalConfigFilePath()
		if err != nil {
			t.Fatalf("failed to resolve global path: %v", err)
		}
		if a.ConfigFile != want {
			t.Fatalf("expected config path %q, got %q", want, a.ConfigFile)
		}
	})
}
