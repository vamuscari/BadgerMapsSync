package gui

import (
	"badgermaps/app"
	"badgermaps/database"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	fapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Gui struct holds all the UI components and application state
type Gui struct {
	app     *app.App
	fyneApp fyne.App
	window  fyne.Window

	logMutex        sync.Mutex
	logData         []string
	logView         *widget.List
	configTab       fyne.CanvasObject
	terminalVisible bool
	tabs            *container.AppTabs // Hold a reference to the tabs container
}

// Launch initializes and runs the GUI
func Launch(a *app.App, icon fyne.Resource) {
	fyneApp := fapp.New()
	fyneApp.SetIcon(icon)
	window := fyneApp.NewWindow("BadgerMaps CLI")

	ui := &Gui{
		app:             a,
		fyneApp:         fyneApp,
		window:          window,
		logData:         []string{"Welcome to BadgerMaps CLI GUI!"},
		terminalVisible: true,
	}

	window.SetContent(ui.createContent())
	window.Resize(fyne.NewSize(800, 600))
	window.ShowAndRun()
}

// createContent builds the main content of the window
func (ui *Gui) createContent() fyne.CanvasObject {
	return ui.createMainContent()
}

// createMainContent builds the main layout with toolbar, tabs, and log view
func (ui *Gui) createMainContent() fyne.CanvasObject {
	ui.configTab = ui.buildConfigTab()
	ui.tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Pull", theme.DownloadIcon(), ui.createPullTab()),
		container.NewTabItemWithIcon("Push", theme.UploadIcon(), ui.createPushTab()),
		container.NewTabItemWithIcon("Configuration", theme.SettingsIcon(), ui.createConfigTab()),
	)

	toolbar := widget.NewToolbar(
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.ComputerIcon(), func() {
			ui.terminalVisible = !ui.terminalVisible
			ui.window.SetContent(ui.createMainContent())
		}),
	)

	mainContent := container.NewBorder(
		toolbar, nil, nil, nil, ui.tabs,
	)

	if ui.terminalVisible {
		ui.logView = widget.NewList(
			func() int {
				return len(ui.logData)
			},
			func() fyne.CanvasObject {
				return widget.NewLabel("template")
			},
			func(i widget.ListItemID, o fyne.CanvasObject) {
				o.(*widget.Label).SetText(ui.logData[i])
			},
		)
		split := container.NewHSplit(mainContent, ui.logView)
		split.Offset = 0.7
		return split
	}

	return mainContent
}

// createPullTab creates the content for the "Pull" tab
func (ui *Gui) createPullTab() fyne.CanvasObject {
	accountIDEntry := widget.NewEntry()
	accountIDEntry.SetPlaceHolder("Account ID")
	pullAccountButton := widget.NewButtonWithIcon("Pull Account", theme.DownloadIcon(), func() {
		go ui.runPullAccount(accountIDEntry.Text)
	})

	checkinIDEntry := widget.NewEntry()
	checkinIDEntry.SetPlaceHolder("Check-in ID")
	pullCheckinButton := widget.NewButtonWithIcon("Pull Check-in", theme.DownloadIcon(), func() {
		go ui.runPullCheckin(checkinIDEntry.Text)
	})

	routeIDEntry := widget.NewEntry()
	routeIDEntry.SetPlaceHolder("Route ID")
	pullRouteButton := widget.NewButtonWithIcon("Pull Route", theme.DownloadIcon(), func() {
		go ui.runPullRoute(routeIDEntry.Text)
	})

	pullAccountsButton := widget.NewButtonWithIcon("Pull All Accounts", theme.DownloadIcon(), func() { go ui.runPullAccounts() })
	pullCheckinsButton := widget.NewButtonWithIcon("Pull All Check-ins", theme.DownloadIcon(), func() { go ui.runPullCheckins() })
	pullRoutesButton := widget.NewButtonWithIcon("Pull All Routes", theme.DownloadIcon(), func() { go ui.runPullRoutes() })
	pullProfileButton := widget.NewButtonWithIcon("Pull User Profile", theme.AccountIcon(), func() { go ui.runPullProfile() })
	pullAllButton := widget.NewButtonWithIcon("Run Full Pull (All Data)", theme.ViewRefreshIcon(), func() { go ui.runPullAll() })

	return container.NewVBox(
		widget.NewLabelWithStyle("Pull Single Item by ID", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2, accountIDEntry, pullAccountButton),
		container.NewGridWithColumns(2, checkinIDEntry, pullCheckinButton),
		container.NewGridWithColumns(2, routeIDEntry, pullRouteButton),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Pull Data Sets", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		pullAccountsButton,
		pullCheckinsButton,
		pullRoutesButton,
		pullProfileButton,
		widget.NewSeparator(),
		pullAllButton,
	)
}

// createPushTab creates the content for the "Push" tab
func (ui *Gui) createPushTab() fyne.CanvasObject {
	pushAccountsButton := widget.NewButtonWithIcon("Push Account Changes", theme.UploadIcon(), func() { go ui.runPushAccounts() })
	pushCheckinsButton := widget.NewButtonWithIcon("Push Check-in Changes", theme.UploadIcon(), func() { go ui.runPushCheckins() })
	pushAllButton := widget.NewButtonWithIcon("Push All Changes", theme.ViewRefreshIcon(), func() { go ui.runPushAll() })

	tableContainer := container.NewVBox()
	entityType := "accounts" // Default view

	radio := widget.NewRadioGroup([]string{"accounts", "checkins"}, func(selected string) {
		entityType = selected
		tableContainer.Objects = []fyne.CanvasObject{ui.createPendingChangesTable(entityType)}
		tableContainer.Refresh()
	})
	radio.SetSelected("accounts")

	tableContainer.Objects = []fyne.CanvasObject{ui.createPendingChangesTable(entityType)}

	return container.NewVBox(
		widget.NewLabelWithStyle("Push Pending Changes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		pushAccountsButton,
		pushCheckinsButton,
		widget.NewSeparator(),
		pushAllButton,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("View Pending Changes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		radio,
		tableContainer,
	)
}

func (ui *Gui) createPendingChangesTable(entityType string) fyne.CanvasObject {
	options := app.PushFilterOptions{
		Status:  "pending",
		OrderBy: "date_desc",
	}

	results, err := app.GetFilteredPendingChanges(ui.app, entityType, options)
	if err != nil {
		return widget.NewLabel(fmt.Sprintf("Error fetching changes: %v", err))
	}

	var headers []string
	var data [][]string

	switch entityType {
	case "accounts":
		headers = []string{"ID", "Account ID", "Type", "Status", "Created At", "Changes"}
		changes, ok := results.([]database.AccountPendingChange)
		if !ok {
			return widget.NewLabel("Error: Could not load account changes.")
		}
		for _, c := range changes {
			data = append(data, []string{
				fmt.Sprintf("%d", c.ChangeId),
				fmt.Sprintf("%d", c.AccountId),
				c.ChangeType,
				c.Status,
				c.CreatedAt.Format(time.RFC3339),
				c.Changes,
			})
		}
	case "checkins":
		headers = []string{"ID", "Checkin ID", "Account ID", "Type", "Status", "Created At", "Changes"}
		changes, ok := results.([]database.CheckinPendingChange)
		if !ok {
			return widget.NewLabel("Error: Could not load check-in changes.")
		}
		for _, c := range changes {
			data = append(data, []string{
				fmt.Sprintf("%d", c.ChangeId),
				fmt.Sprintf("%d", c.CheckinId),
				fmt.Sprintf("%d", c.AccountId),
				c.ChangeType,
				c.Status,
				c.CreatedAt.Format(time.RFC3339),
				c.Changes,
			})
		}
	}

	if len(data) == 0 {
		return widget.NewLabel(fmt.Sprintf("No pending %s changes found.", entityType))
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(data) + 1, len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i.Row == 0 {
				label.SetText(headers[i.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				label.SetText(data[i.Row-1][i.Col])
				label.TextStyle = fyne.TextStyle{}
			}
		},
	)

	return table
}

// createConfigTab returns the config tab content
func (ui *Gui) createConfigTab() fyne.CanvasObject {
	return ui.configTab
}

// buildConfigTab builds the configuration tab UI
func (ui *Gui) buildConfigTab() fyne.CanvasObject {
	// API Settings
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetText(ui.app.API.APIKey)
	baseURLEntry := widget.NewEntry()
	baseURLEntry.SetText(ui.app.API.BaseURL)
	apiForm := widget.NewForm(
		widget.NewFormItem("API Key", apiKeyEntry),
		widget.NewFormItem("Base URL", baseURLEntry),
	)

	// Database Settings
	dbPathEntry := widget.NewEntry()
	dbHostEntry := widget.NewEntry()
	dbPortEntry := widget.NewEntry()
	dbUserEntry := widget.NewEntry()
	dbPassEntry := widget.NewPasswordEntry()
	dbNameEntry := widget.NewEntry()

	dbPathFormItem := widget.NewFormItem("Path", dbPathEntry)
	dbHostFormItem := widget.NewFormItem("Host", dbHostEntry)
	dbPortFormItem := widget.NewFormItem("Port", dbPortEntry)
	dbUserFormItem := widget.NewFormItem("User", dbUserEntry)
	dbPassFormItem := widget.NewFormItem("Password", dbPassEntry)
	dbNameFormItem := widget.NewFormItem("Database Name", dbNameEntry)

	dbForm := widget.NewForm()
	dbTypeSelect := widget.NewSelect([]string{"sqlite3", "postgres", "mssql"}, func(selected string) {
		dbForm.Items = []*widget.FormItem{}
		if selected == "sqlite3" {
			dbForm.AppendItem(dbPathFormItem)
		} else {
			dbForm.AppendItem(dbHostFormItem)
			dbForm.AppendItem(dbPortFormItem)
			dbForm.AppendItem(dbUserFormItem)
			dbForm.AppendItem(dbPassFormItem)
			dbForm.AppendItem(dbNameFormItem)
		}
		dbForm.Refresh()
	})

	// Populate form with current config
	switch config := ui.app.DB.(type) {
	case *database.SQLiteConfig:
		dbPathEntry.SetText(config.Path)
	case *database.PostgreSQLConfig:
		dbHostEntry.SetText(config.Host)
		dbPortEntry.SetText(fmt.Sprintf("%d", config.Port))
		dbUserEntry.SetText(config.Username)
		dbPassEntry.SetText(config.Password)
		dbNameEntry.SetText(config.Database)
	case *database.MSSQLConfig:
		dbHostEntry.SetText(config.Host)
		dbPortEntry.SetText(fmt.Sprintf("%d", config.Port))
		dbUserEntry.SetText(config.Username)
		dbPassEntry.SetText(config.Password)
		dbNameEntry.SetText(config.Database)
	}
	dbTypeSelect.SetSelected(ui.app.DB.GetType())

	// Other Settings
	verboseCheck := widget.NewCheck("Verbose Logging", func(b bool) { ui.app.State.Verbose = b })
	verboseCheck.SetChecked(ui.app.State.Verbose)

	// Buttons
	testButton := widget.NewButtonWithIcon("Test Connection", theme.HelpIcon(), nil)
	testButton.OnTapped = func() {
		go ui.testDBConnection(
			testButton, dbTypeSelect.Selected, dbPathEntry.Text, dbHostEntry.Text,
			dbPortEntry.Text, dbUserEntry.Text, dbPassEntry.Text, dbNameEntry.Text,
		)
	}

	saveButton := widget.NewButtonWithIcon("Save Configuration", theme.ConfirmIcon(), func() {
		ui.saveConfig(
			apiKeyEntry.Text, baseURLEntry.Text, dbTypeSelect.Selected, dbPathEntry.Text,
			dbHostEntry.Text, dbPortEntry.Text, dbUserEntry.Text, dbPassEntry.Text, dbNameEntry.Text,
		)
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("API Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		apiForm,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Database Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2, widget.NewLabel("Database Type"), dbTypeSelect),
		dbForm,
		widget.NewSeparator(),
		verboseCheck,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, testButton, saveButton),
	)
}

// log adds a message to the log view
func (ui *Gui) log(message string) {
	ui.logMutex.Lock()
	defer ui.logMutex.Unlock()
	lines := strings.Split(message, "\n")
	ui.logData = append(ui.logData, lines...)
	if ui.logView != nil {
		ui.logView.Refresh()
		ui.logView.ScrollToBottom()
	}
}

// --- Pull Functions ---
func (ui *Gui) runPullAll() {
	ui.log("Starting full data pull...")
	if err := app.RunPullAll(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
	}
	ui.log("Pull process finished.")
}

func (ui *Gui) runPullAccount(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.log(fmt.Sprintf("Invalid Account ID: '%s'", idStr))
		return
	}
	ui.log(fmt.Sprintf("Starting pull for account ID: %d...", id))
	if err := app.RunPullAccount(ui.app, id, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
	}
	ui.log("Pull process finished.")
}

func (ui *Gui) runPullAccounts() {
	ui.log("Starting pull for all accounts...")
	if err := app.RunPullAccounts(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
	}
	ui.log("Pull process finished.")
}

func (ui *Gui) runPullCheckin(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.log(fmt.Sprintf("Invalid Check-in ID: '%s'", idStr))
		return
	}
	ui.log(fmt.Sprintf("Starting pull for check-in ID: %d...", id))
	if err := app.RunPullCheckin(ui.app, id, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
	}
	ui.log("Pull process finished.")
}

func (ui *Gui) runPullCheckins() {
	ui.log("Starting pull for all check-ins...")
	if err := app.RunPullCheckins(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
	}
	ui.log("Pull process finished.")
}

func (ui *Gui) runPullRoute(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.log(fmt.Sprintf("Invalid Route ID: '%s'", idStr))
		return
	}
	ui.log(fmt.Sprintf("Starting pull for route ID: %d...", id))
	if err := app.RunPullRoute(ui.app, id, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
	}
	ui.log("Pull process finished.")
}

func (ui *Gui) runPullRoutes() {
	ui.log("Starting pull for all routes...")
	if err := app.RunPullRoutes(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
	}
	ui.log("Pull process finished.")
}

func (ui *Gui) runPullProfile() {
	ui.log("Starting pull for user profile...")
	if err := app.RunPullProfile(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
	}
	ui.log("Pull process finished.")
}

// --- Push Functions ---
func (ui *Gui) runPushAccounts() {
	ui.log("Starting push for account changes...")
	if err := app.RunPushAccounts(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
	}
	ui.log("Push process finished.")
}

func (ui *Gui) runPushCheckins() {
	ui.log("Starting push for check-in changes...")
	if err := app.RunPushCheckins(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
	}
	ui.log("Push process finished.")
}

func (ui *Gui) runPushAll() {
	ui.log("Starting push for all changes...")
	if err := app.RunPushAccounts(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR during account push: %v", err))
	}
	if err := app.RunPushCheckins(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR during check-in push: %v", err))
	}
	ui.log("Push process finished.")
}

// --- Config Functions ---
func (ui *Gui) saveConfig(apiKey, baseURL, dbType, dbPath, dbHost, dbPortStr, dbUser, dbPass, dbName string) {
	ui.log("Saving configuration...")

	// Update API config in memory and tell API module to save (which uses viper)
	ui.app.API.APIKey = apiKey
	ui.app.API.BaseURL = baseURL
	ui.app.API.SaveConfig()

	// Create a temporary DB object of the chosen type to save its config
	port, _ := strconv.Atoi(dbPortStr)
	var tempDb database.DB
	switch dbType {
	case "sqlite3":
		tempDb = &database.SQLiteConfig{Path: dbPath}
	case "postgres":
		tempDb = &database.PostgreSQLConfig{
			Host: dbHost, Port: port, Username: dbUser, Password: dbPass, Database: dbName, SSLMode: "disable",
		}
	case "mssql":
		tempDb = &database.MSSQLConfig{
			Host: dbHost, Port: port, Username: dbUser, Password: dbPass, Database: dbName,
		}
	default:
		ui.log(fmt.Sprintf("Unknown database type for saving: %s", dbType))
		return
	}
	tempDb.SaveConfig()

	// Explicitly set DB_TYPE as it's not part of the DB object's SaveConfig
	ui.app.DB.SaveConfig()

	// Write the accumulated viper config to file
	if err := ui.app.WriteConfig(); err != nil {
		ui.log(fmt.Sprintf("ERROR saving config file: %v", err))
		return
	}

	// Reload the application with the new config
	if err := ui.app.LoadConfig(); err != nil {
		ui.log(fmt.Sprintf("ERROR reloading config: %v", err))
		return
	}

	ui.log("Configuration saved successfully.")
	ui.refreshConfigTab()
}

// refreshConfigTab rebuilds and refreshes the configuration tab
func (ui *Gui) refreshConfigTab() {
	newConfigTab := ui.buildConfigTab()
	if ui.tabs != nil {
		for _, tab := range ui.tabs.Items {
			if tab.Text == "Configuration" {
				tab.Content = newConfigTab
				break
			}
		}
		ui.tabs.Refresh()
	}
}

// testDBConnection tests the database connection with the provided credentials
func (ui *Gui) testDBConnection(button *widget.Button, dbType, dbPath, dbHost, dbPortStr, dbUser, dbPass, dbName string) {
	ui.log(fmt.Sprintf("Testing connection for %s...", dbType))
	button.SetText("Testing...")
	button.Disable()
	defer func() {
		button.Enable()
	}()

	port, _ := strconv.Atoi(dbPortStr)

	// Create a temporary DB object for testing
	var db database.DB
	switch dbType {
	case "sqlite3":
		db = &database.SQLiteConfig{Path: dbPath}
	case "postgres":
		db = &database.PostgreSQLConfig{
			Host: dbHost, Port: port, Username: dbUser, Password: dbPass, Database: dbName, SSLMode: "disable",
		}
	case "mssql":
		db = &database.MSSQLConfig{
			Host: dbHost, Port: port, Username: dbUser, Password: dbPass, Database: dbName,
		}
	default:
		ui.log(fmt.Sprintf("Unknown database type for testing: %s", dbType))
		button.SetText("Test Failed")
		return
	}

	if err := db.Connect(); err != nil {
		ui.log(fmt.Sprintf("Failed to create connection: %v", err))
		button.SetText("Connection Failed")
		button.SetIcon(theme.ErrorIcon())
		return
	}
	defer db.Close()

	if err := db.TestConnection(); err != nil {
		ui.log(fmt.Sprintf("Connection failed: %v", err))
		button.SetText("Connection Failed")
		button.SetIcon(theme.ErrorIcon())
		return
	}

	ui.log("Connection successful!")
	button.SetText("Connection Successful")
	button.SetIcon(theme.ConfirmIcon())
}
