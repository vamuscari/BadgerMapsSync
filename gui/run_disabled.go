//go:build nogui

package gui

import (
	"badgermaps/app"
	"fyne.io/fyne/v2"
)

const Enabled = false

// Run is a stub function for when the GUI is disabled.
func Run(a *app.App, icon fyne.Resource) {
	// No GUI support
}
