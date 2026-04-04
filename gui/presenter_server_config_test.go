package gui

import (
	"badgermaps/app"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

type testGuiView struct {
	toasts []string
}

func (v *testGuiView) ShowToast(message string)     { v.toasts = append(v.toasts, message) }
func (v *testGuiView) ShowProgressBar(title string) {}
func (v *testGuiView) HideProgressBar()             {}
func (v *testGuiView) SetProgress(value float64)    {}
func (v *testGuiView) ShowErrorDialog(err error)    {}
func (v *testGuiView) ShowConfirmDialog(title, message string, callback func(bool)) {
}
func (v *testGuiView) RefreshHomeTab()                       {}
func (v *testGuiView) RefreshConfigTab()                     {}
func (v *testGuiView) RefreshPushTab()                       {}
func (v *testGuiView) RefreshAllTabs()                       {}
func (v *testGuiView) ApplyThemePreference(pref string)      {}
func (v *testGuiView) ShowDetails(details fyne.CanvasObject) {}
func (v *testGuiView) GetMainWindow() fyne.Window            { return nil }

func TestHandleSaveServerConfigSavesValidTimezone(t *testing.T) {
	a := app.NewApp()
	a.SetConfigFilePath(filepath.Join(t.TempDir(), "config.yaml"))
	view := &testGuiView{}
	p := NewGuiPresenter(a, view)

	p.HandleSaveServerConfig(
		"localhost",
		"8080",
		"America/New_York",
		false,
		"",
		"",
		"webhook-secret",
		"internal-token",
		true,
	)

	if got, want := a.Config.Server.Timezone, "America/New_York"; got != want {
		t.Fatalf("expected config timezone %q, got %q", want, got)
	}
	if got, want := a.State.ServerTimezone, "America/New_York"; got != want {
		t.Fatalf("expected state timezone %q, got %q", want, got)
	}
	if len(view.toasts) == 0 {
		t.Fatalf("expected success toast")
	}
	if !strings.HasPrefix(view.toasts[len(view.toasts)-1], "Success:") {
		t.Fatalf("expected success toast, got %q", view.toasts[len(view.toasts)-1])
	}
}

func TestHandleSaveServerConfigRejectsInvalidTimezone(t *testing.T) {
	a := app.NewApp()
	a.SetConfigFilePath(filepath.Join(t.TempDir(), "config.yaml"))
	a.Config.Server.Timezone = "UTC"
	a.State.ServerTimezone = "UTC"

	view := &testGuiView{}
	p := NewGuiPresenter(a, view)

	p.HandleSaveServerConfig(
		"localhost",
		"8080",
		"Mars/Olympus",
		false,
		"",
		"",
		"webhook-secret",
		"internal-token",
		true,
	)

	if got, want := a.Config.Server.Timezone, "UTC"; got != want {
		t.Fatalf("expected config timezone to remain %q, got %q", want, got)
	}
	if got, want := a.State.ServerTimezone, "UTC"; got != want {
		t.Fatalf("expected state timezone to remain %q, got %q", want, got)
	}
	if len(view.toasts) == 0 {
		t.Fatalf("expected error toast")
	}
	lastToast := view.toasts[len(view.toasts)-1]
	if !strings.HasPrefix(lastToast, "Error: Invalid timezone") {
		t.Fatalf("expected invalid timezone toast, got %q", lastToast)
	}
}
