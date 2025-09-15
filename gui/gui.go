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
	"fyne.io/fyne/v2/dialog"
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
	detailsView     fyne.CanvasObject
	rightPane       *fyne.Container
	configTab       fyne.CanvasObject
	terminalVisible bool
	tabs            *container.AppTabs // Hold a reference to the tabs container
}

// Launch initializes and runs the GUI
func Launch(a *app.App, icon fyne.Resource) {
	fyneApp := fapp.New()
	fyneApp.SetIcon(icon)
	fyneApp.Settings().SetTheme(newModernTheme())
	window := fyneApp.NewWindow("BadgerMaps CLI")

	ui := &Gui{
		app:             a,
		fyneApp:         fyneApp,
		window:          window,
		logData:         []string{"Welcome to BadgerMaps CLI GUI!"},
		terminalVisible: false, // Default to details view
	}

	// Subscribe to events to refresh the events tab
	eventListener := func(e app.Event) {
		if ui.app.State.Debug {
			ui.log(fmt.Sprintf("GUI received event: %s", e.Type.String()))
		}
		if ui.tabs != nil {
			for _, tab := range ui.tabs.Items {
				if tab.Text == "Events" {
					// Re-create the content of the events tab
					tab.Content = ui.createEventsTab()
					ui.tabs.Refresh()
					break
				}
			}
		}
	}
	a.Events.Subscribe(app.EventCreate, eventListener)
	a.Events.Subscribe(app.EventDelete, eventListener)

	// Subscribe to pull events to show notifications
	pullNotificationListener := func(e app.Event) {
		if e.Type == app.PullStart {
			ui.showToast(fmt.Sprintf("Pulling %s from API...", e.Source))
		}
	}
	a.Events.Subscribe(app.PullStart, pullNotificationListener)

	window.SetContent(ui.createContent())
	window.Resize(fyne.NewSize(1280, 720))
	window.ShowAndRun()
}

// createContent builds the main content of the window
func (ui *Gui) createContent() fyne.CanvasObject {
	return ui.createMainContent()
}

// createMainContent builds the main layout with toolbar, tabs, and log view
func (ui *Gui) createMainContent() fyne.CanvasObject {
	ui.configTab = ui.buildConfigTab()
	tabs := []*container.TabItem{
		container.NewTabItemWithIcon("Pull", theme.DownloadIcon(), ui.createPullTab()),
		container.NewTabItemWithIcon("Push", theme.UploadIcon(), ui.createPushTab()),
		container.NewTabItemWithIcon("Events", theme.ListIcon(), ui.createEventsTab()),
		container.NewTabItemWithIcon("Explorer", theme.FolderIcon(), ui.createExplorerTab()),
		container.NewTabItemWithIcon("Configuration", theme.SettingsIcon(), ui.createConfigTab()),
	}

	if ui.app.State.Debug {
		tabs = append(tabs, container.NewTabItemWithIcon("Debug", theme.WarningIcon(), ui.createDebugTab()))
	}

	ui.tabs = container.NewAppTabs(tabs...)

	mainContent := container.NewBorder(nil, nil, nil, nil, ui.tabs)

	// Initialize log view
	ui.logView = widget.NewList(
		func() int { return len(ui.logData) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("template")
			label.Wrapping = fyne.TextWrapBreak
			return label
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(ui.logData[i])
		},
	)

	// Initialize details view
	ui.detailsView = container.NewCenter(widget.NewLabel("Select an item to see details"))

	rightPaneContent := container.NewBorder(
		ui.createRightPaneHeader(), nil, nil, nil,
		ui.detailsView,
	)
	ui.rightPane = rightPaneContent

	split := container.NewHSplit(mainContent, ui.rightPane)
	split.Offset = 0.7
	return split
}

func (ui *Gui) createRightPaneHeader() fyne.CanvasObject {
	detailsButton := widget.NewButtonWithIcon("Details", theme.InfoIcon(), func() {
		ui.terminalVisible = false
		ui.rightPane.Objects[0] = ui.detailsView
		ui.rightPane.Refresh()
	})

	logButton := widget.NewButtonWithIcon("Log", theme.ComputerIcon(), func() {
		ui.terminalVisible = true
		ui.rightPane.Objects[0] = ui.logView
		ui.rightPane.Refresh()
	})

	return container.NewHBox(detailsButton, logButton)
}

// showDetails updates the right-hand pane to show the provided details object.
func (ui *Gui) showDetails(details fyne.CanvasObject) {
	ui.detailsView = details
	ui.terminalVisible = false
	ui.rightPane.Objects[0] = ui.detailsView
	ui.rightPane.Refresh()
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

	singlePullCard := widget.NewCard("Pull Single Item by ID", "", container.NewVBox(
		container.NewGridWithColumns(2, accountIDEntry, pullAccountButton),
		container.NewGridWithColumns(2, checkinIDEntry, pullCheckinButton),
		container.NewGridWithColumns(2, routeIDEntry, pullRouteButton),
	))

	pullAccountsButton := widget.NewButtonWithIcon("Pull All Accounts", theme.DownloadIcon(), func() { go ui.runPullAccounts() })
	pullCheckinsButton := widget.NewButtonWithIcon("Pull All Check-ins", theme.DownloadIcon(), func() { go ui.runPullCheckins() })
	pullRoutesButton := widget.NewButtonWithIcon("Pull All Routes", theme.DownloadIcon(), func() { go ui.runPullRoutes() })
	pullProfileButton := widget.NewButtonWithIcon("Pull User Profile", theme.AccountIcon(), func() { go ui.runPullProfile() })

	bulkPullCard := widget.NewCard("Pull Data Sets", "", container.NewVBox(
		pullAccountsButton,
		pullCheckinsButton,
		pullRoutesButton,
		pullProfileButton,
	))

	pullAllButton := widget.NewButtonWithIcon("Run Full Pull (All Data)", theme.ViewRefreshIcon(), func() { go ui.runPullAll() })

	return container.NewVBox(
		singlePullCard,
		bulkPullCard,
		pullAllButton,
	)
}

// createPushTab creates the content for the "Push" tab
func (ui *Gui) createPushTab() fyne.CanvasObject {
	pushAccountsButton := widget.NewButtonWithIcon("Push Account Changes", theme.UploadIcon(), func() { go ui.runPushAccounts() })
	pushCheckinsButton := widget.NewButtonWithIcon("Push Check-in Changes", theme.UploadIcon(), func() { go ui.runPushCheckins() })
	pushAllButton := widget.NewButtonWithIcon("Push All Changes", theme.ViewRefreshIcon(), func() { go ui.runPushAll() })

	pushCard := widget.NewCard("Push Pending Changes", "", container.NewVBox(
		pushAccountsButton,
		pushCheckinsButton,
		widget.NewSeparator(),
		pushAllButton,
	))

	tableContainer := container.NewVBox()
	entityType := "accounts" // Default view

	radio := widget.NewRadioGroup([]string{"accounts", "checkins"}, func(selected string) {
		entityType = selected
		tableContainer.Objects = []fyne.CanvasObject{ui.createPendingChangesTable(entityType)}
		tableContainer.Refresh()
	})
	radio.SetSelected("accounts")

	tableContainer.Objects = []fyne.CanvasObject{ui.createPendingChangesTable(entityType)}

	changesCard := widget.NewCard("View Pending Changes", "", container.NewVBox(
		radio,
		tableContainer,
	))

	return container.NewVBox(
		pushCard,
		changesCard,
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
		func() (int, int) { return len(data) + 1, len(headers) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
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

	table.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 { // It's the header
			table.Unselect(id)
			return
		}
		selectedData := data[id.Row-1] // -1 for header

		// Format the data for display
		var details strings.Builder
		for i, header := range headers {
			details.WriteString(fmt.Sprintf("%s: %s\n", header, selectedData[i]))
		}

		detailsLabel := widget.NewLabel(details.String())
		detailsLabel.Wrapping = fyne.TextWrapWord

		ui.showDetails(container.NewScroll(detailsLabel))
	}

	return table
}

// createEventsTab creates the content for the "Events" tab
func (ui *Gui) createEventsTab() fyne.CanvasObject {
	eventsContent := container.NewVBox()

	// Function to refresh the events list
	var refreshEvents func()
	refreshEvents = func() {
		eventsContent.Objects = nil
		eventActions := ui.app.GetEventActions()

		// Create a sorted list of event names for consistent order
		var sortedEvents []string
		for eventName := range eventActions {
			sortedEvents = append(sortedEvents, eventName)
		}

		for _, eventName := range sortedEvents {
			actions := eventActions[eventName]

			eventLabel := widget.NewLabelWithStyle(eventName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			eventsContent.Add(eventLabel)

			for _, action := range actions {
				actionLabel := widget.NewLabel(action)
				// Use function closure to capture the correct eventName and action
				currentEvent := eventName
				currentAction := action
				removeButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
					err := ui.app.RemoveEventAction(currentEvent, currentAction)
					if err != nil {
						ui.log(fmt.Sprintf("Error removing event action: %v", err))
						return
					}
					ui.log(fmt.Sprintf("Removed action '%s' from event '%s'", currentAction, currentEvent))
				})
				runButton := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
					ui.log(fmt.Sprintf("Manually triggering action: %s", currentAction))
					if err := ui.app.TriggerEventAction(currentAction); err != nil {
						ui.log(fmt.Sprintf("Error triggering action: %v", err))
						ui.showToast(fmt.Sprintf("Error: Failed to trigger action: %v", err))
					} else {
						ui.showToast(fmt.Sprintf("Success: Action '%s' triggered.", currentAction))
					}
				})
				actionBox := container.NewHBox(removeButton, runButton, actionLabel)
				eventsContent.Add(actionBox)
			}
		}
		eventsContent.Refresh()
	}

	// Initial population
	refreshEvents()

	// Form for adding new event actions
	eventSelect := widget.NewSelect(app.AllEventTypes(), nil)
	actionTypeSelect := widget.NewSelect([]string{"exec", "db", "api"}, nil)
	actionTypeSelect.SetSelected("exec")
	actionEntry := widget.NewEntry()
	actionEntry.SetPlaceHolder("Enter command, function, or endpoint")

	addButton := widget.NewButtonWithIcon("Add Action", theme.ContentAddIcon(), func() {
		event := eventSelect.Selected
		actionType := actionTypeSelect.Selected
		action := actionEntry.Text

		if event == "" {
			ui.log("Please select an event.")
			return
		}
		if action == "" {
			ui.log("Please enter an action.")
			return
		}

		fullAction := fmt.Sprintf("%s:%s", actionType, action)

		err := ui.app.AddEventAction(event, fullAction)
		if err != nil {
			ui.log(fmt.Sprintf("Error adding event action: %v", err))
			return
		}

		ui.log(fmt.Sprintf("Added action '%s' to event '%s'", fullAction, event))
		actionEntry.SetText("")
	})

	addForm := widget.NewCard("Add New Event Action", "", container.NewVBox(
		eventSelect,
		actionTypeSelect,
		actionEntry,
		addButton,
	))

	return container.NewBorder(nil, addForm, nil, nil, container.NewScroll(eventsContent))
}

// createExplorerTab creates the content for the "Explorer" tab
func (ui *Gui) createExplorerTab() fyne.CanvasObject {
	tableContainer := container.NewMax() // Use NewMax to fill available space
	tableSelect := widget.NewSelect([]string{}, func(tableName string) {
		ui.log(fmt.Sprintf("Explorer: Loading table '%s'", tableName))
		if tableName == "" {
			tableContainer.Objects = nil
			tableContainer.Refresh()
			return
		}

		query := fmt.Sprintf("SELECT * FROM %s", tableName)

		rows, err := ui.app.DB.ExecuteQuery(query)
		if err != nil {
			ui.log(fmt.Sprintf("Error executing query: %v", err))
			tableContainer.Objects = []fyne.CanvasObject{widget.NewLabel(fmt.Sprintf("Error: %v", err))}
			tableContainer.Refresh()
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			ui.log(fmt.Sprintf("Error getting columns: %v", err))
			tableContainer.Objects = []fyne.CanvasObject{widget.NewLabel(fmt.Sprintf("Error: %v", err))}
			tableContainer.Refresh()
			return
		}
		ui.log(fmt.Sprintf("Explorer: Found %d columns in '%s'", len(columns), tableName))

		var data [][]string
		for rows.Next() {
			row := make([]interface{}, len(columns))
			rowData := make([]string, len(columns))
			for i := range row {
				row[i] = new(interface{})
			}
			if err := rows.Scan(row...); err != nil {
				ui.log(fmt.Sprintf("Error scanning row: %v", err))
				continue
			}
			for i, val := range row {
				if val == nil {
					rowData[i] = "NULL"
				} else {
					v := val.(*interface{})
					if b, ok := (*v).([]byte); ok {
						rowData[i] = string(b)
					} else {
						rowData[i] = fmt.Sprintf("%v", *v)
					}
				}
			}
			data = append(data, rowData)
		}
		ui.log(fmt.Sprintf("Explorer: Found %d rows in '%s'", len(data), tableName))

		if len(data) == 0 {
			tableContainer.Objects = []fyne.CanvasObject{widget.NewLabel("No rows in this table.")}
			tableContainer.Refresh()
			return
		}

		// Calculate column widths
		colWidths := make([]float32, len(columns))
		for i, colName := range columns {
			headerSize := fyne.MeasureText(colName, theme.TextSize(), fyne.TextStyle{Bold: true})
			maxWidth := headerSize.Width

			for _, rowData := range data {
				if i < len(rowData) {
					cellSize := fyne.MeasureText(rowData[i], theme.TextSize(), fyne.TextStyle{})
					if cellSize.Width > maxWidth {
						maxWidth = cellSize.Width
					}
				}
			}
			colWidths[i] = maxWidth + 10 // Add some padding
		}

		dataTable := widget.NewTable(
			func() (int, int) { return len(data) + 1, len(columns) },
			func() fyne.CanvasObject { return widget.NewLabel("template") },
			func(i widget.TableCellID, o fyne.CanvasObject) {
				label := o.(*widget.Label)
				if i.Row == 0 { // Header
					label.SetText(columns[i.Col])
					label.TextStyle = fyne.TextStyle{Bold: true}
				} else {
					label.SetText(data[i.Row-1][i.Col])
					label.TextStyle = fyne.TextStyle{}
				}
			},
		)

		dataTable.OnSelected = func(id widget.TableCellID) {
			if id.Row < 0 { // Deselection event
				return
			}
			if id.Row == 0 { // Header
				dataTable.Unselect(id)
				return
			}
			selectedData := data[id.Row-1]

			var details strings.Builder
			for i, header := range columns {
				details.WriteString(fmt.Sprintf("%s: %s\n", header, selectedData[i]))
			}

			detailsLabel := widget.NewLabel(details.String())
			detailsLabel.Wrapping = fyne.TextWrapWord
			ui.showDetails(container.NewScroll(detailsLabel))
		}

		for i, width := range colWidths {
			dataTable.SetColumnWidth(i, width)
		}

		tableContainer.Objects = []fyne.CanvasObject{dataTable}
		tableContainer.Refresh()
	})

	refreshButton := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		go func() {
			tables, err := ui.app.DB.GetTables()
			if err != nil {
				ui.log(fmt.Sprintf("Error getting tables: %v", err))
				return
			}
			tableSelect.Options = tables
			tableSelect.ClearSelected()
			tableSelect.Refresh()
			tableContainer.Objects = nil
			tableContainer.Refresh()
		}()
	})

	// Initial load
	go func() {
		tables, err := ui.app.DB.GetTables()
		if err != nil {
			ui.log(fmt.Sprintf("Error getting tables: %v", err))
			return
		}
		tableSelect.Options = tables
		tableSelect.Refresh()
	}()

	topContent := container.NewVBox(
		widget.NewLabelWithStyle("Database Explorer", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(tableSelect, refreshButton),
	)

	return container.NewBorder(topContent, nil, nil, nil, tableContainer)
}

// createDebugTab creates the content for the "Debug" tab
func (ui *Gui) createDebugTab() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabelWithStyle("Debug Information", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("Debug Mode: %v", ui.app.State.Debug)),
		widget.NewLabel(fmt.Sprintf("Verbose Mode: %v", ui.app.State.Verbose)),
		widget.NewLabel(fmt.Sprintf("Config File: %s", ui.app.ConfigFile)),
	)
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
	apiCard := widget.NewCard("API Configuration", "", widget.NewForm(
		widget.NewFormItem("API Key", apiKeyEntry),
		widget.NewFormItem("Base URL", baseURLEntry),
	))

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

	dbCard := widget.NewCard("Database Configuration", "", container.NewVBox(
		container.NewGridWithColumns(2, widget.NewLabel("Database Type"), dbTypeSelect),
		dbForm,
	))

	// Other Settings
	verboseCheck := widget.NewCheck("Verbose Logging", func(b bool) { ui.app.State.Verbose = b })
	verboseCheck.SetChecked(ui.app.State.Verbose)
	otherCard := widget.NewCard("Other Settings", "", verboseCheck)

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

	// Schema Management
	schemaLabel := "Initialize Schema"
	if err := ui.app.DB.ValidateSchema(); err == nil {
		schemaLabel = "Re-initialize Schema"
	}
	schemaButton := widget.NewButtonWithIcon(schemaLabel, theme.StorageIcon(), func() {
		go ui.runSchemaEnforcement()
	})

	schemaCard := widget.NewCard("Schema Management", "", schemaButton)

	return container.NewVBox(
		apiCard,
		dbCard,
		otherCard,
		schemaCard,
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

// showToast displays a transient popup message in the bottom right of the window.
func (ui *Gui) showToast(content string) {
	toastContent := container.NewPadded(widget.NewLabel(content))
	popup := widget.NewPopUp(toastContent, ui.window.Canvas())

	// Position the toast at the bottom right
	go func() {
		// We need a short delay to allow the popup to be sized
		time.Sleep(10 * time.Millisecond)
		winSize := ui.window.Canvas().Size()
		popupSize := popup.MinSize()
		popup.Move(fyne.NewPos(winSize.Width-popupSize.Width-theme.Padding(), winSize.Height-popupSize.Height-theme.Padding()))
	}()

	popup.Show()

	// Hide the popup after a short duration
	go func() {
		time.Sleep(3 * time.Second)
		popup.Hide()
	}()
}

// --- Pull Functions ---
func (ui *Gui) runPullAll() {
	ui.log("Starting full data pull...")
	if err := app.PullAll(ui.app, 0, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast("Error: The data pull failed.")
		return
	}
	ui.showToast("Success: Full data pull complete.")
}

func (ui *Gui) runPullAccount(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.log(fmt.Sprintf("Invalid Account ID: '%s'", idStr))
		return
	}
	ui.log(fmt.Sprintf("Starting pull for account ID: %d...", id))
	if err := app.PullAccount(ui.app, id, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast(fmt.Sprintf("Error: Failed to pull account %d.", id))
		return
	}
	ui.showToast(fmt.Sprintf("Success: Pulled account %d.", id))
}

func (ui *Gui) runPullAccounts() {
	ui.log("Starting pull for all accounts...")
	if err := app.PullAllAccounts(ui.app, 0, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast("Error: Failed to pull all accounts.")
		return
	}
	ui.showToast("Success: Pulled all accounts.")
}

func (ui *Gui) runPullCheckin(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.log(fmt.Sprintf("Invalid Check-in ID: '%s'", idStr))
		return
	}
	ui.log(fmt.Sprintf("Starting pull for check-in ID: %d...", id))
	if err := app.PullCheckin(ui.app, id, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast(fmt.Sprintf("Error: Failed to pull check-in %d.", id))
		return
	}
	ui.showToast(fmt.Sprintf("Success: Pulled check-in %d.", id))
}

func (ui *Gui) runPullCheckins() {
	ui.log("Starting pull for all check-ins...")
	if err := app.PullAllCheckins(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast("Error: Failed to pull all check-ins.")
		return
	}
	ui.showToast("Success: Pulled all check-ins.")
}

func (ui *Gui) runPullRoute(idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.log(fmt.Sprintf("Invalid Route ID: '%s'", idStr))
		return
	}
	ui.log(fmt.Sprintf("Starting pull for route ID: %d...", id))
	if err := app.PullRoute(ui.app, id, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast(fmt.Sprintf("Error: Failed to pull route %d.", id))
		return
	}
	ui.showToast(fmt.Sprintf("Success: Pulled route %d.", id))
}

func (ui *Gui) runPullRoutes() {
	ui.log("Starting pull for all routes...")
	if err := app.PullAllRoutes(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast("Error: Failed to pull all routes.")
		return
	}
	ui.showToast("Success: Pulled all routes.")
}

func (ui *Gui) runPullProfile() {
	ui.log("Starting pull for user profile...")
	if err := app.PullProfile(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast("Error: Failed to pull user profile.")
		return
	}
	ui.showToast("Success: Pulled user profile.")
}

// --- Push Functions ---
func (ui *Gui) runPushAccounts() {
	ui.log("Starting push for account changes...")
	if err := app.RunPushAccounts(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast("Error: Failed to push account changes.")
		return
	}
	ui.showToast("Success: Account changes pushed.")
}

func (ui *Gui) runPushCheckins() {
	ui.log("Starting push for check-in changes...")
	if err := app.RunPushCheckins(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast("Error: Failed to push check-in changes.")
		return
	}
	ui.showToast("Success: Check-in changes pushed.")
}

func (ui *Gui) runPushAll() {
	ui.log("Starting push for all changes...")
	if err := app.RunPushAccounts(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR during account push: %v", err))
	}
	if err := app.RunPushCheckins(ui.app, ui.log); err != nil {
		ui.log(fmt.Sprintf("ERROR during check-in push: %v", err))
	}
	ui.showToast("Success: All pending changes pushed.")
}

// --- Config Functions ---
func (ui *Gui) saveConfig(apiKey, baseURL, dbType, dbPath, dbHost, dbPortStr, dbUser, dbPass, dbName string) {
	ui.log("Saving configuration...")

	// Update API config in memory
	ui.app.Config.API.APIKey = apiKey
	ui.app.Config.API.BaseURL = baseURL

	port, _ := strconv.Atoi(dbPortStr)

	// Clear old DB config values
	ui.app.Config.DB = database.DBConfig{}

	ui.app.Config.DB.Type = dbType
	switch dbType {
	case "sqlite3":
		ui.app.Config.DB.Path = dbPath
	case "postgres":
		ui.app.Config.DB.Host = dbHost
		ui.app.Config.DB.Port = port
		ui.app.Config.DB.Username = dbUser
		ui.app.Config.DB.Password = dbPass
		ui.app.Config.DB.Database = dbName
		ui.app.Config.DB.SSLMode = "disable"
	case "mssql":
		ui.app.Config.DB.Host = dbHost
		ui.app.Config.DB.Port = port
		ui.app.Config.DB.Username = dbUser
		ui.app.Config.DB.Password = dbPass
		ui.app.Config.DB.Database = dbName
	}

	// Write the accumulated viper config to file
	if err := ui.app.SaveConfig(); err != nil {
		ui.log(fmt.Sprintf("ERROR saving config file: %v", err))
		ui.showToast("Error: Failed to save configuration.")
		return
	}

	// Reload the application with the new config
	if err := ui.app.LoadConfig(); err != nil {
		ui.log(fmt.Sprintf("ERROR reloading config: %v", err))
		ui.showToast("Error: Failed to reload new configuration.")
		return
	}
	if err := ui.app.ReloadDB(); err != nil {
		ui.log(fmt.Sprintf("ERROR reloading database: %v", err))
		ui.showToast("Error: Failed to reload database.")
		return
	}

	ui.showToast("Success: Configuration saved successfully.")
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

func (ui *Gui) runSchemaEnforcement() {
	if err := ui.app.DB.ValidateSchema(); err == nil {
		// Schema exists, confirm re-initialization
		dialog.ShowConfirm("Re-initialize Schema?", "This will delete all existing data. Are you sure?", func(ok bool) {
			if !ok {
				return
			}
			ui.log("Re-initializing database schema...")
			if err := ui.app.DB.EnforceSchema(); err != nil {
				ui.log(fmt.Sprintf("ERROR: %v", err))
				ui.showToast("Error: Failed to re-initialize schema.")
				return
			}
			ui.log("Schema re-initialized successfully.")
			ui.showToast("Success: Schema re-initialized.")
			ui.refreshConfigTab()
		}, ui.window)
	} else {
		// Schema doesn't exist, just initialize it
		ui.log("Initializing database schema...")
		if err := ui.app.DB.EnforceSchema(); err != nil {
			ui.log(fmt.Sprintf("ERROR: %v", err))
			ui.showToast("Error: Failed to initialize schema.")
			return
		}
		ui.log("Schema initialized successfully.")
		ui.showToast("Success: Schema initialized.")
		ui.refreshConfigTab()
	}
}

