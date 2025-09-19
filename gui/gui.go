package gui

import (
	"badgermaps/app"
	"badgermaps/app/push"
	"badgermaps/app/server"
	"badgermaps/database"
	"badgermaps/events"
	"fmt"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"image/color"
	"sort"
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
	app       *app.App
	fyneApp   fyne.App
	window    fyne.Window
	presenter *GuiPresenter

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
	a.Events.Dispatch(events.Infof("gui", "Waiting for database connection to settle..."))

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

	// Create and link the presenter
	presenter := NewGuiPresenter(a, ui)
	ui.presenter = presenter

	// Subscribe to events to refresh the events tab
	eventListener := func(e events.Event) {
		if ui.app.State.Debug {
			a.Events.Dispatch(events.Debugf("gui", "GUI received event: %s", e.Type.String()))
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
		var msg string
		switch e.Type {
		case events.LogEvent:
			logPayload, ok := e.Payload.(events.LogEventPayload)
			if !ok {
				return
			}
			msg = fmt.Sprintf("[%s] [%s] %s", logPayload.Level.String(), e.Source, logPayload.Message)
		case events.ActionStart:
			msg = fmt.Sprintf("Starting action for event '%s': %s", e.Source, e.Payload.(string))
		case events.ActionSuccess:
			msg = fmt.Sprintf("Action for event '%s' completed successfully: %s", e.Source, e.Payload.(string))
		case events.ActionError:
			msg = fmt.Sprintf("Action for event '%s' failed: %v", e.Source, e.Payload.(error))
		case events.Debug:
			if txt, ok := e.Payload.(string); ok {
				msg = fmt.Sprintf("DEBUG: %s", txt)
			}
		}

		if msg != "" {
			ui.logMutex.Lock()
			defer ui.logMutex.Unlock()
			lines := strings.Split(msg, "\n")
			for _, line := range lines {
				ui.logBinding.Append(line)
			}
			if ui.logView != nil {
				ui.logView.ScrollToBottom()
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
			ui.ShowToast(fmt.Sprintf("Pulling %s from API...", e.Source))
		case events.PullComplete:
			ui.ShowToast(fmt.Sprintf("Successfully pulled %s.", e.Source))
		case events.PullError:
			ui.ShowToast(fmt.Sprintf("Error pulling %s.", e.Source))
		case events.PullGroupStart:
			ui.ShowToast(fmt.Sprintf("Starting full pull for %s...", e.Source))
		case events.PullGroupComplete:
			ui.ShowToast(fmt.Sprintf("Successfully pulled all %s.", e.Source))
		case events.PullGroupError:
			ui.ShowToast(fmt.Sprintf("Error pulling all %s.", e.Source))
		}
	}
	a.Events.Subscribe(events.PullStart, pullNotificationListener)
	a.Events.Subscribe(events.PullComplete, pullNotificationListener)
	a.Events.Subscribe(events.PullError, pullNotificationListener)
	a.Events.Subscribe(events.PullGroupStart, pullNotificationListener)
	a.Events.Subscribe(events.PullGroupComplete, pullNotificationListener)
	a.Events.Subscribe(events.PullGroupError, pullNotificationListener)

	// Subscribe to connection status changes to refresh UI
	connectionListener := func(e events.Event) {
		ui.RefreshConfigTab()
		ui.RefreshHomeTab()
	}
	a.Events.Subscribe(events.ConnectionStatusChanged, connectionListener)

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

	// Define all tabs first
	homeTab := container.NewTabItemWithIcon("Home", theme.HomeIcon(), ui.createHomeTab())
	actionsTab := container.NewTabItemWithIcon("Actions", theme.ListIcon(), ui.createActionsTab())
	serverTab := container.NewTabItemWithIcon("Server", theme.ComputerIcon(), ui.createServerTab())
	configTab := container.NewTabItemWithIcon("Configuration", theme.SettingsIcon(), ui.createConfigTab())

	// Conditionally create content for tabs that depend on configuration
	var pullContent, pushContent, explorerContent fyne.CanvasObject
	if ui.app.API != nil && ui.app.API.IsConnected() && ui.app.DB != nil && ui.app.DB.IsConnected() {
		pullContent = ui.createPullTab()
		pushContent = ui.createPushTab()
		explorerContent = ui.createExplorerTab()
	} else {
		pullContent = ui.createDisabledTabView(configTab)
		pushContent = ui.createDisabledTabView(configTab)
		explorerContent = ui.createDisabledTabView(configTab)
	}

	pullTab := container.NewTabItemWithIcon("Pull", theme.DownloadIcon(), pullContent)
	pushTab := container.NewTabItemWithIcon("Push", theme.UploadIcon(), pushContent)
	explorerTab := container.NewTabItemWithIcon("Explorer", theme.FolderIcon(), explorerContent)

	tabs := []*container.TabItem{
		homeTab,
		pullTab,
		pushTab,
		actionsTab,
		explorerTab,
		serverTab,
		configTab,
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
		ui.ShowDetails(container.NewScroll(detailsLabel))
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

func (ui *Gui) createHomeTab() fyne.CanvasObject {
	// Config Status
	configValid := ui.app.API != nil && ui.app.DB != nil
	configStatusText := "Invalid"
	configColor := color.NRGBA{R: 200, G: 0, B: 0, A: 255} // Red
	if configValid {
		configStatusText = "Valid"
		configColor = color.NRGBA{R: 0, G: 200, B: 0, A: 255} // Green
	}
	configStatusLabel := canvas.NewText(configStatusText, configColor)

	// API Status
	apiConnected := ui.app.API != nil && ui.app.API.IsConnected()
	apiStatusText := "Not Connected"
	apiColor := color.NRGBA{R: 200, G: 0, B: 0, A: 255} // Red
	if apiConnected {
		apiStatusText = "Connected"
		apiColor = color.NRGBA{R: 0, G: 200, B: 0, A: 255} // Green
	}
	apiStatusLabel := canvas.NewText(apiStatusText, apiColor)

	// DB Status
	dbConnected := ui.app.DB != nil && ui.app.DB.IsConnected()
	dbStatusText := "Not Connected"
	dbColor := color.NRGBA{R: 200, G: 0, B: 0, A: 255} // Red
	if dbConnected {
		dbStatusText = "Connected"
		dbColor = color.NRGBA{R: 0, G: 200, B: 0, A: 255} // Green
	}
	dbStatusLabel := canvas.NewText(dbStatusText, dbColor)

	// Server Status
	_, serverRunning := server.GetServerStatus(ui.app)
	serverStatusText := "Stopped"
	if serverRunning {
		serverStatusText = "Running"
	}
	serverStatusLabel := widget.NewLabel(serverStatusText) // No color, it's just a state

	// Schema Status
	schemaValid := false
	if dbConnected {
		if err := ui.app.DB.ValidateSchema(ui.app.State); err == nil {
			schemaValid = true
		}
	}
	schemaStatusText := "Invalid"
	schemaColor := color.NRGBA{R: 200, G: 0, B: 0, A: 255} // Red
	if schemaValid {
		schemaStatusText = "Valid"
		schemaColor = color.NRGBA{R: 0, G: 200, B: 0, A: 255} // Green
	}
	schemaStatusLabel := canvas.NewText(schemaStatusText, schemaColor)

	statusGrid := container.NewGridWithColumns(2,
		container.NewCenter(widget.NewLabel("Configuration")),
		container.NewCenter(configStatusLabel),
		container.NewCenter(widget.NewLabel("API Status")),
		container.NewCenter(apiStatusLabel),
		container.NewCenter(widget.NewLabel("Database Status")),
		container.NewCenter(dbStatusLabel),
		container.NewCenter(widget.NewLabel("Server Status")),
		container.NewCenter(serverStatusLabel),
		container.NewCenter(widget.NewLabel("Database Schema")),
		container.NewCenter(schemaStatusLabel),
	)

	statusCard := widget.NewCard("Application Status", "", statusGrid)

	refreshButton := widget.NewButtonWithIcon("Refresh Status", theme.ViewRefreshIcon(), ui.presenter.HandleRefreshStatus)

	return container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("Welcome to BadgerMaps Sync", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusCard,
		refreshButton,
	))
}

// RefreshAllTabs rebuilds the main tabs, which is useful when connection status changes.
func (ui *Gui) RefreshAllTabs() {
	if ui.tabs == nil {
		return
	}

	var configTabItem *container.TabItem
	for _, tab := range ui.tabs.Items {
		if tab.Text == "Configuration" {
			configTabItem = tab
			break
		}
	}

	var pullContent, pushContent, explorerContent fyne.CanvasObject
	if ui.app.API != nil && ui.app.API.IsConnected() && ui.app.DB != nil && ui.app.DB.IsConnected() {
		pullContent = ui.createPullTab()
		pushContent = ui.createPushTab()
		explorerContent = ui.createExplorerTab()
	} else {
		pullContent = ui.createDisabledTabView(configTabItem)
		pushContent = ui.createDisabledTabView(configTabItem)
		explorerContent = ui.createDisabledTabView(configTabItem)
	}

	for _, tab := range ui.tabs.Items {
		switch tab.Text {
		case "Home":
			tab.Content = ui.createHomeTab()
		case "Pull":
			tab.Content = pullContent
		case "Push":
			tab.Content = pushContent
		case "Explorer":
			tab.Content = explorerContent
		case "Configuration":
			tab.Content = ui.createConfigTab()
		}
	}

	ui.tabs.Refresh()
}

func (ui *Gui) RefreshHomeTab() {
	if ui.tabs != nil {
		for _, tab := range ui.tabs.Items {
			if tab.Text == "Home" {
				tab.Content = ui.createHomeTab()
				break
			}
		}
		ui.tabs.Refresh()
	}
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

func (ui *Gui) createDisabledTabView(configTab *container.TabItem) fyne.CanvasObject {
	label := widget.NewLabel("API or Database not configured correctly.")
	label.Alignment = fyne.TextAlignCenter
	label.Wrapping = fyne.TextWrapWord

	button := widget.NewButton("Go to Configuration", func() {
		ui.tabs.Select(configTab)
	})

	return container.NewCenter(container.NewVBox(
		label,
		button,
	))
}

// ShowDetails updates the right-hand pane to show the provided details object.
func (ui *Gui) ShowDetails(details fyne.CanvasObject) {
	var text string
	if l, ok := details.(*widget.Label); ok {
		text = l.Text
	} else if e, ok := details.(*widget.Entry); ok {
		text = e.Text
	} else if s, ok := details.(*container.Scroll); ok {
		if l, ok := s.Content.(*widget.Label); ok {
			text = l.Text
		} else {
			// Not a label in a scroll, just show it.
			ui.detailsView = s
			ui.terminalVisible = false
			ui.rightPane.Objects[0] = ui.detailsView
			ui.rightPane.Refresh()
			return
		}
	} else {
		// Not a label or entry, just show it.
		ui.detailsView = container.NewScroll(details)
		ui.terminalVisible = false
		ui.rightPane.Objects[0] = ui.detailsView
		ui.rightPane.Refresh()
		return
	}

	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	ui.detailsView = container.NewScroll(label)
	ui.terminalVisible = false
	ui.rightPane.Objects[0] = ui.detailsView
	ui.rightPane.Refresh()
}

// createPullTab creates the content for the "Pull" tab
func (ui *Gui) createPullTab() fyne.CanvasObject {
	accountIDEntry := widget.NewEntry()
	accountIDEntry.SetPlaceHolder("Account ID")
	pullAccountButton := widget.NewButtonWithIcon("Pull Account", theme.DownloadIcon(), func() {
		ui.presenter.HandlePullAccount(accountIDEntry.Text)
	})

	checkinIDEntry := widget.NewEntry()
	checkinIDEntry.SetPlaceHolder("Check-in ID")
	pullCheckinButton := widget.NewButtonWithIcon("Pull Check-in", theme.DownloadIcon(), func() {
		ui.presenter.HandlePullCheckin(checkinIDEntry.Text)
	})

	routeIDEntry := widget.NewEntry()
	routeIDEntry.SetPlaceHolder("Route ID")
	pullRouteButton := widget.NewButtonWithIcon("Pull Route", theme.DownloadIcon(), func() {
		ui.presenter.HandlePullRoute(routeIDEntry.Text)
	})

	singlePullCard := widget.NewCard("Pull Single Item by ID", "", container.NewVBox(
		container.NewGridWithColumns(2, accountIDEntry, pullAccountButton),
		container.NewGridWithColumns(2, checkinIDEntry, pullCheckinButton),
		container.NewGridWithColumns(2, routeIDEntry, pullRouteButton),
	))

	pullAccountsButton := widget.NewButtonWithIcon("Pull All Accounts", theme.DownloadIcon(), ui.presenter.HandlePullAccounts)
	pullCheckinsButton := widget.NewButtonWithIcon("Pull All Check-ins", theme.DownloadIcon(), ui.presenter.HandlePullCheckins)
	pullRoutesButton := widget.NewButtonWithIcon("Pull All Routes", theme.DownloadIcon(), ui.presenter.HandlePullRoutes)
	pullProfileButton := widget.NewButtonWithIcon("Pull User Profile", theme.AccountIcon(), ui.presenter.HandlePullProfile)

	bulkPullCard := widget.NewCard("Pull Data Sets", "", container.NewVBox(
		pullAccountsButton,
		pullCheckinsButton,
		pullRoutesButton,
		pullProfileButton,
	))

	pullAllButton := widget.NewButtonWithIcon("Run Full Pull (All Data)", theme.ViewRefreshIcon(), ui.presenter.HandlePullGroup)

	return container.NewVScroll(container.NewVBox(
		singlePullCard,
		bulkPullCard,
		pullAllButton,
	))
}

// createPushTab creates the content for the "Push" tab
func (ui *Gui) createPushTab() fyne.CanvasObject {
	pushAccountsButton := widget.NewButtonWithIcon("Push Account Changes", theme.UploadIcon(), ui.presenter.HandlePushAccounts)
	pushCheckinsButton := widget.NewButtonWithIcon("Push Check-in Changes", theme.UploadIcon(), ui.presenter.HandlePushCheckins)
	pushAllButton := widget.NewButtonWithIcon("Push All Changes", theme.ViewRefreshIcon(), ui.presenter.HandlePushAll)

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

func (ui *Gui) RefreshPushTab() {
	if ui.tabs != nil {
		for _, tab := range ui.tabs.Items {
			if tab.Text == "Push" {
				tab.Content = ui.createPushTab()
				break
			}
		}
		ui.tabs.Refresh()
	}
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

		ui.ShowDetails(detailsEntry)
	}

	return dataTable
}

// createActionsTab creates the content for the "Actions" tab
func (ui *Gui) createActionsTab() fyne.CanvasObject {
	actionsContent := container.NewVBox()

	eventActions := ui.app.GetEventActions()
	sort.Slice(eventActions, func(i, j int) bool {
		return eventActions[i].Name < eventActions[j].Name
	})

	if len(eventActions) == 0 {
		actionsContent.Add(widget.NewLabel("No actions configured."))
	}

	for _, eventAction := range eventActions {
		ea := eventAction // Capture loop variable
		actionsContainer := container.NewVBox()
		for i, action := range ea.Run {
			ac := action
			idx := i
			var iconResource fyne.Resource
			var labelText string

			switch ac.Type {
			case "exec":
				iconResource = theme.FileApplicationIcon()
				labelText = fmt.Sprintf("Exec: %s", ac.Args["command"])
			case "db":
				iconResource = theme.StorageIcon()
				labelText = fmt.Sprintf("DB: %s", ac.Args["function"])
			case "api":
				iconResource = theme.ComputerIcon()
				labelText = fmt.Sprintf("API: %s", ac.Args["endpoint"])
			default:
				iconResource = theme.HelpIcon()
				labelText = "Unknown action"
			}

			label := widget.NewLabel(labelText)
			icon := widget.NewIcon(iconResource)

			toolbar := widget.NewToolbar(
				widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
					ui.app.ExecuteAction(ac)
				}),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
					ui.createActionPopup(&ea, idx)
				}),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(theme.DeleteIcon(), func() {
					dialog.ShowConfirm("Delete Action", "Are you sure you want to delete this action?", func(confirm bool) {
						if confirm {
							err := ui.app.RemoveEventAction(ea.Name, idx)
							if err != nil {
								ui.app.Events.Dispatch(events.Errorf("gui", "Error removing action: %v", err))
							}
						}
					}, ui.window)
				}),
			)
			actionsContainer.Add(container.NewBorder(nil, nil, icon, toolbar, label))
		}

		card := widget.NewCard(ea.Name, fmt.Sprintf("Event: %s, Source: %s", ea.Event, ea.Source), actionsContainer)
		actionsContent.Add(card)
	}

	addButton := widget.NewButtonWithIcon("Add Action", theme.ContentAddIcon(), func() {
		ui.createActionPopup(nil, -1)
	})

	return container.NewBorder(nil, addButton, nil, nil, container.NewVScroll(actionsContent))
}

func (ui *Gui) createActionPopup(eventAction *events.EventAction, actionIndex int) {
	var event, source string
	var actionConfig events.ActionConfig

	if eventAction != nil {
		event = eventAction.Event
		source = eventAction.Source
		if actionIndex != -1 {
			actionConfig = eventAction.Run[actionIndex]
		}
	}

	eventEntry := widget.NewSelect(events.AllEventTypes(), nil)
	eventEntry.SetSelected(event)
	sourceEntry := widget.NewSelect(events.AllEventSources(), nil)
	sourceEntry.SetSelected(source)

	// --- Exec Tab ---
	execCommandEntry := widget.NewEntry()
	execArgsEntry := widget.NewEntry()
	if actionConfig.Type == "exec" {
		if cmd, ok := actionConfig.Args["command"].(string); ok {
			execCommandEntry.SetText(cmd)
		}
		if args, ok := actionConfig.Args["args"].([]interface{}); ok {
			var argStrings []string
			for _, arg := range args {
				argStrings = append(argStrings, arg.(string))
			}
			execArgsEntry.SetText(strings.Join(argStrings, " "))
		}
	}
	execForm := widget.NewForm(
		widget.NewFormItem("Command", execCommandEntry),
		widget.NewFormItem("Args (space-separated)", execArgsEntry),
	)
	execTab := container.NewTabItemWithIcon("Exec", theme.FileApplicationIcon(), execForm)

	// --- DB Tab ---
	dbFunctionEntry := widget.NewEntry()
	if actionConfig.Type == "db" {
		if fn, ok := actionConfig.Args["function"].(string); ok {
			dbFunctionEntry.SetText(fn)
		}
	}
	dbForm := widget.NewForm(widget.NewFormItem("Function", dbFunctionEntry))
	dbTab := container.NewTabItemWithIcon("Database", theme.StorageIcon(), dbForm)

	// --- API Tab ---
	apiEndpointEntry := widget.NewEntry()
	apiMethodEntry := widget.NewSelect([]string{"GET", "POST", "PATCH", "DELETE"}, nil)
	apiDataEntry := widget.NewMultiLineEntry()
	apiDataEntry.SetPlaceHolder("key1=value1\nkey2=value2")

	if actionConfig.Type == "api" {
		if endpoint, ok := actionConfig.Args["endpoint"].(string); ok {
			apiEndpointEntry.SetText(endpoint)
		}
		if method, ok := actionConfig.Args["method"].(string); ok {
			apiMethodEntry.SetSelected(method)
		}
		if data, ok := actionConfig.Args["data"].(map[string]interface{}); ok {
			var dataStrings []string
			for k, v := range data {
				dataStrings = append(dataStrings, fmt.Sprintf("%s=%s", k, v))
			}
			apiDataEntry.SetText(strings.Join(dataStrings, "\n"))
		}
	}

	apiForm := widget.NewForm(
		widget.NewFormItem("Endpoint", apiEndpointEntry),
		widget.NewFormItem("Method", apiMethodEntry),
	)
	apiDataFormItem := widget.NewFormItem("Data", apiDataEntry)

	apiMethodEntry.OnChanged = func(method string) {
		if method == "POST" || method == "PATCH" {
			// Check if the item is already there
			found := false
			for _, item := range apiForm.Items {
				if item == apiDataFormItem {
					found = true
					break
				}
			}
			if !found {
				apiForm.AppendItem(apiDataFormItem)
			}
		} else {
			// Check if the item is there before trying to remove
			found := false
			for _, item := range apiForm.Items {
				if item == apiDataFormItem {
					found = true
					break
				}
			}
			if found {
				apiForm.Items = apiForm.Items[:2] // Keep only endpoint and method
			}
		}
		apiForm.Refresh()
	}
	// Trigger OnChanged to set initial state
	apiMethodEntry.OnChanged(apiMethodEntry.Selected)

	apiTab := container.NewTabItemWithIcon("API", theme.ComputerIcon(), apiForm)

	actionTabs := container.NewAppTabs(execTab, dbTab, apiTab)
	switch actionConfig.Type {
	case "db":
		actionTabs.Select(dbTab)
	case "api":
		actionTabs.Select(apiTab)
	default:
		actionTabs.Select(execTab)
	}

	dialogContent := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Event", eventEntry),
			widget.NewFormItem("Source", sourceEntry),
		),
		actionTabs,
	)

	d := dialog.NewCustomConfirm("Save Action", "Save", "Cancel", dialogContent, func(confirm bool) {
		if !confirm {
			return
		}

		var newAction events.ActionConfig
		newAction.Args = make(map[string]interface{})

		selectedTab := actionTabs.Selected()
		switch selectedTab.Text {
		case "Exec":
			newAction.Type = "exec"
			newAction.Args["command"] = execCommandEntry.Text
			newAction.Args["args"] = strings.Fields(execArgsEntry.Text)
		case "Database":
			newAction.Type = "db"
			newAction.Args["function"] = dbFunctionEntry.Text
		case "API":
			newAction.Type = "api"
			newAction.Args["endpoint"] = apiEndpointEntry.Text
			newAction.Args["method"] = apiMethodEntry.Selected
			if apiMethodEntry.Selected == "POST" || apiMethodEntry.Selected == "PATCH" {
				data := make(map[string]string)
				lines := strings.Split(apiDataEntry.Text, "\n")
				for _, line := range lines {
					if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
						data[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
					}
				}
				newAction.Args["data"] = data
			}
		}

		if eventAction == nil {
			err := ui.app.AddEventAction(eventEntry.Selected, sourceEntry.Selected, newAction)
			if err != nil {
				ui.app.Events.Dispatch(events.Errorf("gui", "Error adding action: %v", err))
			}
		} else {
			err := ui.app.UpdateEventAction(eventAction.Name, actionIndex, newAction)
			if err != nil {
				ui.app.Events.Dispatch(events.Errorf("gui", "Error updating action: %v", err))
			}
		}
	}, ui.window)

	d.Resize(fyne.NewSize(500, 400))
	d.Show()
}

// createExplorerTab creates the content for the "Explorer" tab
func (ui *Gui) createExplorerTab() fyne.CanvasObject {
	tableContainer := container.NewMax() // Use NewMax to fill available space
	tableSelect := widget.NewSelect([]string{}, func(tableName string) {
		ui.app.Events.Dispatch(events.Infof("gui", "Explorer: Loading table '%s'", tableName))
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
			ui.app.Events.Dispatch(events.Errorf("gui", "Error executing query: %v", err))
			tableContainer.Objects = []fyne.CanvasObject{widget.NewLabel(fmt.Sprintf("Error: %v", err))}
			tableContainer.Refresh()
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			ui.app.Events.Dispatch(events.Errorf("gui", "Error getting columns: %v", err))
			tableContainer.Objects = []fyne.CanvasObject{widget.NewLabel(fmt.Sprintf("Error: %v", err))}
			tableContainer.Refresh()
			return
		}
		ui.app.Events.Dispatch(events.Infof("gui", "Explorer: Found %d columns in '%s'", len(columns), tableName))

		var data [][]string
		for rows.Next() {
			row := make([]interface{}, len(columns))
			rowData := make([]string, len(columns))
			for i := range row {
				row[i] = new(interface{})
			}
			if err := rows.Scan(row...); err != nil {
				ui.app.Events.Dispatch(events.Errorf("gui", "Error scanning row: %v", err))
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
		ui.app.Events.Dispatch(events.Infof("gui", "Explorer: Found %d rows in '%s'", len(data), tableName))

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
			ui.ShowDetails(container.NewScroll(detailsLabel))
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
				ui.app.Events.Dispatch(events.Errorf("gui", "Error getting tables: %v", err))
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
			ui.app.Events.Dispatch(events.Errorf("gui", "Error getting tables: %v", err))
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
		ui.presenter.HandleStartServer()
		refreshServerStatus()
	}

	stopButton.OnTapped = func() {
		ui.presenter.HandleStopServer()
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

	apiIcon := theme.HelpIcon()
	if ui.app.API.IsConnected() {
		apiIcon = theme.ConfirmIcon()
	} else if ui.app.API.APIKey != "" { // If key exists but not connected, show error
		apiIcon = theme.ErrorIcon()
	}

	testApiButton := widget.NewButtonWithIcon("Test Connection", apiIcon, func() {
		ui.presenter.HandleTestAPIConnection(apiKeyEntry.Text, baseURLEntry.Text)
	})
	apiCard := widget.NewCard("API Configuration", "", container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("API Key", apiKeyEntry),
			widget.NewFormItem("Base URL", baseURLEntry),
		),
		testApiButton,
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

	dbIcon := theme.HelpIcon()
	if ui.app.DB.IsConnected() {
		dbIcon = theme.ConfirmIcon()
	} else if ui.app.DB.GetType() != "" {
		dbIcon = theme.ErrorIcon()
	}

	testDbButton := widget.NewButtonWithIcon("Test Connection", dbIcon, nil)
	testDbButton.OnTapped = func() {
		ui.presenter.HandleTestDBConnection(
			dbTypeSelect.Selected, dbPathEntry.Text, dbHostEntry.Text,
			dbPortEntry.Text, dbUserEntry.Text, dbPassEntry.Text, dbNameEntry.Text,
		)
	}

	dbCard := widget.NewCard("Database Configuration", "", container.NewVBox(
		container.NewGridWithColumns(2, widget.NewLabel("Database Type"), dbTypeSelect),
		dbForm,
		testDbButton,
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
	verboseCheck := widget.NewCheck("Verbose Logging", nil)
	verboseCheck.SetChecked(ui.app.State.Verbose)
	otherCard := widget.NewCard("Other Settings", "", verboseCheck)

	// Buttons
	saveButton := NewSecondaryButton("Save Configuration", theme.ConfirmIcon(), func() {
		ui.presenter.HandleSaveConfig(
			apiKeyEntry.Text, baseURLEntry.Text, dbTypeSelect.Selected, dbPathEntry.Text,
			dbHostEntry.Text, dbPortEntry.Text, dbUserEntry.Text, dbPassEntry.Text, dbNameEntry.Text,
			serverHostEntry.Text, serverPortEntry.Text, tlsEnabledCheck.Checked, tlsCertEntry.Text, tlsKeyEntry.Text,
			verboseCheck.Checked,
		)
	})

	viewButton := widget.NewButtonWithIcon("View", theme.VisibilityIcon(), ui.presenter.HandleViewConfig)

	// Schema Management
	schemaLabel := "Initialize Schema"
	if ui.app.DB != nil && ui.app.DB.IsConnected() {
		if err := ui.app.DB.ValidateSchema(ui.app.State); err == nil {
			schemaLabel = "Re-initialize Schema"
		}
	}
	schemaButton := widget.NewButtonWithIcon(schemaLabel, theme.StorageIcon(), ui.presenter.HandleSchemaEnforcement)

	schemaCard := widget.NewCard("Schema Management", "", schemaButton)

	topButtons := container.NewGridWithColumns(2, viewButton, saveButton)

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

// --- GuiView Implementation ---

// ShowToast displays a transient popup message in the bottom right of the window.
func (ui *Gui) ShowToast(content string) {
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

func (ui *Gui) ShowProgressBar(title string) {
	ui.progressTitle.SetText(title)
	ui.progressContainer.Show()
}

func (ui *Gui) HideProgressBar() {
	ui.progressContainer.Hide()
	ui.progressTitle.SetText("")
}

func (ui *Gui) SetProgress(value float64) {
	ui.progressBar.SetValue(value)
}

func (ui *Gui) ShowErrorDialog(err error) {
	dialog.ShowError(err, ui.window)
}

func (ui *Gui) ShowConfirmDialog(title, message string, callback func(bool)) {
	dialog.ShowConfirm(title, message, callback, ui.window)
}

func (ui *Gui) GetMainWindow() fyne.Window {
	return ui.window
}

// refreshConfigTab rebuilds and refreshes the configuration tab
func (ui *Gui) RefreshConfigTab() {
	newConfigTab := ui.buildConfigTab()
	if ui.tabs != nil {
		for _, tab := range ui.tabs.Items {
			if tab.Text == "Configuration" {
				tab.Content = container.NewVScroll(newConfigTab)
				break
			}
		}
		ui.tabs.Refresh()
	}
}

// WrappingLabel is a simple custom widget that wraps text.
type WrappingLabel struct {
	widget.BaseWidget
	label *widget.Label
}

// NewWrappingLabel creates a new WrappingLabel
func NewWrappingLabel(text string) *WrappingLabel {
	l := &WrappingLabel{
		label: widget.NewLabel(text),
	}
	l.label.Wrapping = fyne.TextWrapWord
	l.ExtendBaseWidget(l)
	return l
}

// CreateRenderer implements the Widget interface
func (l *WrappingLabel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(l.label)
}
