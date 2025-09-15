//go:build gui

package main

import (
	"badgermaps/gui"
)

const hasGUI = true

func runGUI() {
	// For the GUI, we need to ensure the basic configuration is loaded
	// so the app can function. We can trigger the same logic Cobra uses.
	App.EnsureConfig()

	// Launch the Fyne GUI
	gui.Launch(App, AppIcon)
}
