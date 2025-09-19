package gui

import "fyne.io/fyne/v2"

// GuiView defines the interface for the GUI, abstracting the UI implementation
// from the presenter. This allows the presenter to be tested independently of the UI.
type GuiView interface {
	// ShowToast displays a short, transient message.
	ShowToast(message string)
	// ShowProgressBar displays a progress bar with a title.
	ShowProgressBar(title string)
	// HideProgressBar hides the progress bar.
	HideProgressBar()
	// SetProgress updates the value of the progress bar.
	SetProgress(value float64)
	// ShowErrorDialog shows a modal dialog with an error message.
	ShowErrorDialog(err error)
	// ShowConfirmDialog shows a confirmation dialog with a callback.
	ShowConfirmDialog(title, message string, callback func(bool))

	// RefreshHomeTab rebuilds and refreshes the home tab content.
	RefreshHomeTab()
	// RefreshConfigTab rebuilds and refreshes the configuration tab content.
	RefreshConfigTab()
	// RefreshPushTab rebuilds and refreshes the push tab content.
	RefreshPushTab()
	// RefreshAllTabs rebuilds and refreshes all main tabs, useful after major state changes like connection status.
	RefreshAllTabs()

	// ShowDetails displays detailed information in the right-hand pane.
	ShowDetails(details fyne.CanvasObject)
	// GetMainWindow returns the main application window, needed for dialogs.
	GetMainWindow() fyne.Window
}
