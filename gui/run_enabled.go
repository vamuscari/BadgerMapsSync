//go:build !nogui

package gui

import (
	"badgermaps/app"
	"fyne.io/fyne/v2"
)

const Enabled = true

// Run launches the Fyne GUI.
// It ensures the application configuration is loaded before starting the UI.
func Run(a *app.App, icon fyne.Resource) {
	a.EnsureConfig(true)
	Launch(a, icon)
}
