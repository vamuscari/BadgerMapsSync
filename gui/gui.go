package gui

import (
	"badgermaps/app"
	"badgermaps/app/pull"
	"badgermaps/app/push"
	"badgermaps/app/server"
	"badgermaps/database"
	"badgermaps/events"
	"fmt"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"gopkg.in/yaml.v2"
	"image/color"
	"sort"
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

// SecondaryButton is a custom button that can be styled with a secondary color
type SecondaryButton struct {
	widget.Button
}

// NewSecondaryButton creates a new SecondaryButton
func NewSecondaryButton(label string, icon fyne.Resource, tapped func()) *SecondaryButton {
	b := &SecondaryButton{}
	b.Text = label
	b.Icon = icon
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
}

// CreateRenderer implements the Widget interface
func (b *SecondaryButton) CreateRenderer() fyne.WidgetRenderer {
	r := &secondaryButtonRenderer{
		button:     b,
		label:      widget.NewLabel(b.Text),
		icon:       widget.NewIcon(b.Icon),
		background: canvas.NewRectangle(theme.ButtonColor()),
	}
	r.objects = []fyne.CanvasObject{r.background, r.icon, r.label}
	return r
}

type secondaryButtonRenderer struct {
	button     *SecondaryButton
	label      *widget.Label
	icon       *widget.Icon
	background *canvas.Rectangle
	objects    []fyne.CanvasObject
}

func (r *secondaryButtonRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	padding := theme.Padding()
	if r.button.Icon != nil {
		iconSize := theme.IconInlineSize()
		r.icon.Resize(fyne.NewSize(iconSize, iconSize))
		r.icon.Move(fyne.NewPos(padding, (size.Height-iconSize)/2))
		r.label.Move(fyne.NewPos(padding*2+iconSize, (size.Height-r.label.MinSize().Height)/2))
	} else {
		r.label.Move(fyne.NewPos(padding, (size.Height-r.label.MinSize().Height)/2))
	}
}

func (r *secondaryButtonRenderer) MinSize() fyne.Size {
	iconSize := theme.IconInlineSize()
	padding := theme.Padding()
	min := r.label.MinSize()
	if r.button.Icon != nil {
		min.Width += iconSize + padding
	}
	min.Width += padding * 2
	min.Height += padding * 2
	return min
}

func (r *secondaryButtonRenderer) Refresh() {
	r.label.SetText(r.button.Text)
	r.icon.SetResource(r.button.Icon)
	r.background.FillColor = color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff} // A dark grey for secondary buttons
	if r.button.Disabled() {
		r.background.FillColor = theme.DisabledButtonColor()
	}
	r.background.Refresh()
	r.label.Refresh()
	r.icon.Refresh()
}

func (r *secondaryButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *secondaryButtonRenderer) Destroy() {}

// Spacer is a simple widget that creates a fixed-size empty space
type Spacer struct {
	widget.BaseWidget
	minSize fyne.Size
}

// NewSpacer creates a new Spacer
func NewSpacer(size fyne.Size) *Spacer {
	s := &Spacer{minSize: size}
	s.ExtendBaseWidget(s)
	return s
}

// CreateRenderer implements the Widget interface
func (s *Spacer) CreateRenderer() fyne.WidgetRenderer {
	return &spacerRenderer{spacer: s}
}

type spacerRenderer struct {
	spacer *Spacer
}

func (r *spacerRenderer) Layout(size fyne.Size) {}

func (r *spacerRenderer) MinSize() fyne.Size {
	return r.spacer.minSize
}

func (r *spacerRenderer) Refresh() {}

func (r *spacerRenderer) Objects() []fyne.CanvasObject {
	return nil
}

func (r *spacerRenderer) Destroy() {}

type logEntry struct {
	widget.BaseWidget
	label *widget.Label
}

func newLogEntry() *logEntry {
	e := &logEntry{
		label: widget.NewLabel(""),
	}
	e.label.Wrapping = fyne.TextWrapWord
	e.ExtendBaseWidget(e)
	return e
}

func (e *logEntry) SetText(text string) {
	lines := strings.Split(text, "\n")
	if len(lines) > 3 {
		text = strings.Join(lines[:3], "\n") + "..."
	}
	e.label.SetText(text)
}

func (e *logEntry) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(e.label)
}

func (e *logEntry) MinSize() fyne.Size {
	min := e.label.MinSize()
	if e.label.Text != "" {
		lines := strings.Count(e.label.Text, "\n") + 1
		if lines > 3 {
			min.Height = e.label.MinSize().Height * 3
		}
	}
	return min
}

// Gui struct holds all the UI components and application state
type Gui struct {
	app        *app.App
	fyneApp    fyne.App
	window     fyne.Window
	logMutex   sync.Mutex
	toastMutex sync.Mutex

	logBinding        binding.StringList
	logView           *widget.List
	detailsView       fyne.CanvasObject
	rightPane         *fyne.Container
	configTab         fyne.CanvasObject
	progressBar       *widget.ProgressBar
	progressContainer *fyne.Container
	progressTitle     *widget.Label

	terminalVisible bool
	tabs            *container.AppTabs // Hold a reference to the tabs container
}

// Launch initializes and runs the GUI
func Launch(a *app.App, icon fyne.Resource) {
	a.Events.Dispatch(events.Debugf("gui", "GUI initiated"))
	fyneApp := fapp.New()
	fyneApp.SetIcon(icon)
	fyneApp.Settings().SetTheme(newModernTheme())
	window := fyneApp.NewWindow("BadgerMaps CLI")

	ui := &Gui{
		app:             a,
		fyneApp:         fyneApp,
		window:          window,
		logBinding:      binding.NewStringList(),
		terminalVisible: false, // Default to details view
	}

	// Subscribe to events to refresh the events tab
	eventListener := func(e events.Event) {
		if ui.app.State.Debug {
			ui.log(fmt.Sprintf("GUI received event: %s", e.Type.String()))
		}
		if ui.tabs != nil {
			for _, tab := range ui.tabs.Items {
				if tab.Text == "Actions" {
					// Re-create the content of the events tab
					tab.Content = ui.createActionsTab()
					ui.tabs.Refresh()
					break
				}
			}
		}
	}
	a.Events.Subscribe(events.ActionConfigCreated, eventListener)
	a.Events.Subscribe(events.ActionConfigUpdated, eventListener)
	a.Events.Subscribe(events.ActionConfigDeleted, eventListener)

	// Subscribe to logging and action events
	logListener := func(e events.Event) {
		switch e.Type {
		case events.LogEvent:
			// This is now handled by the central log listener, but we could add GUI-specific actions here if needed.
		case events.ActionStart:
			ui.log(fmt.Sprintf("Starting action for event '%s': %s", e.Source, e.Payload.(string)))
		case events.ActionSuccess:
			ui.log(fmt.Sprintf("Action for event '%s' completed successfully: %s", e.Source, e.Payload.(string)))
		case events.ActionError:
			ui.log(fmt.Sprintf("Action for event '%s' failed: %v", e.Source, e.Payload.(error)))
		case events.Debug:
			if msg, ok := e.Payload.(string); ok {
				ui.log(fmt.Sprintf("DEBUG: %s", msg))
			}
		}
	}
	a.Events.Subscribe(events.LogEvent, logListener)
	a.Events.Subscribe(events.ActionStart, logListener)
	a.Events.Subscribe(events.ActionSuccess, logListener)
	a.Events.Subscribe(events.ActionError, logListener)
	a.Events.Subscribe(events.Debug, logListener)

	// Subscribe to pull events to show notifications
	pullNotificationListener := func(e events.Event) {
		switch e.Type {
		case events.PullStart:
			ui.showToast(fmt.Sprintf("Pulling %s from API...", e.Source))
		case events.PullComplete:
			ui.showToast(fmt.Sprintf("Successfully pulled %s.", e.Source))
		case events.PullError:
			ui.showToast(fmt.Sprintf("Error pulling %s.", e.Source))
		case events.PullGroupStart:
			ui.showToast(fmt.Sprintf("Starting full pull for %s...", e.Source))
		case events.PullGroupComplete:
			ui.showToast(fmt.Sprintf("Successfully pulled all %s.", e.Source))
		case events.PullGroupError:
			ui.showToast(fmt.Sprintf("Error pulling all %s.", e.Source))
		}
	}
	a.Events.Subscribe(events.PullStart, pullNotificationListener)
	a.Events.Subscribe(events.PullComplete, pullNotificationListener)
	a.Events.Subscribe(events.PullError, pullNotificationListener)
	a.Events.Subscribe(events.PullGroupStart, pullNotificationListener)
	a.Events.Subscribe(events.PullGroupComplete, pullNotificationListener)
	a.Events.Subscribe(events.PullGroupError, pullNotificationListener)

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
		container.NewTabItemWithIcon("Actions", theme.ListIcon(), ui.createActionsTab()),
		container.NewTabItemWithIcon("Explorer", theme.FolderIcon(), ui.createExplorerTab()),
		container.NewTabItemWithIcon("Server", theme.ComputerIcon(), ui.createServerTab()),
		container.NewTabItemWithIcon("Configuration", theme.SettingsIcon(), ui.createConfigTab()),
	}

	if ui.app.State.Debug {
		tabs = append(tabs, container.NewTabItemWithIcon("Debug", theme.WarningIcon(), ui.createDebugTab()))
	}

	ui.tabs = container.NewAppTabs(tabs...)

	ui.progressBar = widget.NewProgressBar()
	ui.progressTitle = widget.NewLabel("")
	ui.progressContainer = container.NewVBox(ui.progressTitle, ui.progressBar)
	ui.progressContainer.Hide()

	mainContent := container.NewBorder(nil, ui.progressContainer, nil, nil, ui.tabs)

	// Initialize log view
	ui.logView = widget.NewListWithData(ui.logBinding,
		func() fyne.CanvasObject {
			return newLogEntry()
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			text, _ := i.(binding.String).Get()
			o.(*logEntry).SetText(text)
		},
	)
	ui.logView.OnSelected = func(id widget.ListItemID) {
		fullLog, _ := ui.logBinding.GetValue(id)
		detailsLabel := widget.NewLabel(fullLog)
		detailsLabel.Wrapping = fyne.TextWrapWord
		ui.showDetails(container.NewScroll(detailsLabel))
		ui.logView.Unselect(id)
	}

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
	var content fyne.CanvasObject
	if label, ok := details.(*widget.Label); ok {
		entry := widget.NewMultiLineEntry()
		entry.SetText(label.Text)
		entry.Disable()
		content = entry
	} else {
		content = details
	}

	ui.detailsView = container.NewScroll(content)
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

	pullAllButton := widget.NewButtonWithIcon("Run Full Pull (All Data)", theme.ViewRefreshIcon(), func() { go ui.runPullGroup() })

	return container.NewVScroll(container.NewVBox(
		singlePullCard,
		bulkPullCard,
		pullAllButton,
	))
}

// createPushTab creates the content for the "Push" tab
func (ui *Gui) createPushTab() fyne.CanvasObject {
	if ui.app.DB == nil || !ui.app.DB.IsConnected() {
		return widget.NewLabel("Database not configured. Please configure it in the Configuration tab.")
	}
	pushAccountsButton := widget.NewButtonWithIcon("Push Account Changes", theme.UploadIcon(), func() { go ui.runPushAccounts() })
	pushCheckinsButton := widget.NewButtonWithIcon("Push Check-in Changes", theme.UploadIcon(), func() { go ui.runPushCheckins() })
	pushAllButton := widget.NewButtonWithIcon("Push All Changes", theme.ViewRefreshIcon(), func() { go ui.runPushAll() })

	pushCard := widget.NewCard("Push Pending Changes", "", container.NewVBox(
		pushAccountsButton,
		pushCheckinsButton,
		widget.NewSeparator(),
		pushAllButton,
	))

	tableContainer := container.NewMax()
	entityType := "accounts" // Default view

	radio := widget.NewRadioGroup([]string{"accounts", "checkins"}, func(selected string) {
		entityType = selected
		tableContainer.Objects = []fyne.CanvasObject{ui.createPendingChangesTable(entityType)}
		tableContainer.Refresh()
	})
	radio.SetSelected("accounts")

	tableContainer.Objects = []fyne.CanvasObject{ui.createPendingChangesTable(entityType)}

	changesCard := widget.NewCard("View Pending Changes", "", container.NewBorder(radio, nil, nil, nil, tableContainer))

	return container.NewVScroll(container.NewBorder(pushCard, nil, nil, nil, changesCard))
}

func (ui *Gui) createPendingChangesTable(entityType string) fyne.CanvasObject {
	options := push.PushFilterOptions{
		Status:  "pending",
		OrderBy: "date_desc",
	}

	results, err := push.GetFilteredPendingChanges(ui.app, entityType, options)
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

	dataTable := widget.NewTable(
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
		for i, header := range headers {
			details.WriteString(fmt.Sprintf("%s: %s\n", header, selectedData[i]))
		}

		detailsEntry := widget.NewMultiLineEntry()
		detailsEntry.SetText(details.String())
		detailsEntry.Disable()

		ui.showDetails(detailsEntry)
	}

	return dataTable
}

// createActionsTab creates the content for the "Actions" tab
func (ui *Gui) createActionsTab() fyne.CanvasObject {
	actionsContent := container.NewVBox()

	var refreshActions func()
	refreshActions = func() {
		actionsContent.Objects = nil
		eventActions := ui.app.GetEventActions()

		sort.Slice(eventActions, func(i, j int) bool {
			return eventActions[i].Name < eventActions[j].Name
		})

		for _, eventAction := range eventActions {
			eventLabel := widget.NewLabelWithStyle(eventAction.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			actionsContent.Add(eventLabel)

			for i, action := range eventAction.Run {
				actionLabel := widget.NewLabel(fmt.Sprintf("%s: %v", action.Type, action.Args))
				currentEventAction := eventAction
				actionIndex := i

				editButton := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
					ui.createActionPopup(&currentEventAction, actionIndex)
				})
				removeButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
					err := ui.app.RemoveEventAction(currentEventAction.Name, actionIndex)
					if err != nil {
						ui.log(fmt.Sprintf("Error removing action: %v", err))
					}
				})

				actionBox := container.NewHBox(editButton, removeButton, actionLabel)
				actionsContent.Add(actionBox)
			}
		}
		actionsContent.Refresh()
	}

	refreshActions()

	addButton := widget.NewButtonWithIcon("Add Action", theme.ContentAddIcon(), func() {
		ui.createActionPopup(nil, -1)
	})

	return container.NewBorder(nil, addButton, nil, nil, container.NewScroll(actionsContent))
}

func (ui *Gui) createActionPopup(eventAction *events.EventAction, actionIndex int) {
	var event, source, actionType, command, function, endpoint, method string
	var args []string

	if eventAction != nil {
		event = eventAction.Event
		source = eventAction.Source
		if actionIndex != -1 {
			action := eventAction.Run[actionIndex]
			actionType = action.Type
			switch actionType {
			case "exec":
				command = action.Args["command"].(string)
				if action.Args["args"] != nil {
					for _, arg := range action.Args["args"].([]interface{}) {
						args = append(args, arg.(string))
					}
				}
			case "db":
				function = action.Args["function"].(string)
			case "api":
				endpoint = action.Args["endpoint"].(string)
				method = action.Args["method"].(string)
			}
		}
	}

	eventEntry := widget.NewSelect(events.AllEventTypes(), nil)
	eventEntry.SetSelected(event)
	sourceEntry := widget.NewSelect(events.AllEventSources(), nil)
	sourceEntry.SetSelected(source)
	actionTypeEntry := widget.NewSelect([]string{"exec", "db", "api"}, nil)
	actionTypeEntry.SetSelected(actionType)

	formItems := []*widget.FormItem{
		widget.NewFormItem("Event", eventEntry),
		widget.NewFormItem("Source", sourceEntry),
		widget.NewFormItem("Action Type", actionTypeEntry),
	}

	execCommandEntry := widget.NewEntry()
	execCommandEntry.SetText(command)
	execArgsEntry := widget.NewEntry()
	execArgsEntry.SetText(strings.Join(args, " "))
	dbFunctionEntry := widget.NewEntry()
	dbFunctionEntry.SetText(function)
	apiEndpointEntry := widget.NewEntry()
	apiEndpointEntry.SetText(endpoint)
	apiMethodEntry := widget.NewSelect([]string{"GET", "POST", "PUT", "DELETE"}, nil)
	apiMethodEntry.SetSelected(method)

	execForm := []*widget.FormItem{
		widget.NewFormItem("Command", execCommandEntry),
		widget.NewFormItem("Args", execArgsEntry),
	}
	dbForm := []*widget.FormItem{
		widget.NewFormItem("Function", dbFunctionEntry),
	}
	apiForm := []*widget.FormItem{
		widget.NewFormItem("Endpoint", apiEndpointEntry),
		widget.NewFormItem("Method", apiMethodEntry),
	}

	form := widget.NewForm(formItems...)

	actionTypeEntry.OnChanged = func(selected string) {
		form.Items = formItems
		switch selected {
		case "exec":
			for _, item := range execForm {
				form.AppendItem(item)
			}
		case "db":
			for _, item := range dbForm {
				form.AppendItem(item)
			}
		case "api":
			for _, item := range apiForm {
				form.AppendItem(item)
			}
		}
		form.Refresh()
	}
	actionTypeEntry.OnChanged(actionType)

	content := container.NewVBox(form)
	d := dialog.NewCustom("Action", "Cancel", content, ui.window)
	d.Resize(fyne.NewSize(400, 300))

	saveButton := widget.NewButton("Save", func() {
		var newAction events.ActionConfig
		newAction.Type = actionTypeEntry.Selected
		newAction.Args = make(map[string]interface{})

		switch newAction.Type {
		case "exec":
			newAction.Args["command"] = execCommandEntry.Text
			newAction.Args["args"] = strings.Split(execArgsEntry.Text, " ")
		case "db":
			newAction.Args["function"] = dbFunctionEntry.Text
		case "api":
			newAction.Args["endpoint"] = apiEndpointEntry.Text
			newAction.Args["method"] = apiMethodEntry.Selected
		}

		if eventAction == nil {
			// Add new event action
			err := ui.app.AddEventAction(eventEntry.Selected, sourceEntry.Selected, newAction)
			if err != nil {
				ui.log(fmt.Sprintf("Error adding action: %v", err))
			}
		} else {
			// Update existing event action
			err := ui.app.UpdateEventAction(eventAction.Name, actionIndex, newAction)
			if err != nil {
				ui.log(fmt.Sprintf("Error updating action: %v", err))
			}
		}
		d.Hide()
	})

	d.SetButtons([]fyne.CanvasObject{saveButton})
	d.Show()
}

// createExplorerTab creates the content for the "Explorer" tab
func (ui *Gui) createExplorerTab() fyne.CanvasObject {
	if ui.app.DB == nil || !ui.app.DB.IsConnected() {
		return widget.NewLabel("Database not configured. Please configure it in the Configuration tab.")
	}
	tableContainer := container.NewMax() // Use NewMax to fill available space
	tableSelect := widget.NewSelect([]string{}, func(tableName string) {
		ui.log(fmt.Sprintf("Explorer: Loading table '%s'", tableName))
		if tableName == "" {
			tableContainer.Objects = nil
			tableContainer.Refresh()
			return
		}

		var query string
		dbType := ui.app.DB.GetType()
		switch dbType {
		case "mssql":
			query = fmt.Sprintf("SELECT TOP 100 * FROM %s", tableName)
		default: // sqlite3 and postgres
			query = fmt.Sprintf("SELECT * FROM %s LIMIT 100", tableName)
		}

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

	return container.NewVScroll(container.NewBorder(topContent, nil, nil, nil, tableContainer))
}

// createServerTab creates the content for the "Server" tab
func (ui *Gui) createServerTab() fyne.CanvasObject {
	serverStatusLabel := widget.NewLabel("Status: Unknown")
	startButton := widget.NewButtonWithIcon("Start Server", theme.MediaPlayIcon(), nil)
	stopButton := widget.NewButtonWithIcon("Stop Server", theme.MediaStopIcon(), nil)

	var refreshServerStatus func()
	refreshServerStatus = func() {
		if pid, running := server.GetServerStatus(ui.app); running {
			serverStatusLabel.SetText(fmt.Sprintf("Status: Running (PID: %d)", pid))
			startButton.Disable()
			stopButton.Enable()
		} else {
			serverStatusLabel.SetText("Status: Stopped")
			startButton.Enable()
			stopButton.Disable()
		}
	}

	startButton.OnTapped = func() {
		if err := server.StartServer(ui.app); err != nil {
			ui.log(fmt.Sprintf("Error starting server: %v", err))
			dialog.ShowError(err, ui.window)
		}
		refreshServerStatus()
	}

	stopButton.OnTapped = func() {
		if err := server.StopServer(ui.app); err != nil {
			ui.log(fmt.Sprintf("Error stopping server: %v", err))
			dialog.ShowError(err, ui.window)
		}
		refreshServerStatus()
	}

	refreshServerStatus()

	serverCard := widget.NewCard("Server Management", "", container.NewVBox(
		serverStatusLabel,
		container.NewGridWithColumns(2, startButton, stopButton),
	))

	return container.NewVScroll(container.NewVBox(serverCard))
}

// createDebugTab creates the content for the "Debug" tab
func (ui *Gui) createDebugTab() fyne.CanvasObject {
	return container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("Debug Information", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("Debug Mode: %v", ui.app.State.Debug)),
		widget.NewLabel(fmt.Sprintf("Verbose Mode: %v", ui.app.State.Verbose)),
		widget.NewLabel(fmt.Sprintf("Config File: %s", ui.app.ConfigFile)),
	))
}

// createConfigTab returns the config tab content
func (ui *Gui) createConfigTab() fyne.CanvasObject {
	return container.NewVScroll(ui.configTab)
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

	// Server Settings
	serverHostEntry := widget.NewEntry()
	serverHostEntry.SetText(ui.app.State.ServerHost)
	serverPortEntry := widget.NewEntry()
	serverPortEntry.SetText(fmt.Sprintf("%d", ui.app.State.ServerPort))
	tlsCertEntry := widget.NewEntry()
	tlsCertEntry.SetText(ui.app.State.TLSCert)
	tlsKeyEntry := widget.NewEntry()
	tlsKeyEntry.SetText(ui.app.State.TLSKey)

	tlsCertFormItem := widget.NewFormItem("TLS Cert Path", tlsCertEntry)
	tlsKeyFormItem := widget.NewFormItem("TLS Key Path", tlsKeyEntry)

	serverForm := widget.NewForm(
		widget.NewFormItem("Host", serverHostEntry),
		widget.NewFormItem("Port", serverPortEntry),
	)

	tlsEnabledCheck := widget.NewCheck("Enable TLS", func(enabled bool) {
		if enabled {
			serverForm.AppendItem(tlsCertFormItem)
			serverForm.AppendItem(tlsKeyFormItem)
		} else {
			serverForm.Items = serverForm.Items[:2] // Keep only host and port
		}
		serverForm.Refresh()
	})
	tlsEnabledCheck.SetChecked(ui.app.State.TLSEnabled)
	if ui.app.State.TLSEnabled {
		serverForm.AppendItem(tlsCertFormItem)
		serverForm.AppendItem(tlsKeyFormItem)
	}

	serverCard := widget.NewCard("Server Configuration", "", container.NewVBox(
		tlsEnabledCheck,
		serverForm,
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

	saveButton := NewSecondaryButton("Save Configuration", theme.ConfirmIcon(), func() {
		ui.saveConfig(
			apiKeyEntry.Text, baseURLEntry.Text, dbTypeSelect.Selected, dbPathEntry.Text,
			dbHostEntry.Text, dbPortEntry.Text, dbUserEntry.Text, dbPassEntry.Text, dbNameEntry.Text,
			serverHostEntry.Text, serverPortEntry.Text, tlsEnabledCheck.Checked, tlsCertEntry.Text, tlsKeyEntry.Text,
		)
	})

	viewButton := widget.NewButtonWithIcon("View", theme.VisibilityIcon(), func() {
		configData, err := yaml.Marshal(ui.app.Config)
		if err != nil {
			ui.log(fmt.Sprintf("Error marshaling config: %v", err))
			return
		}
		ui.showDetails(widget.NewLabel(string(configData)))
	})

	// Schema Management
	schemaLabel := "Initialize Schema"
	if ui.app.DB != nil && ui.app.DB.IsConnected() {
		if err := ui.app.DB.ValidateSchema(ui.app.State); err == nil {
			schemaLabel = "Re-initialize Schema"
		}
	}
	schemaButton := widget.NewButtonWithIcon(schemaLabel, theme.StorageIcon(), func() {
		go ui.runSchemaEnforcement()
	})

	schemaCard := widget.NewCard("Schema Management", "", schemaButton)

	topButtons := container.NewGridWithColumns(3, viewButton, testButton, saveButton)

	return container.NewVBox(
		NewSpacer(fyne.NewSize(0, 10)),
		topButtons,
		apiCard,
		dbCard,
		serverCard,
		otherCard,
		schemaCard,
	)
}

// log adds a message to the log view
func (ui *Gui) log(message string) {
	ui.logMutex.Lock()
	defer ui.logMutex.Unlock()
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		ui.logBinding.Append(line)
	}
	if ui.logView != nil {
		ui.logView.ScrollToBottom()
	}
	if strings.HasPrefix(message, "ERROR") {
		ui.app.Events.Dispatch(events.Errorf("gui", message))
	}
}

// showToast displays a transient popup message in the bottom right of the window.
func (ui *Gui) showToast(content string) {
	if !ui.toastMutex.TryLock() {
		return // Don't show a new toast if one is already visible
	}

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
		ui.toastMutex.Unlock()
	}()
}

// --- Pull Functions ---

func (ui *Gui) showProgressBar(title string) {
	ui.progressTitle.SetText(title)
	ui.progressContainer.Show()
}

func (ui *Gui) hideProgressBar() {
	ui.progressContainer.Hide()
	ui.progressTitle.SetText("")
}

func (ui *Gui) setProgress(value float64) {
	ui.progressBar.SetValue(value)
}

func (ui *Gui) runPullGroup() {
	ui.app.Events.Dispatch(events.Debugf("gui", "runPullGroup called"))
	ui.log("Starting full data pull...")
	ui.showProgressBar("Running Full Pull...")
	ui.setProgress(0)
	defer ui.hideProgressBar()

	totalMajorSteps := 4.0
	majorStepWeight := 1.0 / totalMajorSteps

	ui.log("Pulling accounts...")
	accountsCallback := func(current, total int) {
		progress := (float64(current) / float64(total)) * majorStepWeight
		ui.setProgress(progress)
	}
	if err := pull.PullGroupAccounts(ui.app, 0, accountsCallback); err != nil {
		ui.app.Events.Dispatch(events.Errorf("gui", "Error pulling accounts: %v", err))
		ui.showToast("Error: The data pull failed.")
		return
	}
	ui.setProgress(majorStepWeight)

	ui.log("Pulling checkins...")
	checkinsCallback := func(current, total int) {
		progress := majorStepWeight + (float64(current)/float64(total))*majorStepWeight
		ui.setProgress(progress)
	}
	if err := pull.PullGroupCheckins(ui.app, checkinsCallback); err != nil {
		ui.app.Events.Dispatch(events.Errorf("gui", "Error pulling checkins: %v", err))
		ui.showToast("Error: The data pull failed.")
		return
	}
	ui.setProgress(2 * majorStepWeight)

	ui.log("Pulling routes...")
	routesCallback := func(current, total int) {
		progress := 2*majorStepWeight + (float64(current)/float64(total))*majorStepWeight
		ui.setProgress(progress)
	}
	if err := pull.PullGroupRoutes(ui.app, routesCallback); err != nil {
		ui.app.Events.Dispatch(events.Errorf("gui", "Error pulling routes: %v", err))
		ui.showToast("Error: The data pull failed.")
		return
	}
	ui.setProgress(3 * majorStepWeight)

	ui.log("Pulling user profile...")
	profileCallback := func(current, total int) {
		progress := 3*majorStepWeight + (float64(current)/float64(total))*majorStepWeight
		ui.setProgress(progress)
	}
	if err := pull.PullProfile(ui.app, profileCallback); err != nil {
		ui.app.Events.Dispatch(events.Errorf("gui", "Error pulling user profile: %v", err))
		ui.showToast("Error: The data pull failed.")
		return
	}
	ui.setProgress(4 * majorStepWeight)

	ui.log("Finished pulling all data.")
	ui.showToast("Success: Full data pull complete.")
}

func (ui *Gui) runPullAccount(idStr string) {
	ui.app.Events.Dispatch(events.Debugf("gui", "runPullAccount called with id: %s", idStr))
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.log(fmt.Sprintf("Invalid Account ID: '%s'", idStr))
		return
	}
	ui.log(fmt.Sprintf("Starting pull for account ID: %d...", id))
	if err := pull.PullAccount(ui.app, id); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast(fmt.Sprintf("Error: Failed to pull account %d.", id))
		return
	}
	ui.showToast(fmt.Sprintf("Success: Pulled account %d.", id))
}

func (ui *Gui) runPullAccounts() {
	go func() {
		ui.app.Events.Dispatch(events.Debugf("gui", "runPullAccounts called"))
		ui.log("Starting pull for all accounts...")
		ui.showProgressBar("Pulling Accounts...")
		ui.setProgress(0)
		defer ui.hideProgressBar()

		callback := func(current, total int) {
			ui.setProgress(float64(current) / float64(total))
		}
		if err := pull.PullGroupAccounts(ui.app, 0, callback); err != nil {
			ui.log(fmt.Sprintf("ERROR: %v", err))
			ui.showToast("Error: Failed to pull all accounts.")
			return
		}
		ui.setProgress(1)
		ui.showToast("Success: Pulled all accounts.")
	}()
}

func (ui *Gui) runPullCheckin(idStr string) {
	ui.app.Events.Dispatch(events.Debugf("gui", "runPullCheckin called with id: %s", idStr))
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.log(fmt.Sprintf("Invalid Check-in ID: '%s'", idStr))
		return
	}
	ui.log(fmt.Sprintf("Starting pull for check-in ID: %d...", id))
	if err := pull.PullCheckin(ui.app, id); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast(fmt.Sprintf("Error: Failed to pull check-in %d.", id))
		return
	}
	ui.showToast(fmt.Sprintf("Success: Pulled check-in %d.", id))
}

func (ui *Gui) runPullCheckins() {
	go func() {
		ui.app.Events.Dispatch(events.Debugf("gui", "runPullCheckins called"))
		ui.log("Starting pull for all check-ins...")
		ui.showProgressBar("Pulling Check-ins...")
		ui.setProgress(0)
		defer ui.hideProgressBar()

		callback := func(current, total int) {
			ui.setProgress(float64(current) / float64(total))
		}
		if err := pull.PullGroupCheckins(ui.app, callback); err != nil {
			ui.log(fmt.Sprintf("ERROR: %v", err))
			ui.showToast("Error: Failed to pull all check-ins.")
			return
		}
		ui.setProgress(1)
		ui.showToast("Success: Pulled all check-ins.")
	}()
}

func (ui *Gui) runPullRoute(idStr string) {
	ui.app.Events.Dispatch(events.Debugf("gui", "runPullRoute called with id: %s", idStr))
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.log(fmt.Sprintf("Invalid Route ID: '%s'", idStr))
		return
	}
	ui.log(fmt.Sprintf("Starting pull for route ID: %d...", id))
	if err := pull.PullRoute(ui.app, id); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast(fmt.Sprintf("Error: Failed to pull route %d.", id))
		return
	}
	ui.showToast(fmt.Sprintf("Success: Pulled route %d.", id))
}

func (ui *Gui) runPullRoutes() {
	go func() {
		ui.app.Events.Dispatch(events.Debugf("gui", "runPullRoutes called"))
		ui.log("Starting pull for all routes...")
		ui.showProgressBar("Pulling Routes...")
		ui.setProgress(0)
		defer ui.hideProgressBar()

		callback := func(current, total int) {
			ui.setProgress(float64(current) / float64(total))
		}
		if err := pull.PullGroupRoutes(ui.app, callback); err != nil {
			ui.log(fmt.Sprintf("ERROR: %v", err))
			ui.showToast("Error: Failed to pull all routes.")
			return
		}
		ui.setProgress(1)
		ui.showToast("Success: Pulled all routes.")
		return
	}()
}

func (ui *Gui) runPullProfile() {
	go func() {
		if ui.app.DB == nil || ui.app.DB.GetDB() == nil {
			if err := ui.app.ReloadDB(); err != nil {
				ui.log(fmt.Sprintf("ERROR: Failed to connect to database: %v", err))
				ui.showToast("Error: Failed to connect to database.")
				return
			}
		}
		ui.app.Events.Dispatch(events.Debugf("gui", "runPullProfile called"))
		ui.log("Starting pull for user profile...")
		ui.showProgressBar("Pulling User Profile...")
		ui.setProgress(0)
		defer ui.hideProgressBar()

		callback := func(current, total int) {
			ui.setProgress(float64(current) / float64(total))
		}
		if err := pull.PullProfile(ui.app, callback); err != nil {
			ui.log(fmt.Sprintf("ERROR: %v", err))
			ui.showToast("Error: Failed to pull user profile.")
			return
		}
		ui.setProgress(1)
		ui.showToast("Success: Pulled user profile.")
	}()
}

// --- Push Functions ---
func (ui *Gui) runPushAccounts() {
	ui.app.Events.Dispatch(events.Debugf("gui", "runPushAccounts called"))
	ui.log("Starting push for account changes...")
	if err := push.RunPushAccounts(ui.app); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast("Error: Failed to push account changes.")
		return
	}
	ui.showToast("Success: Account changes pushed.")
}

func (ui *Gui) runPushCheckins() {
	ui.app.Events.Dispatch(events.Debugf("gui", "runPushCheckins called"))
	ui.log("Starting push for check-in changes...")
	if err := push.RunPushCheckins(ui.app); err != nil {
		ui.log(fmt.Sprintf("ERROR: %v", err))
		ui.showToast("Error: Failed to push check-in changes.")
		return
	}
	ui.showToast("Success: Check-in changes pushed.")
}

func (ui *Gui) runPushAll() {
	ui.app.Events.Dispatch(events.Debugf("gui", "runPushAll called"))
	ui.log("Starting push for all changes...")
	if err := push.RunPushAccounts(ui.app); err != nil {
		ui.log(fmt.Sprintf("ERROR during account push: %v", err))
	}
	if err := push.RunPushCheckins(ui.app); err != nil {
		ui.log(fmt.Sprintf("ERROR during check-in push: %v", err))
	}
	ui.showToast("Success: All pending changes pushed.")
}

// --- Config Functions ---
func (ui *Gui) saveConfig(
	apiKey, baseURL, dbType, dbPath, dbHost, dbPortStr, dbUser, dbPass, dbName,
	serverHost, serverPortStr string, tlsEnabled bool, tlsCert, tlsKey string,
) {
	ui.app.Events.Dispatch(events.Debugf("gui", "saveConfig called"))
	ui.log("Saving configuration...")

	// Update API config in memory
	ui.app.Config.API.APIKey = apiKey
	ui.app.Config.API.BaseURL = baseURL

	// Update Server config in memory
	ui.app.State.ServerHost = serverHost
	serverPort, _ := strconv.Atoi(serverPortStr)
	ui.app.State.ServerPort = serverPort
	ui.app.State.TLSEnabled = tlsEnabled
	ui.app.State.TLSCert = tlsCert
	ui.app.State.TLSKey = tlsKey

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
	ui.app.Events.Dispatch(events.Debugf("gui", "testDBConnection called"))
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
	ui.app.Events.Dispatch(events.Debugf("gui", "runSchemaEnforcement called"))
	if err := ui.app.DB.ValidateSchema(ui.app.State); err == nil {
		// Schema exists, confirm re-initialization
		dialog.ShowConfirm("Re-initialize Schema?", "This will delete all existing data. Are you sure?", func(ok bool) {
			if !ok {
				return
			}
			ui.log("Re-initializing database schema...")
			if err := ui.app.DB.EnforceSchema(ui.app.State); err != nil {
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
		if err := ui.app.DB.EnforceSchema(ui.app.State); err != nil {
			ui.log(fmt.Sprintf("ERROR: %v", err))
			ui.showToast("Error: Failed to initialize schema.")
			return
		}
		ui.log("Schema initialized successfully.")
		ui.showToast("Success: Schema initialized.")
		ui.refreshConfigTab()
	}
}
