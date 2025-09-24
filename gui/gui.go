package gui

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	fapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"badgermaps/app"
	"badgermaps/app/action"
	"badgermaps/app/push"
	"badgermaps/database"
	"badgermaps/events"
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
	background := canvas.NewRectangle(theme.ButtonColor())
	background.CornerRadius = theme.InputRadiusSize()
	r := &secondaryButtonRenderer{
		button:     b,
		label:      widget.NewLabel(b.Text),
		icon:       widget.NewIcon(b.Icon),
		background: background,
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
	labelSize := r.label.MinSize()
	if r.button.Icon != nil {
		iconSize := theme.IconInlineSize()
		totalWidth := iconSize + padding + labelSize.Width
		if totalWidth > size.Width {
			totalWidth = size.Width
		}
		startX := (size.Width - totalWidth) / 2
		iconY := (size.Height - iconSize) / 2
		r.icon.Resize(fyne.NewSize(iconSize, iconSize))
		r.icon.Move(fyne.NewPos(startX, iconY))
		r.label.Move(fyne.NewPos(startX+iconSize+padding, (size.Height-labelSize.Height)/2))
	} else {
		r.label.Move(fyne.NewPos((size.Width-labelSize.Width)/2, (size.Height-labelSize.Height)/2))
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
	if r.button.Icon != nil {
		r.icon.SetResource(r.button.Icon)
		r.icon.Show()
	} else {
		r.icon.Hide()
	}
	r.background.FillColor = theme.ButtonColor()
	if r.button.Disabled() {
		r.background.FillColor = theme.DisabledButtonColor()
	}
	r.background.CornerRadius = theme.InputRadiusSize()
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

type backdropOverlay struct {
	widget.BaseWidget
	background *canvas.Rectangle
	onTapped   func()
}

func newBackdropOverlay(fill color.Color, tapped func()) *backdropOverlay {
	o := &backdropOverlay{
		background: canvas.NewRectangle(fill),
		onTapped:   tapped,
	}
	o.ExtendBaseWidget(o)
	return o
}

func (o *backdropOverlay) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(o.background)
}

func (o *backdropOverlay) MinSize() fyne.Size {
	return fyne.NewSize(0, 0)
}

func (o *backdropOverlay) Tapped(*fyne.PointEvent) {
	if o.onTapped != nil {
		o.onTapped()
	}
}

func (o *backdropOverlay) TappedSecondary(*fyne.PointEvent) {
	if o.onTapped != nil {
		o.onTapped()
	}
}

type logEntry struct {
	widget.BaseWidget
	label *widget.Label
	lines int
}

func newLogEntry() *logEntry {
	e := &logEntry{
		label: widget.NewLabel(""),
		lines: 1,
	}
	e.label.Wrapping = fyne.TextWrapWord
	e.ExtendBaseWidget(e)
	return e
}

func (e *logEntry) SetText(text string) {
	lines := strings.Split(text, "\n")
	if len(lines) > 3 {
		text = strings.Join(lines[:3], "\n") + "..."
		lines = lines[:3]
	}
	e.label.SetText(text)
	e.lines = len(lines)
	if e.lines < 1 {
		e.lines = 1
	}
}

func (e *logEntry) CreateRenderer() fyne.WidgetRenderer {
	renderer := &logEntryRenderer{
		entry:   e,
		objects: []fyne.CanvasObject{e.label},
	}
	return renderer
}

type logEntryRenderer struct {
	entry   *logEntry
	objects []fyne.CanvasObject
}

func (r *logEntryRenderer) Layout(size fyne.Size) {
	r.entry.label.Resize(size)
}

func (r *logEntryRenderer) MinSize() fyne.Size {
	labelMin := r.entry.label.MinSize()
	lineHeight := labelMin.Height
	if lineHeight <= 0 {
		lineHeight = float32(theme.TextSize())
	}
	lines := r.entry.lines + 1 // Allow an extra wrapped line for narrow layouts
	const maxLines = 5
	if lines > maxLines {
		lines = maxLines
	}
	height := lineHeight * float32(lines)
	height += theme.Padding()
	width := labelMin.Width
	if width <= 0 {
		width = float32(theme.TextSize()) * 10
	}
	return fyne.NewSize(width, height)
}

func (r *logEntryRenderer) Refresh() {
	r.entry.label.Refresh()
}

func (r *logEntryRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *logEntryRenderer) Destroy() {}

const rightPaneWidth float32 = 360

// Gui struct holds all the UI components and application state
type Gui struct {
	app       *app.App
	fyneApp   fyne.App
	window    fyne.Window
	presenter *GuiPresenter

	logMutex   sync.Mutex
	toastMutex sync.Mutex

	logBinding            binding.StringList
	logView               *widget.List
	detailsView           fyne.CanvasObject
	rightPaneContent      *fyne.Container
	rightPaneOverlay      *fyne.Container
	rightPaneBackdrop     fyne.CanvasObject
	rightPaneToggleButton *widget.Button
	rightPaneVisible      bool
	configTab             fyne.CanvasObject
	progressBar           *widget.ProgressBar
	progressContainer     *fyne.Container
	progressTitle         *widget.Label

	terminalVisible bool
	tabs            *container.AppTabs // Hold a reference to the tabs container

	// Explorer references for cross-navigation
	explorerTableSelect     *widget.Select
	explorerLoadPage        func(tableName string, page, pageSize int, opts ExplorerQueryOptions)
	explorerCurrentPageSize int
	explorerCurrentQuery    ExplorerQueryOptions
	explorerApplyQuery      func(opts ExplorerQueryOptions, reload bool)

	// New components
	syncCenter     *SyncCenter
	welcomeScreen  *WelcomeScreen
	smartDashboard *SmartDashboard
	tableFactory   *TableFactory
	showWelcome    bool
}

func (ui *Gui) themeColor(name fyne.ThemeColorName) color.Color {
	if ui == nil {
		return newModernTheme().Color(name, theme.VariantDark)
	}
	if ui.fyneApp == nil {
		return newModernTheme().Color(name, theme.VariantDark)
	}
	settings := ui.fyneApp.Settings()
	return settings.Theme().Color(name, settings.ThemeVariant())
}

func (ui *Gui) newSectionCard(title, subtitle string, content ...fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	body := container.NewVBox(titleLabel)
	if subtitle != "" {
		subtitleLabel := widget.NewLabel(subtitle)
		subtitleLabel.Alignment = fyne.TextAlignLeading
		subtitleLabel.Wrapping = fyne.TextWrapWord
		body.Add(subtitleLabel)
	}
	if len(content) > 0 {
		body.Add(widget.NewSeparator())
		for _, c := range content {
			if c != nil {
				body.Add(c)
			}
		}
	}
	background := canvas.NewRectangle(ui.themeColor(StatusCardBackgroundColorName))
	background.CornerRadius = theme.Padding()
	background.StrokeColor = ui.themeColor(StatusCardBorderColorName)
	background.StrokeWidth = 1

	return container.NewStack(
		background,
		container.NewPadded(body),
	)
}

// Launch initializes and runs the GUI
func Launch(a *app.App, icon fyne.Resource) {
	a.Events.Dispatch(events.Debugf("gui", "GUI initiated"))
	a.Events.Dispatch(events.Infof("gui", "Waiting for database connection to settle..."))

	fyneApp := fapp.New()
	fyneApp.SetIcon(icon)
	window := fyneApp.NewWindow("Badger Maps Sync")

	ui := &Gui{
		app:             a,
		fyneApp:         fyneApp,
		window:          window,
		logBinding:      binding.NewStringList(),
		terminalVisible: false, // Default to details view
	}

	ui.applyThemePreference()

	// Create and link the presenter
	presenter := NewGuiPresenter(a, ui)
	ui.presenter = presenter

	// Create new components
	ui.syncCenter = NewSyncCenter(ui, presenter)
	ui.smartDashboard = NewSmartDashboard(ui, presenter)

	// Initialize table factory for consistent table creation
	if ui.tableFactory == nil {
		ui.tableFactory = NewTableFactory(ui)
	}

	// Check if we should show welcome screen (first time setup or no config)
	ui.showWelcome = (a.API == nil || a.API.APIKey == "") || (a.DB == nil || a.DB.GetType() == "")

	// Subscribe to events to refresh the events tab
	eventListener := func(e events.Event) {
		if ui.app.State.Debug {
			a.Events.Dispatch(events.Debugf("gui", "GUI received event: %s", e.Type))
		}
		fyne.Do(func() {
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
		})
	}
	a.Events.Subscribe("action.config.*", eventListener)

	// Subscribe to logging and action events
	logListener := func(e events.Event) {
		var msg string
		switch e.Type {
		case "log":
			logPayload, ok := e.Payload.(events.LogPayload)
			if !ok {
				return
			}
			msg = fmt.Sprintf("[%s] [%s] %s", logPayload.Level.String(), e.Source, logPayload.Message)
		}

		if msg != "" {
			lines := strings.Split(msg, "\n")
			fyne.Do(func() {
				ui.logMutex.Lock()
				defer ui.logMutex.Unlock()
				for _, line := range lines {
					ui.logBinding.Append(line)
				}
				if ui.logView != nil {
					ui.logView.ScrollToBottom()
				}
			})
		}
	}
	a.Events.Subscribe("log", logListener)

	// Subscribe to pull events to show notifications
	pullNotificationListener := func(e events.Event) {
		switch e.Type {
		case "pull.start":
			fyne.Do(func() {
				ui.ShowToast(fmt.Sprintf("Pulling %s from API...", e.Source))
			})
		case "pull.complete":
			fyne.Do(func() {
				ui.ShowToast(fmt.Sprintf("Successfully pulled %s.", e.Source))
			})
		case "pull.error":
			fyne.Do(func() {
				ui.ShowToast(fmt.Sprintf("Error pulling %s.", e.Source))
			})
		case "pull.group.start":
			fyne.Do(func() {
				ui.ShowToast(fmt.Sprintf("Starting full pull for %s...", e.Source))
			})
		case "pull.group.complete":
			fyne.Do(func() {
				ui.ShowToast(fmt.Sprintf("Successfully pulled all %s.", e.Source))
			})
		case "pull.group.error":
			fyne.Do(func() {
				ui.ShowToast(fmt.Sprintf("Error pulling all %s.", e.Source))
			})
		}
	}
	a.Events.Subscribe("pull.*", pullNotificationListener)

	// Subscribe to connection status changes to refresh UI
	connectionListener := func(e events.Event) {
		fyne.Do(func() {
			ui.RefreshConfigTab()
			ui.RefreshHomeTab()
		})
	}
	a.Events.Subscribe("connection.status.changed", connectionListener)

	window.SetContent(ui.createContent())
	// Set more reasonable default size and allow proper resizing
	window.Resize(fyne.NewSize(1000, 600))
	window.SetFixedSize(false) // Allow resizing
	window.CenterOnScreen()
	window.ShowAndRun()
}

func (ui *Gui) applyThemePreference() {
	if ui.fyneApp == nil {
		return
	}

	preference := app.NormalizeThemePreference(ui.app.Config.ThemePreference)
	var selectedTheme fyne.Theme
	switch preference {
	case app.ThemePreferenceLight:
		selectedTheme = newModernThemeForVariant(theme.VariantLight)
	case app.ThemePreferenceDark:
		selectedTheme = newModernThemeForVariant(theme.VariantDark)
	default:
		selectedTheme = newModernTheme()
	}

	ui.fyneApp.Settings().SetTheme(selectedTheme)
}

func (ui *Gui) ApplyThemePreference(pref string) {
	ui.app.Config.ThemePreference = app.NormalizeThemePreference(pref)
	ui.applyThemePreference()
}

// createContent builds the main content of the window
func (ui *Gui) createContent() fyne.CanvasObject {
	if ui.showWelcome {
		// Show welcome screen on first launch
		ui.welcomeScreen = NewWelcomeScreen(ui.app, ui.presenter, func() {
			// When welcome is complete, switch to main content
			ui.showWelcome = false
			ui.window.SetContent(ui.createMainContent())
		})
		return ui.welcomeScreen.CreateContent()
	}
	return ui.createMainContent()
}

// createMainContent builds the main layout with toolbar, tabs, and log view
func (ui *Gui) createMainContent() fyne.CanvasObject {
	ui.configTab = ui.buildConfigTab()

	// Define all tabs first
	homeTab := container.NewTabItemWithIcon("Home", theme.HomeIcon(), ui.createHomeTab())
	configTab := container.NewTabItemWithIcon("Configuration", theme.SettingsIcon(), ui.createConfigTab())

	// Conditionally create content for tabs that depend on configuration
	var syncContent, explorerContent fyne.CanvasObject
	if ui.app.API != nil && ui.app.API.IsConnected() && ui.app.DB != nil && ui.app.DB.IsConnected() {
		syncContent = ui.syncCenter.CreateContent()
		explorerContent = ui.createExplorerTab()
	} else {
		syncContent = ui.createDisabledTabView(configTab)
		explorerContent = ui.createDisabledTabView(configTab)
	}

	syncTab := container.NewTabItemWithIcon("Sync Center", theme.DownloadIcon(), syncContent)
	explorerTab := container.NewTabItemWithIcon("Explorer", theme.FolderIcon(), explorerContent)
	actionsTab := container.NewTabItemWithIcon("Actions", theme.ViewRefreshIcon(), ui.createActionsTab())
	serverTab := container.NewTabItemWithIcon("Server", theme.ComputerIcon(), ui.createServerTab())

	tabs := []*container.TabItem{
		homeTab,
		syncTab,
		explorerTab,
		actionsTab,
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
	ui.rightPaneContent = container.NewMax(ui.detailsView)

	rightPanePanel := container.NewBorder(
		ui.createRightPaneHeader(), nil, nil, nil,
		ui.rightPaneContent,
	)
	panelWithPadding := container.NewPadded(rightPanePanel)
	panelBackground := canvas.NewRectangle(theme.BackgroundColor())
	//panelBackground.CornerRadius = theme.Padding()
	panelBackground.StrokeColor = theme.SeparatorColor()
	panelBackground.StrokeWidth = 1
	panelWrapper := container.NewStack(panelBackground, panelWithPadding)

	ui.rightPaneBackdrop = newBackdropOverlay(color.NRGBA{R: 0, G: 0, B: 0, A: 144}, ui.hideRightPane)
	ui.rightPaneBackdrop.Hide()

	ui.rightPaneOverlay = container.New(&slideOverLayout{
		panelWidth:   rightPaneWidth,
		panelPadding: theme.Padding(),
	}, ui.rightPaneBackdrop, panelWrapper)
	ui.rightPaneOverlay.Hide()
	ui.rightPaneVisible = false

	toggleButton := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		ui.toggleRightPane()
	})
	toggleButton.Importance = widget.LowImportance
	ui.rightPaneToggleButton = toggleButton
	floatingToggle := container.New(&floatingButtonLayout{padding: theme.Padding()}, toggleButton)
	ui.updateRightPaneToggle()

	if ui.syncCenter != nil {
		ui.syncCenter.applyStoredDetail()
	}

	return container.NewStack(mainContent, ui.rightPaneOverlay, floatingToggle)
}

type slideOverLayout struct {
	panelWidth   float32
	panelPadding float32
}

func (l *slideOverLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	if backdrop := objects[0]; backdrop != nil {
		backdrop.Resize(size)
		backdrop.Move(fyne.NewPos(0, 0))
	}
	panelWidth := l.panelWidth
	if panelWidth <= 0 || panelWidth > size.Width {
		panelWidth = size.Width
	}
	x := size.Width - panelWidth - l.panelPadding
	if x < 0 {
		x = 0
	}
	for i := 1; i < len(objects); i++ {
		obj := objects[i]
		if obj == nil {
			continue
		}
		obj.Resize(fyne.NewSize(panelWidth, size.Height))
		obj.Move(fyne.NewPos(x, 0))
	}
}

func (l *slideOverLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(l.panelWidth+l.panelPadding, 0)
}

type floatingButtonLayout struct {
	padding float32
}

func (l *floatingButtonLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		if obj == nil {
			continue
		}
		min := obj.MinSize()
		width := min.Width
		height := min.Height
		if width > size.Width {
			width = size.Width
		}
		if height > size.Height {
			height = size.Height
		}
		x := size.Width - width - l.padding
		if x < l.padding {
			x = l.padding
		}
		y := l.padding
		if y+height > size.Height {
			y = size.Height - height
			if y < 0 {
				y = 0
			}
		}
		obj.Resize(fyne.NewSize(width, height))
		obj.Move(fyne.NewPos(x, y))
	}
}

func (l *floatingButtonLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var maxWidth, maxHeight float32
	for _, obj := range objects {
		if obj == nil {
			continue
		}
		min := obj.MinSize()
		w := min.Width + l.padding*2
		h := min.Height + l.padding*2
		if w > maxWidth {
			maxWidth = w
		}
		if h > maxHeight {
			maxHeight = h
		}
	}
	return fyne.NewSize(maxWidth, maxHeight)
}

func (ui *Gui) createHomeTab() fyne.CanvasObject {
	// Use the new smart dashboard
	if ui.smartDashboard != nil {
		return ui.smartDashboard.CreateContent()
	}

	// Fallback to original home tab if smart dashboard not initialized
	// Config Status
	configValid := ui.app.API != nil && ui.app.DB != nil
	configStatusText := "Invalid"
	configColor := theme.ErrorColor()
	if configValid {
		configStatusText = "Valid"
		configColor = theme.PrimaryColor()
	}
	configStatusLabel := canvas.NewText(configStatusText, configColor)

	// API Status
	apiConnected := ui.app.API != nil && ui.app.API.IsConnected()
	apiStatusText := "Not Connected"
	apiColor := theme.ErrorColor()
	if apiConnected {
		apiStatusText = "Connected"
		apiColor = theme.PrimaryColor()
	}
	apiStatusLabel := canvas.NewText(apiStatusText, apiColor)

	// DB Status
	dbConnected := ui.app.DB != nil && ui.app.DB.IsConnected()
	dbStatusText := "Not Connected"
	dbColor := theme.ErrorColor()
	if dbConnected {
		dbStatusText = "Connected"
		dbColor = theme.PrimaryColor()
	}
	dbStatusLabel := canvas.NewText(dbStatusText, dbColor)

	// Server Status
	_, serverRunning := ui.app.Server.GetServerStatus()
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
	schemaColor := theme.ErrorColor()
	if schemaValid {
		schemaStatusText = "Valid"
		schemaColor = theme.PrimaryColor()
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

	body := container.NewVBox(
		widget.NewLabelWithStyle("Welcome to BadgerMaps Sync", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusCard,
	)

	return container.NewBorder(
		nil,
		container.NewCenter(refreshButton),
		nil,
		nil,
		container.NewVScroll(body),
	)
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

	var syncContent, explorerContent fyne.CanvasObject
	if ui.app.API != nil && ui.app.API.IsConnected() && ui.app.DB != nil && ui.app.DB.IsConnected() {
		syncContent = ui.syncCenter.CreateContent()
		explorerContent = ui.createExplorerTab()
	} else {
		syncContent = ui.createDisabledTabView(configTabItem)
		explorerContent = ui.createDisabledTabView(configTabItem)
	}

	for _, tab := range ui.tabs.Items {
		switch tab.Text {
		case "Home":
			tab.Content = ui.createHomeTab()
		case "Sync Center":
			tab.Content = syncContent
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
	detailsButton := widget.NewButtonWithIcon("Details", theme.ListIcon(), func() {
		ui.terminalVisible = false
		if ui.detailsView != nil {
			ui.setRightPaneContent(ui.detailsView)
		}
		ui.showRightPane()
	})

	logButton := widget.NewButtonWithIcon("Log", theme.ComputerIcon(), func() {
		ui.terminalVisible = true
		ui.setRightPaneContent(ui.logView)
		ui.showRightPane()
	})

	buttonRow := container.NewHBox(detailsButton, logButton)
	return container.NewBorder(nil, nil, nil, nil, buttonRow)
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

func (ui *Gui) toggleRightPane() {
	if ui.rightPaneVisible {
		ui.hideRightPane()
		return
	}
	if ui.rightPaneContent == nil {
		return
	}
	if len(ui.rightPaneContent.Objects) == 0 {
		ui.setRightPaneContent(ui.detailsView)
	}
	ui.showRightPane()
}

func (ui *Gui) showRightPane() {
	if ui.rightPaneOverlay == nil {
		return
	}
	if ui.rightPaneBackdrop != nil {
		ui.rightPaneBackdrop.Show()
	}
	ui.rightPaneOverlay.Show()
	ui.rightPaneOverlay.Refresh()
	ui.rightPaneVisible = true
	ui.updateRightPaneToggle()
}

func (ui *Gui) hideRightPane() {
	if ui.rightPaneOverlay == nil {
		return
	}
	ui.rightPaneOverlay.Hide()
	if ui.rightPaneBackdrop != nil {
		ui.rightPaneBackdrop.Hide()
	}
	ui.rightPaneVisible = false
	ui.updateRightPaneToggle()
}

func (ui *Gui) setRightPaneContent(content fyne.CanvasObject) {
	if ui.rightPaneContent == nil || content == nil {
		return
	}
	ui.rightPaneContent.Objects = []fyne.CanvasObject{content}
	ui.rightPaneContent.Refresh()
}

// ShowDetails updates the right-hand pane to show the provided details object.
func (ui *Gui) ShowDetails(details fyne.CanvasObject) {
	ui.setDetails(details, true)
}

func (ui *Gui) updateRightPaneToggle() {
	if ui.rightPaneToggleButton == nil {
		return
	}
	if ui.rightPaneVisible {
		ui.rightPaneToggleButton.SetIcon(theme.NavigateNextIcon())
	} else {
		ui.rightPaneToggleButton.SetIcon(theme.ListIcon())
	}
	ui.rightPaneToggleButton.Refresh()
}

func (ui *Gui) setDetails(details fyne.CanvasObject, reveal bool) {
	display := ui.formatDetails(details)
	if display == nil {
		return
	}
	ui.detailsView = display
	ui.terminalVisible = false
	ui.setRightPaneContent(ui.detailsView)
	if reveal {
		ui.showRightPane()
	} else {
		ui.updateRightPaneToggle()
	}
}

func (ui *Gui) formatDetails(details fyne.CanvasObject) fyne.CanvasObject {
	if details == nil {
		return nil
	}

	switch d := details.(type) {
	case *widget.Label:
		label := widget.NewLabel(d.Text)
		label.Wrapping = fyne.TextWrapWord
		return container.NewScroll(label)
	case *widget.Entry:
		label := widget.NewLabel(d.Text)
		label.Wrapping = fyne.TextWrapWord
		return container.NewScroll(label)
	case *container.Scroll:
		if l, ok := d.Content.(*widget.Label); ok {
			label := widget.NewLabel(l.Text)
			label.Wrapping = fyne.TextWrapWord
			return container.NewScroll(label)
		}
		return d
	default:
		return container.NewScroll(details)
	}
}

// createPullTab creates the content for the "Pull" tab
func (ui *Gui) createPullTab() fyne.CanvasObject {
	omniEntry := widget.NewEntry()
	omniEntry.SetPlaceHolder("Search by name or ID…")
	omniScope := widget.NewSelect([]string{"All", "Accounts", "Check-ins", "Routes"}, nil)
	omniScope.SetSelected("All")
	omniSearchButton := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		ui.presenter.HandleOmniSearch(omniEntry.Text, omniScope.Selected)
	})
	omniConfigButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		ui.ShowToast("Omnibox configuration not yet implemented.")
	})
	omniBox := container.NewBorder(nil, nil, omniScope, container.NewHBox(omniSearchButton, omniConfigButton), omniEntry)

	searchCard := widget.NewCard("Omnibox Search", "", container.NewVBox(
		omniBox,
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
		searchCard,
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

	eventActions := ui.app.Config.EventActions
	sort.Slice(eventActions, func(i, j int) bool {
		return eventActions[i].Name < eventActions[j].Name
	})

	if len(eventActions) == 0 {
		empty := ui.newSectionCard(
			"Actions",
			"No event actions configured yet.",
			widget.NewLabel("Use the button below to add automation."),
		)
		actionsContent.Add(empty)
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
				labelText = func() string {
					if ac.Args == nil {
						return "DB action"
					}
					if cmd, ok := ac.Args["command"].(string); ok && cmd != "" {
						return fmt.Sprintf("DB command: %s", cmd)
					}
					if fn, ok := ac.Args["function"].(string); ok && fn != "" {
						return fmt.Sprintf("DB function: %s", fn)
					}
					if proc, ok := ac.Args["procedure"].(string); ok && proc != "" {
						return fmt.Sprintf("DB procedure: %s", proc)
					}
					if query, ok := ac.Args["query"].(string); ok && query != "" {
						trimmed := strings.TrimSpace(query)
						runes := []rune(trimmed)
						if len(runes) > 32 {
							trimmed = string(runes[:32]) + "..."
						}
						return fmt.Sprintf("DB query: %s", trimmed)
					}
					return "DB action"
				}()
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

		if len(actionsContainer.Objects) == 0 {
			actionsContainer.Add(widget.NewLabel("No steps configured for this action."))
		}

		friendlyEvent := formatEventName(ea.Event)
		friendlySource := "Any"
		if strings.TrimSpace(ea.Source) != "" {
			friendlySource = formatEventName(ea.Source)
		}

		cardTitle := friendlyEvent
		subtitle := fmt.Sprintf("Source: %s", friendlySource)

		card := ui.newSectionCard(
			cardTitle,
			subtitle,
			actionsContainer,
		)
		actionsContent.Add(card)
	}

	addButton := widget.NewButtonWithIcon("Add Action", theme.ContentAddIcon(), func() {
		ui.createActionPopup(nil, -1)
	})

	return container.NewBorder(nil, addButton, nil, nil, container.NewVScroll(actionsContent))
}

func (ui *Gui) createActionPopup(eventAction *action.EventAction, actionIndex int) {
	var event, source string
	var actionConfig action.ActionConfig

	if eventAction != nil {
		event = eventAction.Event
		source = eventAction.Source
		if actionIndex != -1 {
			actionConfig = eventAction.Run[actionIndex]
		}
	}

	eventEntry := widget.NewSelectEntry(events.AllEventTypes())
	eventEntry.SetPlaceHolder("e.g. pull.complete")
	if event != "" {
		eventEntry.SetText(event)
	}
	sourceEntry := widget.NewSelectEntry(events.AllEventSources())
	sourceEntry.SetPlaceHolder("Leave blank for any source")
	if source != "" {
		sourceEntry.SetText(source)
	}

	optionLookup := make(map[string]events.EventTokenOption)
	tokenSelect := widget.NewSelect([]string{}, nil)
	tokenSelect.PlaceHolder = "Insert event token"

	var currentTarget *tokenInsertionTarget

	refreshTokenOptions := func() {
		eventValue := strings.TrimSpace(eventEntry.Text)
		sourceValue := strings.TrimSpace(sourceEntry.Text)
		tokenOptions := events.EventTokenOptions(eventValue, sourceValue)
		tokenLabels := make([]string, 0, len(tokenOptions))
		optionLookup = make(map[string]events.EventTokenOption, len(tokenOptions))
		for _, opt := range tokenOptions {
			tokenLabels = append(tokenLabels, opt.Label)
			optionLookup[opt.Label] = opt
		}
		tokenSelect.Options = tokenLabels
		tokenSelect.ClearSelected()
		tokenSelect.Refresh()
	}

	eventEntry.OnChanged = func(string) {
		refreshTokenOptions()
	}
	sourceEntry.OnChanged = func(string) {
		refreshTokenOptions()
	}

	refreshTokenOptions()
	registerTarget := func(entry *widget.Entry) *tokenInsertionTarget {
		target := &tokenInsertionTarget{entry: entry}
		entry.OnCursorChanged = func() {
			target.row = entry.CursorRow
			target.col = entry.CursorColumn
			target.hasCursor = true
			currentTarget = target
		}
		return target
	}

	var defaultTarget *tokenInsertionTarget

	// --- Exec Tab ---
	execCommandEntry := widget.NewEntry()
	execArgsEntry := widget.NewEntry()
	execCommandTarget := registerTarget(execCommandEntry)
	registerTarget(execArgsEntry)
	defaultTarget = execCommandTarget
	if currentTarget == nil {
		currentTarget = execCommandTarget
	}
	if actionConfig.Type == "exec" {
		if cmd, ok := actionConfig.Args["command"].(string); ok {
			execCommandEntry.SetText(cmd)
		}
		if args, ok := actionConfig.Args["args"].([]interface{}); ok && len(args) > 0 {
			var argStrings []string
			for _, arg := range args {
				argStrings = append(argStrings, fmt.Sprintf("%v", arg))
			}
			execCommandEntry.SetText(execCommandEntry.Text + " " + strings.Join(argStrings, " "))
		}
	}
	execForm := widget.NewForm(
		widget.NewFormItem("Command", execCommandEntry),
		widget.NewFormItem("Args (space-separated)", execArgsEntry),
	)
	execTab := container.NewTabItemWithIcon("Exec", theme.FileApplicationIcon(), execForm)

	// --- DB Tab ---
	dbCommandEntry := widget.NewEntry()
	dbFunctionEntry := widget.NewEntry()
	dbProcedureEntry := widget.NewEntry()
	dbQueryEntry := widget.NewMultiLineEntry()
	registerTarget(dbCommandEntry)
	registerTarget(dbFunctionEntry)
	registerTarget(dbProcedureEntry)
	registerTarget(dbQueryEntry)
	dbQueryEntry.SetPlaceHolder("SELECT ...")

	dbActionType := "command"
	if actionConfig.Type == "db" && actionConfig.Args != nil {
		if cmd, ok := actionConfig.Args["command"].(string); ok && cmd != "" {
			dbActionType = "command"
			dbCommandEntry.SetText(cmd)
		} else if fn, ok := actionConfig.Args["function"].(string); ok && fn != "" {
			dbActionType = "function"
			dbFunctionEntry.SetText(fn)
		} else if proc, ok := actionConfig.Args["procedure"].(string); ok && proc != "" {
			dbActionType = "procedure"
			dbProcedureEntry.SetText(proc)
		} else if query, ok := actionConfig.Args["query"].(string); ok && query != "" {
			dbActionType = "query"
			dbQueryEntry.SetText(query)
		}
	}

	dbActionTypeRadio := widget.NewRadioGroup([]string{"command", "function", "procedure", "query"}, nil)
	dbActionTypeRadio.Required = true
	dbActionTypeRadio.Horizontal = true

	dbForms := map[string]*widget.Form{
		"command":   widget.NewForm(widget.NewFormItem("Command Key", dbCommandEntry)),
		"function":  widget.NewForm(widget.NewFormItem("Function Name", dbFunctionEntry)),
		"procedure": widget.NewForm(widget.NewFormItem("Procedure Name", dbProcedureEntry)),
		"query":     widget.NewForm(widget.NewFormItem("SQL Query", dbQueryEntry)),
	}

	dbInputContainer := container.NewMax()
	showDbInput := func(selection string) {
		form, ok := dbForms[selection]
		if !ok {
			form = dbForms["command"]
		}
		dbInputContainer.Objects = []fyne.CanvasObject{form}
		dbInputContainer.Refresh()
	}

	dbActionTypeRadio.SetSelected(dbActionType)
	showDbInput(dbActionType)
	dbActionTypeRadio.OnChanged = func(selection string) {
		showDbInput(selection)
	}

	dbTabContent := container.NewVBox(
		widget.NewForm(widget.NewFormItem("Action Type", dbActionTypeRadio)),
		dbInputContainer,
	)
	dbTab := container.NewTabItemWithIcon("Database", theme.StorageIcon(), dbTabContent)

	// --- API Tab ---
	apiEndpointEntry := widget.NewEntry()
	apiMethodEntry := widget.NewSelect([]string{"GET", "POST", "PATCH", "DELETE"}, nil)
	apiDataEntry := widget.NewMultiLineEntry()
	registerTarget(apiEndpointEntry)
	registerTarget(apiDataEntry)
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

	performInsert := func(token string) {
		if token == "" {
			return
		}
		target := currentTarget
		if target == nil {
			target = defaultTarget
		}
		if target == nil || target.entry == nil {
			return
		}
		insertTokenIntoEntry(target, token)
		tokenSelect.ClearSelected()
	}

	insertButton := widget.NewButtonWithIcon("Insert Token", theme.ContentAddIcon(), func() {
		selected := tokenSelect.Selected
		opt, ok := optionLookup[selected]
		if !ok {
			return
		}
		if opt.RequiresPath {
			entry := widget.NewEntry()
			if opt.Placeholder != "" {
				entry.SetPlaceHolder(opt.Placeholder)
			}
			dlg := dialog.NewForm("Insert Payload Field", "Insert", "Cancel", []*widget.FormItem{
				widget.NewFormItem("Field path", entry),
			}, func(confirm bool) {
				if !confirm {
					return
				}
				path := strings.TrimSpace(entry.Text)
				if path == "" {
					return
				}
				token := fmt.Sprintf(opt.Format, path)
				performInsert(token)
			}, ui.window)
			dlg.Resize(fyne.NewSize(420, 0))
			dlg.Show()
			return
		}
		token := opt.Token
		if token == "" && opt.Format != "" {
			token = fmt.Sprintf(opt.Format, "")
		}
		performInsert(token)
	})

	tokenControls := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewLabel("Event tokens"),
			container.NewAdaptiveGrid(2, tokenSelect, insertButton),
		),
	)

	actionTabs := container.NewAppTabs(execTab, dbTab, apiTab)
	switch actionConfig.Type {
	case "db":
		actionTabs.Select(dbTab)
	case "api":
		actionTabs.Select(apiTab)
	default:
		actionTabs.Select(execTab)
	}

	dialogBody := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Event", eventEntry),
			widget.NewFormItem("Source", sourceEntry),
		),
		actionTabs,
	)

	dialogContent := container.NewBorder(nil, tokenControls, nil, nil, dialogBody)

	d := dialog.NewCustomConfirm("Save Action", "Save", "Cancel", dialogContent, func(confirm bool) {
		if !confirm {
			return
		}

		var newAction action.ActionConfig
		newAction.Args = make(map[string]interface{})

		selectedTab := actionTabs.Selected()
		switch selectedTab.Text {
		case "Exec":
			newAction.Type = "exec"
			command := execCommandEntry.Text
			if execArgsEntry.Text != "" {
				command += " " + execArgsEntry.Text
			}
			newAction.Args["command"] = command
		case "Database":
			newAction.Type = "db"
			if actionConfig.Type == "db" && actionConfig.Args != nil {
				for k, v := range actionConfig.Args {
					newAction.Args[k] = v
				}
			}
			for _, key := range []string{"command", "function", "procedure", "query"} {
				delete(newAction.Args, key)
			}
			selectedKind := dbActionTypeRadio.Selected
			switch selectedKind {
			case "command":
				if val := strings.TrimSpace(dbCommandEntry.Text); val != "" {
					newAction.Args["command"] = val
				}
			case "function":
				if val := strings.TrimSpace(dbFunctionEntry.Text); val != "" {
					newAction.Args["function"] = val
				}
			case "procedure":
				if val := strings.TrimSpace(dbProcedureEntry.Text); val != "" {
					newAction.Args["procedure"] = val
				}
			case "query":
				if val := strings.TrimSpace(dbQueryEntry.Text); val != "" {
					newAction.Args["query"] = val
				}
			}
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

		eventValue := strings.TrimSpace(eventEntry.Text)
		if eventValue == "" {
			ui.app.Events.Dispatch(events.Warningf("gui", "Event type is required"))
			return
		}
		sourceValue := strings.TrimSpace(sourceEntry.Text)

		if newAction.Type == "db" {
			hasSpecifier := false
			for _, key := range []string{"command", "function", "procedure", "query"} {
				if _, ok := newAction.Args[key]; ok {
					hasSpecifier = true
					break
				}
			}
			if !hasSpecifier {
				ui.app.Events.Dispatch(events.Warningf("gui", "Select a database action type and provide a value"))
				return
			}
		}

		if eventAction == nil {
			err := ui.app.AddEventAction(eventValue, sourceValue, newAction)
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

type tokenInsertionTarget struct {
	entry     *widget.Entry
	row       int
	col       int
	hasCursor bool
}

func insertTokenIntoEntry(target *tokenInsertionTarget, token string) {
	if target == nil || target.entry == nil || token == "" {
		return
	}

	entry := target.entry
	currentText := entry.Text
	currentRunes := []rune(currentText)

	insertIndex := len(currentRunes)
	if target.hasCursor {
		insertIndex = runeIndexForCursor(currentText, target.row, target.col)
	}
	if insertIndex < 0 || insertIndex > len(currentRunes) {
		insertIndex = len(currentRunes)
	}

	before := string(currentRunes[:insertIndex])
	after := string(currentRunes[insertIndex:])
	newText := before + token + after
	entry.SetText(newText)

	newIndex := insertIndex + len([]rune(token))
	newRow, newCol := cursorForIndex(newText, newIndex)
	entry.CursorRow = newRow
	entry.CursorColumn = newCol
	entry.Refresh()

	target.row = newRow
	target.col = newCol
	target.hasCursor = true
}

func runeIndexForCursor(text string, row, col int) int {
	if row < 0 || col < 0 {
		return len([]rune(text))
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return 0
	}
	if row >= len(lines) {
		row = len(lines) - 1
	}
	index := 0
	for i := 0; i < row; i++ {
		index += len([]rune(lines[i]))
		index++ // account for newline
	}
	lineRunes := []rune(lines[row])
	if col > len(lineRunes) {
		col = len(lineRunes)
	}
	index += col
	return index
}

func cursorForIndex(text string, index int) (int, int) {
	runes := []rune(text)
	if index < 0 {
		index = 0
	}
	if index > len(runes) {
		index = len(runes)
	}
	row, col := 0, 0
	for i := 0; i < index; i++ {
		if runes[i] == '\n' {
			row++
			col = 0
		} else {
			col++
		}
	}
	return row, col
}

// createExplorerTab creates the content for the "Explorer" tab with pagination support
func (ui *Gui) createExplorerTab() fyne.CanvasObject {
	tableContainer := container.NewMax() // Use NewMax to fill available space

	// Pagination state
	var (
		currentTableName     string
		currentPaginatedData *PaginatedTableData
		searchEntry          *widget.Entry
		pageInfoLabel        *widget.Label
		prevBtn, nextBtn     *widget.Button
		firstPageBtn         *widget.Button
		lastPageBtn          *widget.Button
		pageSizeSelect       *widget.Select
		orderColumnSelect    *widget.Select
		orderDirectionSelect *widget.Select
		quickPresetSelect    *widget.Select
		filtersPanel         = container.NewVBox()
		filterRows           []*explorerFilterRow
		suppressPresetChange bool
	)
	var availableColumns []string

	queryOptions := ui.explorerCurrentQuery
	pendingFilters := cloneExplorerFilters(queryOptions.Filters)
	pendingOrderColumn := strings.TrimSpace(queryOptions.OrderColumn)
	pendingOrderDescending := queryOptions.OrderDescending

	// Page size options with descriptions for better UX
	pageSizeOptions := []string{"10", "25", "50", "100", "250", "500", "1000"}
	currentPageSize := 50
	ui.explorerCurrentPageSize = currentPageSize

	// Prepare filter mode metadata for control updates
	filterModeOptions := []struct {
		Label string
		Mode  ExplorerFilterMode
	}{
		{"Contains", FilterModeContains},
		{"Equals", FilterModeEquals},
		{"Not Equals", FilterModeNotEquals},
		{"Starts With", FilterModeStartsWith},
		{"Ends With", FilterModeEndsWith},
	}
	modeLabels := make([]string, len(filterModeOptions))
	modeLabelByMode := make(map[ExplorerFilterMode]string, len(filterModeOptions))
	modeByLabel := make(map[string]ExplorerFilterMode, len(filterModeOptions))
	for i, opt := range filterModeOptions {
		modeLabels[i] = opt.Label
		modeLabelByMode[opt.Mode] = opt.Label
		modeByLabel[opt.Label] = opt.Mode
	}

	var (
		updateFilterRowOptions func()
		rebuildFilterRows      func()
		presetLabelForFilters  func() string
	)

	createFilterRow := func(clause *ExplorerFilterClause) *explorerFilterRow {
		if clause.Mode == FilterModeNone {
			clause.Mode = FilterModeContains
		}

		row := &explorerFilterRow{clause: clause}

		columnSelect := widget.NewSelect(append([]string{}, availableColumns...), func(value string) {
			clause.Column = strings.TrimSpace(value)
		})
		columnSelect.PlaceHolder = "Column"
		if clause.Column != "" {
			columnSelect.SetSelected(clause.Column)
		}

		modeSelect := widget.NewSelect(modeLabels, func(label string) {
			mode := modeByLabel[label]
			if mode == "" {
				mode = FilterModeContains
			}
			clause.Mode = mode
		})
		initialModeLabel := modeLabelByMode[clause.Mode]
		if initialModeLabel == "" {
			clause.Mode = FilterModeContains
			initialModeLabel = modeLabelByMode[FilterModeContains]
		}
		modeSelect.SetSelected(initialModeLabel)

		valueEntry := widget.NewEntry()
		valueEntry.SetPlaceHolder("Value")
		valueEntry.SetText(clause.Value)
		valueEntry.OnChanged = func(val string) {
			clause.Value = val
		}

		removeButton := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() {
			for idx := range pendingFilters {
				if &pendingFilters[idx] == clause {
					pendingFilters = append(pendingFilters[:idx], pendingFilters[idx+1:]...)
					break
				}
			}
			rebuildFilterRows()
		})
		removeButton.Importance = widget.LowImportance

		row.column = columnSelect
		row.mode = modeSelect
		row.value = valueEntry
		row.remove = removeButton
		row.container = container.NewGridWithColumns(4, columnSelect, modeSelect, valueEntry, removeButton)

		return row
	}

	rebuildFilterRows = func() {
		filtersPanel.Objects = nil
		filterRows = filterRows[:0]

		if len(pendingFilters) == 0 {
			empty := widget.NewLabel("No filters applied")
			empty.Alignment = fyne.TextAlignCenter
			empty.Wrapping = fyne.TextWrapWord
			filtersPanel.Add(container.NewPadded(empty))
		} else {
			for i := range pendingFilters {
				clause := &pendingFilters[i]
				row := createFilterRow(clause)
				filterRows = append(filterRows, row)
				filtersPanel.Add(row.container)
			}
		}

		filtersPanel.Refresh()

		if updateFilterRowOptions != nil {
			updateFilterRowOptions()
		}
	}

	updateFilterRowOptions = func() {
		options := append([]string{}, availableColumns...)
		for _, row := range filterRows {
			row.column.Options = options
			if row.clause.Column != "" && !containsString(options, row.clause.Column) {
				row.clause.Column = ""
				row.column.ClearSelected()
			} else if row.clause.Column != "" {
				row.column.SetSelected(row.clause.Column)
			}
			row.column.Refresh()
		}

		if orderColumnSelect != nil {
			orderOptions := append([]string{""}, availableColumns...)
			orderColumnSelect.Options = orderOptions
			if pendingOrderColumn == "" {
				orderColumnSelect.ClearSelected()
			} else if containsString(availableColumns, pendingOrderColumn) {
				orderColumnSelect.SetSelected(pendingOrderColumn)
			} else {
				pendingOrderColumn = ""
				orderColumnSelect.ClearSelected()
			}
			orderColumnSelect.Refresh()
		}
	}

	rebuildFilterRows()

	// Function to load a specific page
	loadPage := func(tableName string, page int, pageSize int, opts ExplorerQueryOptions) {
		if tableName == "" {
			return
		}

		queryOptions = opts
		pendingFilters = cloneExplorerFilters(queryOptions.Filters)
		pendingOrderColumn = strings.TrimSpace(queryOptions.OrderColumn)
		pendingOrderDescending = queryOptions.OrderDescending
		ui.explorerCurrentQuery = queryOptions

		ui.app.Events.Dispatch(events.Infof("gui", "Explorer: Loading page %d of table '%s'", page+1, tableName))

		// Load paginated data
		paginatedData := ui.loadPaginatedTableData(tableName, page, pageSize, queryOptions)
		if paginatedData == nil {
			tableContainer.Objects = []fyne.CanvasObject{widget.NewLabel("Error loading table data")}
			tableContainer.Refresh()
			return
		}

		currentPaginatedData = paginatedData
		currentTableName = tableName

		availableColumns = paginatedData.Headers
		if len(availableColumns) == 0 {
			availableColumns = ui.getTableColumns(tableName)
		}

		rebuildFilterRows()
		updateFilterRowOptions()

		if quickPresetSelect != nil {
			preset := presetLabelForFilters()
			suppressPresetChange = true
			if preset == "" {
				quickPresetSelect.ClearSelected()
			} else {
				quickPresetSelect.SetSelected(preset)
			}
			suppressPresetChange = false
		}

		if orderDirectionSelect != nil {
			directionLabel := "Ascending"
			if pendingOrderDescending {
				directionLabel = "Descending"
			}
			if orderDirectionSelect.Selected != directionLabel {
				orderDirectionSelect.SetSelected(directionLabel)
			}
		}

		// Create table using the table factory
		factory := NewTableFactory(ui)
		config := TableConfig{
			Headers:       paginatedData.Headers,
			Data:          paginatedData.Data,
			HasCheckboxes: false, // Explorer doesn't need checkboxes
			EmptyMessage:  fmt.Sprintf("No rows found in %s.", tableName),
		}

		// Create auto-truncated table for better display
		table := factory.CreateAutoTruncatedTable(config)

		// Update page info with row range
		startRow := paginatedData.CurrentPage*paginatedData.PageSize + 1
		endRow := startRow + len(paginatedData.Data) - 1
		if len(paginatedData.Data) == 0 {
			startRow = 0
			endRow = 0
		}

		pageInfoLabel.SetText(fmt.Sprintf("Page %d of %d | Rows %d-%d of %d",
			paginatedData.CurrentPage+1, paginatedData.TotalPages, startRow, endRow, paginatedData.TotalRows))

		// Update pagination buttons
		prevBtn.Disable()
		nextBtn.Disable()
		firstPageBtn.Disable()
		lastPageBtn.Disable()

		if paginatedData.TotalPages > 1 {
			if paginatedData.CurrentPage > 0 {
				prevBtn.Enable()
				firstPageBtn.Enable()
			}
			if paginatedData.CurrentPage < paginatedData.TotalPages-1 {
				nextBtn.Enable()
				lastPageBtn.Enable()
			}
		}

		// Clear search when changing pages
		if searchEntry != nil {
			searchEntry.SetText("")
		}

		tableContainer.Objects = []fyne.CanvasObject{table}
		tableContainer.Refresh()
	}

	var applyQuery func(resetPage bool)

	ui.explorerLoadPage = loadPage

	// Table selection
	tableSelect := widget.NewSelect([]string{}, func(tableName string) {
		if tableName == "" {
			tableContainer.Objects = nil
			tableContainer.Refresh()
			return
		}
		queryOptions = ui.explorerCurrentQuery
		loadPage(tableName, 0, currentPageSize, queryOptions)
	})
	ui.explorerTableSelect = tableSelect

	applyQuery = func(resetPage bool) {
		queryOptions.Filters = cloneExplorerFilters(pendingFilters)
		queryOptions.OrderColumn = strings.TrimSpace(pendingOrderColumn)
		queryOptions.OrderDescending = pendingOrderDescending

		ui.explorerCurrentQuery = queryOptions

		if currentTableName != "" {
			targetPage := 0
			if !resetPage && currentPaginatedData != nil {
				targetPage = currentPaginatedData.CurrentPage
			}
			loadPage(currentTableName, targetPage, currentPageSize, queryOptions)
		}
	}

	// Search functionality (searches within current page results)
	searchEntry = widget.NewEntry()
	searchEntry.SetPlaceHolder("Search current page...")
	searchEntry.OnChanged = func(query string) {
		if currentPaginatedData == nil {
			return
		}

		// Filter current page data
		var filteredData [][]string
		if query == "" {
			// Show all data from current page
			filteredData = currentPaginatedData.Data
		} else {
			query = strings.ToLower(query)
			for _, row := range currentPaginatedData.Data {
				for _, cell := range row {
					if strings.Contains(strings.ToLower(cell), query) {
						filteredData = append(filteredData, row)
						break
					}
				}
			}
		}

		// Recreate table with filtered data
		factory := NewTableFactory(ui)
		config := TableConfig{
			Headers:       currentPaginatedData.Headers,
			Data:          filteredData,
			HasCheckboxes: false,
			EmptyMessage:  fmt.Sprintf("No rows found in %s.", currentTableName),
		}

		table := factory.CreateAutoTruncatedTable(config)
		tableContainer.Objects = []fyne.CanvasObject{table}
		tableContainer.Refresh()
	}

	applyStatusPreset := func(status string) {
		if status == "" {
			filtered := pendingFilters[:0]
			for _, clause := range pendingFilters {
				if !strings.EqualFold(clause.Column, "Status") {
					filtered = append(filtered, clause)
				}
			}
			pendingFilters = filtered
			rebuildFilterRows()
			updateFilterRowOptions()
			return
		}

		if !containsString(availableColumns, "Status") {
			ui.ShowToast("Selected table does not have a Status column.")
			return
		}

		updated := false
		for idx := range pendingFilters {
			if strings.EqualFold(pendingFilters[idx].Column, "Status") {
				pendingFilters[idx].Column = "Status"
				pendingFilters[idx].Mode = FilterModeEquals
				pendingFilters[idx].Value = status
				updated = true
				break
			}
		}

		if !updated {
			pendingFilters = append(pendingFilters, ExplorerFilterClause{Column: "Status", Mode: FilterModeEquals, Value: status})
		}

		rebuildFilterRows()
		updateFilterRowOptions()
	}

	presetLabelForFilters = func() string {
		for _, clause := range pendingFilters {
			if strings.EqualFold(clause.Column, "Status") {
				switch strings.ToLower(clause.Value) {
				case "pending":
					return "Status: Pending"
				case "processing", "in_progress":
					return "Status: Processing"
				case "completed", "success":
					return "Status: Completed"
				case "failed", "error":
					return "Status: Failed"
				}
			}
		}
		return ""
	}

	quickPresetLabels := []string{"No Preset", "Status: Pending", "Status: Processing", "Status: Completed", "Status: Failed"}
	quickPresetSelect = widget.NewSelect(quickPresetLabels, func(label string) {
		if suppressPresetChange {
			return
		}
		switch label {
		case "", "No Preset":
			applyStatusPreset("")
		case "Status: Pending":
			applyStatusPreset("pending")
		case "Status: Processing":
			applyStatusPreset("processing")
		case "Status: Completed":
			applyStatusPreset("completed")
		case "Status: Failed":
			applyStatusPreset("failed")
		}
	})
	quickPresetSelect.PlaceHolder = "Quick preset"

	orderColumnSelect = widget.NewSelect([]string{}, func(value string) {
		pendingOrderColumn = strings.TrimSpace(value)
	})
	orderColumnSelect.PlaceHolder = "Order column"

	orderDirectionOptions := []string{"Ascending", "Descending"}
	orderDirectionSelect = widget.NewSelect(orderDirectionOptions, func(label string) {
		pendingOrderDescending = (label == "Descending")
	})
	orderDirectionSelect.PlaceHolder = "Direction"
	if pendingOrderDescending {
		orderDirectionSelect.SetSelected("Descending")
	} else {
		orderDirectionSelect.SetSelected("Ascending")
	}

	addFilterButton := widget.NewButtonWithIcon("Add Filter", theme.ContentAddIcon(), func() {
		pendingFilters = append(pendingFilters, ExplorerFilterClause{Mode: FilterModeContains})
		rebuildFilterRows()
		updateFilterRowOptions()
	})

	clearFiltersButton := widget.NewButtonWithIcon("Clear Filters", theme.ContentClearIcon(), func() {
		pendingFilters = nil
		rebuildFilterRows()
		updateFilterRowOptions()
		if quickPresetSelect != nil {
			quickPresetSelect.ClearSelected()
		}
	})

	applyFiltersButton := widget.NewButtonWithIcon("Apply", theme.ConfirmIcon(), func() {
		applyQuery(true)
	})

	// Page size selection
	pageSizeSelect = widget.NewSelect(pageSizeOptions, func(selected string) {
		if selected == "" {
			return
		}

		newPageSize := 50 // default
		switch selected {
		case "10":
			newPageSize = 10
		case "25":
			newPageSize = 25
		case "50":
			newPageSize = 50
		case "100":
			newPageSize = 100
		case "250":
			newPageSize = 250
		case "500":
			newPageSize = 500
		case "1000":
			newPageSize = 1000
		}

		if newPageSize != currentPageSize {
			currentPageSize = newPageSize
			ui.explorerCurrentPageSize = currentPageSize
			if currentTableName != "" {
				loadPage(currentTableName, 0, currentPageSize, queryOptions) // Reset to first page
			}
		}
	})
	pageSizeSelect.SetSelected("50")

	// Pagination controls
	pageInfoLabel = widget.NewLabel("No data loaded")

	prevBtn = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if currentPaginatedData != nil && currentPaginatedData.CurrentPage > 0 {
			loadPage(currentTableName, currentPaginatedData.CurrentPage-1, currentPageSize, queryOptions)
		}
	})
	prevBtn.Disable()

	nextBtn = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		if currentPaginatedData != nil && currentPaginatedData.CurrentPage < currentPaginatedData.TotalPages-1 {
			loadPage(currentTableName, currentPaginatedData.CurrentPage+1, currentPageSize, queryOptions)
		}
	})
	nextBtn.Disable()

	firstPageBtn = widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		if currentPaginatedData != nil && currentPaginatedData.CurrentPage > 0 {
			loadPage(currentTableName, 0, currentPageSize, queryOptions)
		}
	})

	lastPageBtn = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		if currentPaginatedData != nil && currentPaginatedData.CurrentPage < currentPaginatedData.TotalPages-1 {
			loadPage(currentTableName, currentPaginatedData.TotalPages-1, currentPageSize, queryOptions)
		}
	})
	firstPageBtn.Disable()
	lastPageBtn.Disable()

	// Go to page functionality
	gotoPageEntry := widget.NewEntry()
	gotoPageEntry.SetPlaceHolder("#")
	gotoPageEntry.Resize(fyne.NewSize(50, gotoPageEntry.MinSize().Height))

	gotoPageBtn := widget.NewButton("Go", func() {
		if currentPaginatedData == nil {
			return
		}

		pageText := gotoPageEntry.Text
		if pageText == "" {
			return
		}

		var targetPage int
		if _, err := fmt.Sscanf(pageText, "%d", &targetPage); err != nil {
			ui.app.Events.Dispatch(events.Errorf("gui", "Invalid page number: %s", pageText))
			return
		}

		// Convert to 0-based indexing and validate
		targetPage--
		if targetPage < 0 {
			targetPage = 0
		}
		if targetPage >= currentPaginatedData.TotalPages {
			targetPage = currentPaginatedData.TotalPages - 1
		}

		if targetPage != currentPaginatedData.CurrentPage {
			loadPage(currentTableName, targetPage, currentPageSize, queryOptions)
		}

		gotoPageEntry.SetText("")
	})

	// Export current page data
	exportBtn := widget.NewButtonWithIcon("Export Page", theme.DocumentSaveIcon(), func() {
		if currentPaginatedData == nil || len(currentPaginatedData.Data) == 0 {
			ui.app.Events.Dispatch(events.Infof("gui", "No data to export"))
			return
		}

		// Create CSV content
		var csvContent strings.Builder

		// Add headers
		csvContent.WriteString(strings.Join(currentPaginatedData.Headers, ","))
		csvContent.WriteString("\n")

		// Add data rows
		for _, row := range currentPaginatedData.Data {
			// Escape fields that contain commas or quotes
			var escapedRow []string
			for _, field := range row {
				if strings.Contains(field, ",") || strings.Contains(field, "\"") || strings.Contains(field, "\n") {
					field = "\"" + strings.ReplaceAll(field, "\"", "\"\"") + "\""
				}
				escapedRow = append(escapedRow, field)
			}
			csvContent.WriteString(strings.Join(escapedRow, ","))
			csvContent.WriteString("\n")
		}

		// Save dialog
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				ui.app.Events.Dispatch(events.Errorf("gui", "Error opening file for export: %v", err))
				return
			}
			if writer == nil {
				return // User cancelled
			}
			defer writer.Close()

			_, err = writer.Write([]byte(csvContent.String()))
			if err != nil {
				ui.app.Events.Dispatch(events.Errorf("gui", "Error writing export file: %v", err))
				return
			}

			ui.app.Events.Dispatch(events.Infof("gui", "Exported %d rows from %s (page %d) to %s",
				len(currentPaginatedData.Data), currentTableName, currentPaginatedData.CurrentPage+1, writer.URI().Path()))
		}, ui.window)
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
			pageInfoLabel.SetText("No data loaded")
			prevBtn.Disable()
			nextBtn.Disable()
			firstPageBtn.Disable()
			lastPageBtn.Disable()
		}()
	})

	// Initial table list load
	go func() {
		tables, err := ui.app.DB.GetTables()
		if err != nil {
			ui.app.Events.Dispatch(events.Errorf("gui", "Error getting tables: %v", err))
			return
		}
		tableSelect.Options = tables
		tableSelect.Refresh()
	}()

	// Layout - simplified top section
	controlsLeft := container.NewHBox(widget.NewLabel("Table:"), tableSelect, refreshButton)
	var filterSidebar *fyne.Container
	filterToggleBtn := widget.NewButtonWithIcon("Filters", theme.MenuDropDownIcon(), func() {
		// Show filters in the slide-over (right pane)
		if filterSidebar != nil {
			ui.ShowDetails(filterSidebar)
		}
	})
	// Make search entry take up available middle space
	searchControls := container.NewBorder(nil, nil, widget.NewLabel("Search:"), nil, container.NewMax(searchEntry))
	// Place filters button all the way to the right
	topRow := container.NewBorder(nil, nil, controlsLeft, filterToggleBtn, searchControls)

	filtersCard := widget.NewCard(
		"Filters & Sorting",
		"",
		container.NewVBox(
			quickPresetSelect,
			container.NewGridWithColumns(2, addFilterButton, clearFiltersButton),
			widget.NewSeparator(),
			container.NewMax(filtersPanel),
			widget.NewSeparator(),
			container.NewGridWithColumns(2, orderColumnSelect, orderDirectionSelect),
			applyFiltersButton,
		),
	)

	filterSidebar = container.NewVBox(filtersCard)

	topContent := container.NewVBox(
		widget.NewLabelWithStyle("Database Explorer", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		topRow,
	)

	ui.explorerApplyQuery = func(opts ExplorerQueryOptions, reload bool) {
		queryOptions = opts
		pendingFilters = cloneExplorerFilters(opts.Filters)
		pendingOrderColumn = strings.TrimSpace(opts.OrderColumn)
		pendingOrderDescending = opts.OrderDescending
		ui.explorerCurrentQuery = queryOptions

		rebuildFilterRows()
		updateFilterRowOptions()

		if quickPresetSelect != nil {
			preset := presetLabelForFilters()
			suppressPresetChange = true
			if preset == "" {
				quickPresetSelect.ClearSelected()
			} else {
				quickPresetSelect.SetSelected(preset)
			}
			suppressPresetChange = false
		}

		if orderDirectionSelect != nil {
			if pendingOrderDescending {
				orderDirectionSelect.SetSelected("Descending")
			} else {
				orderDirectionSelect.SetSelected("Ascending")
			}
		}

		if reload {
			applyQuery(true)
		}
	}

	// Create a refresh data button for the pagination bar
	refreshDataBtn := widget.NewButtonWithIcon("Refresh Data", theme.ViewRefreshIcon(), func() {
		if currentTableName != "" && currentPaginatedData != nil {
			loadPage(currentTableName, currentPaginatedData.CurrentPage, currentPageSize, queryOptions)
		}
	})

	// Comprehensive bottom pagination bar
	leftSection := container.NewHBox(
		widget.NewLabel("Rows:"),
		pageSizeSelect,
		widget.NewSeparator(),
		exportBtn,
		refreshDataBtn,
	)

	centerSection := container.NewHBox(
		firstPageBtn,
		prevBtn,
		widget.NewLabel(" "),
		pageInfoLabel,
		widget.NewLabel(" "),
		nextBtn,
		lastPageBtn,
	)

	rightSection := container.NewHBox(
		widget.NewLabel("Jump to:"),
		gotoPageEntry,
		gotoPageBtn,
	)

	// Create a comprehensive pagination toolbar
	paginationBar := container.NewBorder(
		nil, nil,
		leftSection,
		rightSection,
		container.NewCenter(centerSection),
	)

	// Add visual styling to the pagination bar with minimal padding
	paginationBarStyled := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox( // Use HBox instead of Padded to reduce vertical space
			widget.NewLabel(" "), // Small left spacer
			paginationBar,
			widget.NewLabel(" "), // Small right spacer
		),
	)

	centerContent := container.NewBorder(nil, nil, nil, nil, container.NewVScroll(tableContainer))
	// Remove persistent sidebar; show filters via slide-over using right pane
	return container.NewBorder(topContent, paginationBarStyled, nil, nil, centerContent)
}

// OpenExplorerPendingChanges switches to the explorer tab and selects a pending changes table.
func (ui *Gui) OpenExplorerPendingChanges() bool {
	tableCandidates := []string{"AccountsPendingChanges", "AccountCheckinsPendingChanges"}
	for _, table := range tableCandidates {
		if ui.OpenExplorerTable(table) {
			return true
		}
	}

	ui.ShowToast("Pending changes tables are not available in the database.")
	return false
}

// OpenExplorerTable activates the explorer tab and loads the specified table.
func (ui *Gui) OpenExplorerTable(tableName string) bool {
	if ui.app == nil || ui.app.DB == nil || !ui.app.DB.IsConnected() {
		ui.ShowToast("Connect to the database to browse tables in Explorer.")
		return false
	}

	if ui.tabs == nil {
		return false
	}

	explorerIndex := -1
	for idx, tab := range ui.tabs.Items {
		if tab.Text == "Explorer" {
			explorerIndex = idx
			break
		}
	}
	if explorerIndex == -1 {
		ui.ShowToast("Explorer tab is not available.")
		return false
	}
	ui.tabs.SelectIndex(explorerIndex)

	if ui.explorerTableSelect == nil {
		ui.ShowToast("Explorer controls are not initialized yet.")
		return false
	}

	if !ui.ensureExplorerTableOption(tableName) {
		ui.ShowToast(fmt.Sprintf("Table %s is not available.", tableName))
		return false
	}

	if defaults, ok := defaultExplorerQuery(tableName); ok {
		ui.explorerCurrentQuery = defaults
		if ui.explorerApplyQuery != nil {
			ui.explorerApplyQuery(defaults, false)
		}
	}

	if ui.explorerTableSelect.Selected != tableName {
		ui.explorerTableSelect.SetSelected(tableName)
	} else if ui.explorerLoadPage != nil {
		pageSize := ui.explorerCurrentPageSize
		if pageSize <= 0 {
			pageSize = 50
		}
		ui.explorerLoadPage(tableName, 0, pageSize, ui.explorerCurrentQuery)
	}

	return true
}

func (ui *Gui) OpenConfigTab() bool {
	if ui.tabs == nil {
		return false
	}

	for idx, tab := range ui.tabs.Items {
		if tab.Text == "Configuration" {
			ui.tabs.SelectIndex(idx)
			return true
		}
	}

	return false
}

func (ui *Gui) OpenServerTab() bool {
	if ui.tabs == nil {
		return false
	}

	for idx, tab := range ui.tabs.Items {
		if tab.Text == "Server" {
			ui.tabs.SelectIndex(idx)
			return true
		}
	}

	return false
}

func defaultExplorerQuery(tableName string) (ExplorerQueryOptions, bool) {
	switch tableName {
	case "AccountsPendingChanges", "AccountCheckinsPendingChanges":
		return ExplorerQueryOptions{
			Filters: []ExplorerFilterClause{
				{Column: "Status", Mode: FilterModeNotEquals, Value: "completed"},
			},
			OrderColumn:     "CreatedAt",
			OrderDescending: true,
		}, true
	default:
		return ExplorerQueryOptions{}, false
	}
}

func (ui *Gui) ensureExplorerTableOption(tableName string) bool {
	if ui.explorerTableSelect == nil {
		return false
	}

	if containsString(ui.explorerTableSelect.Options, tableName) {
		return true
	}

	if ui.app != nil && ui.app.DB != nil && ui.app.DB.IsConnected() {
		tables, err := ui.app.DB.GetTables()
		if err != nil {
			ui.app.Events.Dispatch(events.Errorf("gui", "Error refreshing explorer tables: %v", err))
			return false
		}
		ui.explorerTableSelect.Options = tables
		ui.explorerTableSelect.Refresh()
		return containsString(tables, tableName)
	}

	return false
}

func containsString(haystack []string, needle string) bool {
	for _, value := range haystack {
		if value == needle {
			return true
		}
	}
	return false
}

func formatEventName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Any"
	}

	replacer := strings.NewReplacer(".", " ", "_", " ", "-", " ")
	normalized := replacer.Replace(value)
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return value
	}
	for i, part := range parts {
		runes := []rune(strings.ToLower(part))
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		} else {
			parts[i] = part
		}
	}
	return strings.Join(parts, " ")
}

func (ui *Gui) buildSyncAutomationCard() fyne.CanvasObject {
	summary := widget.NewLabel("Timed automatic sync keeps data aligned without manual runs.")
	summary.Alignment = fyne.TextAlignLeading
	summary.Wrapping = fyne.TextWrapWord

	autoSyncCheck := widget.NewCheck("Enable automatic sync", nil)
	autoSyncCheck.SetChecked(false)

	scheduleSelect := widget.NewSelect([]string{
		"Every 5 minutes",
		"Every 15 minutes",
		"Every 30 minutes",
		"Hourly",
		"Every 6 hours",
		"Daily",
	}, nil)
	scheduleSelect.SetSelected("Every 30 minutes")
	scheduleSelect.Disable()

	autoSyncCheck.OnChanged = func(enabled bool) {
		if enabled {
			scheduleSelect.Enable()
		} else {
			scheduleSelect.Disable()
		}
	}

	controls := container.NewGridWithColumns(2,
		widget.NewLabel("Interval"),
		scheduleSelect,
	)

	syncedHeading := widget.NewLabelWithStyle("Automatically synced items:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	syncedList := container.NewVBox(
		widget.NewLabel("• Accounts"),
		widget.NewLabel("• Check-ins"),
		widget.NewLabel("• Health endpoint availability"),
	)

	saveBtn := widget.NewButtonWithIcon("Save Automation Settings", theme.DocumentSaveIcon(), func() {
		ui.ShowToast("Sync automation settings saved (coming soon)")
	})

	return ui.newSectionCard(
		"Automatic Sync",
		"Manage the embedded scheduler settings.",
		summary,
		autoSyncCheck,
		controls,
		syncedHeading,
		syncedList,
		container.NewCenter(saveBtn),
	)
}

// createServerTab creates the content for the "Server" tab
func (ui *Gui) createServerTab() fyne.CanvasObject {
	statusValue := canvas.NewText("Unknown", ui.themeColor(StatusNegativeColorName))
	statusValue.TextStyle = fyne.TextStyle{Bold: true}
	statusValue.TextSize = theme.TextSize()
	toggleServerButton := widget.NewButtonWithIcon("", nil, nil)
	toggleServerButton.Importance = widget.HighImportance

	webhooks := ui.app.Config.Server.Webhooks
	if webhooks == nil {
		webhooks = map[string]bool{
			app.WebhookAccountCreate: true,
			app.WebhookCheckin:       true,
		}
	}

	accountWebhookCheck := widget.NewCheck("Account create webhook", nil)
	checkinWebhookCheck := widget.NewCheck("Check-in webhook", nil)
	accountWebhookCheck.SetChecked(webhooks[app.WebhookAccountCreate])
	checkinWebhookCheck.SetChecked(webhooks[app.WebhookCheckin])
	webhookStatusLabel := widget.NewLabel("")
	updateWebhookStatus := func(accountEnabled, checkinEnabled bool) {
		if !accountEnabled && !checkinEnabled {
			webhookStatusLabel.SetText("All webhooks disabled; server will only serve /health.")
			return
		}
		webhookStatusLabel.SetText("Select which webhooks the embedded server should handle.")
	}
	updateWebhookStatus(accountWebhookCheck.Checked, checkinWebhookCheck.Checked)

	accountWebhookCheck.OnChanged = func(enabled bool) {
		currentCheckin := checkinWebhookCheck.Checked
		updateWebhookStatus(enabled, currentCheckin)
		ui.presenter.HandleUpdateServerWebhooks(enabled, currentCheckin)
	}
	checkinWebhookCheck.OnChanged = func(enabled bool) {
		currentAccount := accountWebhookCheck.Checked
		updateWebhookStatus(currentAccount, enabled)
		ui.presenter.HandleUpdateServerWebhooks(currentAccount, enabled)
	}

	var refreshServerStatus func()
	setToggleButton := func(running bool) {
		if running {
			toggleServerButton.SetText("Stop Server")
			toggleServerButton.SetIcon(theme.MediaStopIcon())
			toggleServerButton.OnTapped = func() {
				ui.presenter.HandleStopServer()
				refreshServerStatus()
			}
		} else {
			toggleServerButton.SetText("Start Server")
			toggleServerButton.SetIcon(theme.MediaPlayIcon())
			toggleServerButton.OnTapped = func() {
				ui.presenter.HandleStartServer()
				refreshServerStatus()
			}
		}
	}

	refreshServerStatus = func() {
		if pid, running := ui.app.Server.GetServerStatus(); running {
			statusValue.Text = fmt.Sprintf("Running (PID: %d)", pid)
			statusValue.Color = ui.themeColor(StatusPositiveColorName)
			setToggleButton(true)
		} else {
			statusValue.Text = "Stopped"
			statusValue.Color = ui.themeColor(StatusNegativeColorName)
			setToggleButton(false)
		}
		canvas.Refresh(statusValue)
	}

	refreshServerStatus()

	statusLabel := widget.NewLabelWithStyle("Server Status:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	serverHeader := container.NewHBox(statusLabel, statusValue)

	webhookCard := ui.newSectionCard(
		"Webhook Routing",
		"Select which webhooks the embedded server should handle.",
		accountWebhookCheck,
		checkinWebhookCheck,
		webhookStatusLabel,
	)

	// Server configuration form
	serverHostEntry := widget.NewEntry()
	serverHostEntry.SetText(ui.app.Config.Server.Host)
	serverPortEntry := widget.NewEntry()
	serverPortEntry.SetText(fmt.Sprintf("%d", ui.app.Config.Server.Port))
	tlsCertEntry := widget.NewEntry()
	tlsCertEntry.SetText(ui.app.Config.Server.TLSCert)
	tlsKeyEntry := widget.NewEntry()
	tlsKeyEntry.SetText(ui.app.Config.Server.TLSKey)
	logRequestsCheck := widget.NewCheck("Log incoming requests", nil)
	logRequestsCheck.SetChecked(ui.app.Config.Server.LogRequests)

	var serverForm *widget.Form
	tlsCertFormItem := widget.NewFormItem("TLS Cert Path", tlsCertEntry)
	tlsKeyFormItem := widget.NewFormItem("TLS Key Path", tlsKeyEntry)
	tlsEnabledCheck := widget.NewCheck("Enable TLS", func(enabled bool) {
		if enabled {
			serverForm.AppendItem(tlsCertFormItem)
			serverForm.AppendItem(tlsKeyFormItem)
		} else {
			var newItems []*widget.FormItem
			for _, item := range serverForm.Items {
				if item != tlsCertFormItem && item != tlsKeyFormItem {
					newItems = append(newItems, item)
				}
			}
			serverForm.Items = newItems
		}
		serverForm.Refresh()
	})

	serverForm = widget.NewForm(
		widget.NewFormItem("Host", serverHostEntry),
		widget.NewFormItem("Port", serverPortEntry),
	)

	tlsEnabledCheck.SetChecked(ui.app.Config.Server.TLSEnabled)

	saveServerButton := NewSecondaryButton("Save Server Settings", theme.DocumentSaveIcon(), func() {
		ui.presenter.HandleSaveServerConfig(
			serverHostEntry.Text,
			serverPortEntry.Text,
			tlsEnabledCheck.Checked,
			tlsCertEntry.Text,
			tlsKeyEntry.Text,
			logRequestsCheck.Checked,
		)
	})

	serverSettingsCard := ui.newSectionCard(
		"Server Configuration",
		"Configure host, TLS, and request logging for the embedded server.",
		tlsEnabledCheck,
		serverForm,
		logRequestsCheck,
	)

	autoSyncCard := ui.buildSyncAutomationCard()

	scrollContent := container.NewVScroll(container.NewVBox(
		serverHeader,
		webhookCard,
		autoSyncCard,
		serverSettingsCard,
	))

	buttonGrid := container.NewGridWithColumns(2, saveServerButton, toggleServerButton)
	footer := container.NewVBox(widget.NewSeparator(), buttonGrid)

	return container.NewBorder(nil, footer, nil, nil, scrollContent)
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
	return ui.configTab
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
	apiCard := ui.newSectionCard(
		"API Configuration",
		"Provide the API credentials used for syncing.",
		widget.NewForm(
			widget.NewFormItem("API Key", apiKeyEntry),
			widget.NewFormItem("Base URL", baseURLEntry),
		),
		container.NewCenter(testApiButton),
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

	// Schema Management
	schemaLabel := "Initialize Schema"
	if ui.app.DB != nil && ui.app.DB.IsConnected() {
		if err := ui.app.DB.ValidateSchema(ui.app.State); err == nil {
			schemaLabel = "Re-initialize Schema"
		}
	}
	schemaButton := widget.NewButtonWithIcon(schemaLabel, theme.StorageIcon(), ui.presenter.HandleSchemaEnforcement)

	dbCard := ui.newSectionCard(
		"Database Configuration",
		"Select your database type and connection information.",
		container.NewGridWithColumns(2, widget.NewLabel("Database Type"), dbTypeSelect),
		dbForm,
		container.NewCenter(testDbButton),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Schema Management", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Initialize or rebuild the target database schema."),
		container.NewCenter(schemaButton),
	)

	// Sync Preferences
	conflictStrategyRadio := widget.NewRadioGroup([]string{
		"Always use local changes",
		"Always use remote changes",
		"Use most recent",
		"Ask every time",
	}, nil)
	conflictStrategyRadio.SetSelected("Ask every time")

	maxConcurrent := ui.app.Config.MaxConcurrentRequests
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	defaultParallelConcurrency := maxConcurrent
	if defaultParallelConcurrency < 2 {
		defaultParallelConcurrency = 2
	}

	parallelProcessingCheck := widget.NewCheck("Enable parallel processing", nil)
	parallelProcessingCheck.SetChecked(maxConcurrent > 1)
	maxConcurrentEntry := widget.NewEntry()
	maxConcurrentEntry.SetText(strconv.Itoa(maxConcurrent))
	lastParallelValue := strconv.Itoa(defaultParallelConcurrency)
	if maxConcurrent > 1 {
		lastParallelValue = strconv.Itoa(maxConcurrent)
	}
	if !parallelProcessingCheck.Checked {
		maxConcurrentEntry.Disable()
	}
	maxConcurrentEntry.OnChanged = func(value string) {
		if !parallelProcessingCheck.Checked {
			return
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || trimmed == "0" || trimmed == "1" {
			return
		}
		lastParallelValue = trimmed
	}
	parallelProcessingCheck.OnChanged = func(enabled bool) {
		if enabled {
			value := strings.TrimSpace(maxConcurrentEntry.Text)
			if value == "" || value == "1" {
				maxConcurrentEntry.SetText(lastParallelValue)
			}
			maxConcurrentEntry.Enable()
			return
		}
		current := strings.TrimSpace(maxConcurrentEntry.Text)
		if current != "" && current != "0" && current != "1" {
			lastParallelValue = current
		}
		maxConcurrentEntry.SetText("1")
		maxConcurrentEntry.Disable()
	}

	batchSizeEntry := widget.NewEntry()
	batchSizeEntry.SetText("100")

	verboseLoggingCheck := widget.NewCheck("Verbose logging", nil)
	logRetentionSelect := widget.NewSelect([]string{
		"7 days",
		"30 days",
		"90 days",
		"Forever",
	}, nil)
	logRetentionSelect.SetSelected("30 days")

	saveSyncPrefsBtn := widget.NewButtonWithIcon("Save Sync Preferences", theme.DocumentSaveIcon(), func() {
		ui.ShowToast("Sync preferences saved (coming soon)")
	})

	syncPreferencesCard := ui.newSectionCard(
		"Sync Preferences",
		"Control conflict handling and logging for manual sync runs.",
		widget.NewForm(
			widget.NewFormItem("Conflict Resolution", conflictStrategyRadio),
			widget.NewFormItem("Batch Size", batchSizeEntry),
			widget.NewFormItem("Verbose Logging", verboseLoggingCheck),
			widget.NewFormItem("Log Retention", logRetentionSelect),
			widget.NewFormItem("Parallel Processing", parallelProcessingCheck),
			widget.NewFormItem("Max Concurrent", maxConcurrentEntry),
		),
		container.NewCenter(saveSyncPrefsBtn),
	)

	themeLabels := []string{"Auto (Follow System)", "Light", "Dark"}
	themeLabelToValue := map[string]string{
		themeLabels[0]: app.ThemePreferenceAuto,
		themeLabels[1]: app.ThemePreferenceLight,
		themeLabels[2]: app.ThemePreferenceDark,
	}
	themeValueToLabel := map[string]string{}
	for label, value := range themeLabelToValue {
		themeValueToLabel[value] = label
	}

	themeRadio := widget.NewRadioGroup(themeLabels, nil)
	themeRadio.Required = true
	currentThemeLabel := themeValueToLabel[app.NormalizeThemePreference(ui.app.Config.ThemePreference)]
	if currentThemeLabel == "" {
		currentThemeLabel = themeLabels[0]
	}
	themeRadio.SetSelected(currentThemeLabel)

	appearanceCard := ui.newSectionCard(
		"Appearance",
		"Choose how BadgerMaps Sync looks.",
		widget.NewForm(
			widget.NewFormItem("Theme", themeRadio),
		),
	)

	// Other Settings
	verboseCheck := widget.NewCheck("Debug", nil)
	verboseCheck.SetChecked(ui.app.State.Debug)
	otherCard := ui.newSectionCard("Other Settings", "", verboseCheck)

	// Buttons
	saveButton := NewSecondaryButton("Save Configuration", theme.ConfirmIcon(), func() {
		selectedThemePreference := app.ThemePreferenceAuto
		if label := themeRadio.Selected; label != "" {
			if value, ok := themeLabelToValue[label]; ok {
				selectedThemePreference = value
			}
		}
		ui.presenter.HandleSaveConfig(
			apiKeyEntry.Text, baseURLEntry.Text, dbTypeSelect.Selected, dbPathEntry.Text,
			dbHostEntry.Text, dbPortEntry.Text, dbUserEntry.Text, dbPassEntry.Text, dbNameEntry.Text,
			selectedThemePreference,
			false, // verbose is deprecated in gui
			verboseCheck.Checked,
			maxConcurrentEntry.Text,
			parallelProcessingCheck.Checked,
		)
	})

	viewButton := widget.NewButtonWithIcon("View", theme.VisibilityIcon(), ui.presenter.HandleViewConfig)

	actionsCard := ui.newSectionCard(
		"Configuration Actions",
		"Review or persist the current configuration.",
		container.NewGridWithColumns(2, viewButton, saveButton),
	)

	scrollContent := container.NewVScroll(container.NewVBox(
		NewSpacer(fyne.NewSize(0, 10)),
		apiCard,
		dbCard,
		syncPreferencesCard,
		appearanceCard,
		otherCard,
		actionsCard,
	))

	return scrollContent
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
		fyne.Do(func() {
			winSize := ui.window.Canvas().Size()
			popupSize := popup.MinSize()
			popup.Move(fyne.NewPos(winSize.Width-popupSize.Width-theme.Padding(), winSize.Height-popupSize.Height-theme.Padding()))
		})
	}()

	popup.Show()

	// Hide the popup after a short duration
	go func() {
		time.Sleep(3 * time.Second)
		fyne.Do(func() {
			popup.Hide()
			ui.toastMutex.Unlock()
		})
	}()
}

func (ui *Gui) ShowProgressBar(title string) {
	fyne.Do(func() {
		ui.progressTitle.SetText(title)
		ui.progressContainer.Show()
	})
}

func (ui *Gui) HideProgressBar() {
	fyne.Do(func() {
		ui.progressContainer.Hide()
		ui.progressTitle.SetText("")
	})
}

func (ui *Gui) SetProgress(value float64) {
	fyne.Do(func() {
		ui.progressBar.SetValue(value)
	})
}

func (ui *Gui) ShowErrorDialog(err error) {
	fyne.Do(func() {
		dialog.ShowError(err, ui.window)
	})
}

func (ui *Gui) ShowConfirmDialog(title, message string, callback func(bool)) {
	fyne.Do(func() {
		dialog.ShowConfirm(title, message, callback, ui.window)
	})
}

func (ui *Gui) GetMainWindow() fyne.Window {
	return ui.window
}

// refreshConfigTab rebuilds and refreshes the configuration tab
func (ui *Gui) RefreshConfigTab() {
	newConfigTab := ui.buildConfigTab()
	ui.configTab = newConfigTab
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

// TableData holds table structure for unified table handling
type TableData struct {
	Headers []string
	Data    [][]string
}

// PaginatedTableData contains table data with pagination info
type PaginatedTableData struct {
	TableData
	TotalRows   int
	CurrentPage int
	PageSize    int
	TotalPages  int
}

type ExplorerFilterMode string

const (
	FilterModeNone       ExplorerFilterMode = ""
	FilterModeContains   ExplorerFilterMode = "contains"
	FilterModeEquals     ExplorerFilterMode = "equals"
	FilterModeNotEquals  ExplorerFilterMode = "not_equals"
	FilterModeStartsWith ExplorerFilterMode = "starts_with"
	FilterModeEndsWith   ExplorerFilterMode = "ends_with"
)

type ExplorerQueryOptions struct {
	Filters         []ExplorerFilterClause
	OrderColumn     string
	OrderDescending bool
}

type ExplorerFilterClause struct {
	Column string
	Mode   ExplorerFilterMode
	Value  string
}

type explorerFilterRow struct {
	clause    *ExplorerFilterClause
	container *fyne.Container
	column    *widget.Select
	mode      *widget.Select
	value     *widget.Entry
	remove    *widget.Button
}

// loadTableData loads data from a database table using unified approach
func (ui *Gui) loadTableData(tableName string) *TableData {
	paginatedData := ui.loadPaginatedTableData(tableName, 0, 100, ExplorerQueryOptions{})
	return &paginatedData.TableData
}

// loadPaginatedTableData loads a specific page of data from a database table

func (ui *Gui) loadPaginatedTableData(tableName string, page, pageSize int, opts ExplorerQueryOptions) *PaginatedTableData {
	if ui.app == nil || ui.app.DB == nil || !ui.app.DB.IsConnected() {
		return &PaginatedTableData{
			TableData:   TableData{Headers: []string{}, Data: [][]string{}},
			TotalRows:   0,
			CurrentPage: 0,
			PageSize:    pageSize,
			TotalPages:  0,
		}
	}

	if pageSize <= 0 {
		pageSize = 50
	}
	if page < 0 {
		page = 0
	}

	normalized := normalizeExplorerOptions(opts)
	columns := ui.getTableColumns(tableName)

	resolvedFilters := resolveExplorerFilters(normalized.Filters, columns)
	orderColumn := matchColumn(columns, normalized.OrderColumn)

	whereClause := buildExplorerWhereClause(resolvedFilters)
	dbType := ui.app.DB.GetType()
	orderClause := buildExplorerOrderClause(columns, orderColumn, normalized.OrderDescending, dbType)

	countQuery := buildExplorerCountQuery(tableName, whereClause)

	countRows, err := ui.app.DB.ExecuteQuery(countQuery)
	if err != nil {
		ui.app.Events.Dispatch(events.Errorf("gui", "Error counting rows for %s: %v", tableName, err))
		return &PaginatedTableData{
			TableData:   TableData{Headers: []string{}, Data: [][]string{}},
			TotalRows:   0,
			CurrentPage: 0,
			PageSize:    pageSize,
			TotalPages:  0,
		}
	}
	defer countRows.Close()

	totalRows := 0
	if countRows.Next() {
		if err := countRows.Scan(&totalRows); err != nil {
			ui.app.Events.Dispatch(events.Errorf("gui", "Error reading row count for %s: %v", tableName, err))
			totalRows = 0
		}
	}

	totalPages := 1
	if totalRows > 0 {
		totalPages = (totalRows + pageSize - 1) / pageSize
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	if page < 0 {
		page = 0
	}
	selectQuery := buildExplorerSelectQuery(tableName, whereClause, orderClause, page, pageSize, dbType)

	rows, err := ui.app.DB.ExecuteQuery(selectQuery)
	if err != nil {
		ui.app.Events.Dispatch(events.Errorf("gui", "Error executing paginated query: %v", err))
		return &PaginatedTableData{
			TableData:   TableData{Headers: []string{}, Data: [][]string{}},
			TotalRows:   totalRows,
			CurrentPage: page,
			PageSize:    pageSize,
			TotalPages:  totalPages,
		}
	}
	defer rows.Close()

	resultColumns, err := rows.Columns()
	if err != nil {
		ui.app.Events.Dispatch(events.Errorf("gui", "Error getting columns: %v", err))
		return &PaginatedTableData{
			TableData:   TableData{Headers: []string{}, Data: [][]string{}},
			TotalRows:   totalRows,
			CurrentPage: page,
			PageSize:    pageSize,
			TotalPages:  totalPages,
		}
	}

	var data [][]string
	for rows.Next() {
		row := make([]interface{}, len(resultColumns))
		rowData := make([]string, len(resultColumns))
		for i := range row {
			row[i] = new(interface{})
		}
		if err := rows.Scan(row...); err != nil {
			ui.app.Events.Dispatch(events.Errorf("gui", "Error scanning row: %v", err))
			continue
		}
		for i, val := range row {
			if val == nil {
				rowData[i] = ""
				continue
			}
			v := val.(*interface{})
			if v == nil || *v == nil {
				rowData[i] = ""
				continue
			}
			if b, ok := (*v).([]byte); ok {
				rowData[i] = string(b)
			} else {
				rowData[i] = fmt.Sprintf("%v", *v)
			}
		}
		data = append(data, rowData)
	}

	ui.app.Events.Dispatch(events.Infof("gui", "Explorer: Loaded page %d of %d (%d rows) from '%s'", page+1, totalPages, len(data), tableName))

	return &PaginatedTableData{
		TableData:   TableData{Headers: resultColumns, Data: data},
		TotalRows:   totalRows,
		CurrentPage: page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
	}
}

func normalizeExplorerOptions(opts ExplorerQueryOptions) ExplorerQueryOptions {
	cleaned := make([]ExplorerFilterClause, 0, len(opts.Filters))
	for _, clause := range opts.Filters {
		clause.Column = strings.TrimSpace(clause.Column)
		clause.Value = strings.TrimSpace(clause.Value)

		if clause.Column == "" {
			continue
		}

		if clause.Mode == FilterModeNone {
			clause.Mode = FilterModeContains
		}

		if clause.Value == "" {
			// NotEquals with empty value is not meaningful
			continue
		}

		cleaned = append(cleaned, clause)
	}

	opts.Filters = cleaned
	opts.OrderColumn = strings.TrimSpace(opts.OrderColumn)
	return opts
}

func matchColumn(columns []string, name string) string {
	if name == "" {
		return ""
	}
	for _, col := range columns {
		if strings.EqualFold(col, name) {
			return col
		}
	}
	return ""
}

func fallbackOrderColumn(columns []string) string {
	if len(columns) == 0 {
		return ""
	}
	for _, col := range columns {
		lower := strings.ToLower(col)
		if lower == "createdat" || lower == "created_at" {
			return col
		}
	}
	for _, col := range columns {
		if strings.Contains(strings.ToLower(col), "created") {
			return col
		}
	}
	for _, col := range columns {
		if strings.Contains(strings.ToLower(col), "id") {
			return col
		}
	}
	return columns[0]
}

func escapeSQLLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func buildFilterCondition(column string, mode ExplorerFilterMode, value string) string {
	lowerColumn := fmt.Sprintf("LOWER(%s)", column)
	lowerValue := strings.ToLower(value)
	escaped := escapeSQLLiteral(lowerValue)

	switch mode {
	case FilterModeEquals:
		return fmt.Sprintf("%s = '%s'", lowerColumn, escaped)
	case FilterModeNotEquals:
		return fmt.Sprintf("%s <> '%s'", lowerColumn, escaped)
	case FilterModeStartsWith:
		return fmt.Sprintf("%s LIKE '%s%%'", lowerColumn, escaped)
	case FilterModeEndsWith:
		return fmt.Sprintf("%s LIKE '%%%s'", lowerColumn, escaped)
	default:
		return fmt.Sprintf("%s LIKE '%%%s%%'", lowerColumn, escaped)
	}
}

func cloneExplorerFilters(filters []ExplorerFilterClause) []ExplorerFilterClause {
	if len(filters) == 0 {
		return nil
	}
	out := make([]ExplorerFilterClause, len(filters))
	copy(out, filters)
	return out
}

func resolveExplorerFilters(filters []ExplorerFilterClause, columns []string) []ExplorerFilterClause {
	if len(filters) == 0 {
		return nil
	}

	resolved := make([]ExplorerFilterClause, 0, len(filters))
	for _, clause := range filters {
		matched := matchColumn(columns, clause.Column)
		if matched == "" {
			continue
		}
		resolved = append(resolved, ExplorerFilterClause{
			Column: matched,
			Mode:   clause.Mode,
			Value:  clause.Value,
		})
	}
	return resolved
}

func buildExplorerWhereClause(filters []ExplorerFilterClause) string {
	if len(filters) == 0 {
		return ""
	}

	clauses := make([]string, 0, len(filters))
	for _, clause := range filters {
		if clause.Column == "" || clause.Mode == FilterModeNone {
			continue
		}
		condition := buildFilterCondition(clause.Column, clause.Mode, clause.Value)
		if condition != "" {
			clauses = append(clauses, condition)
		}
	}

	if len(clauses) == 0 {
		return ""
	}

	if len(clauses) == 1 {
		return clauses[0]
	}

	return strings.Join(clauses, " AND ")
}

func buildExplorerOrderClause(columns []string, orderColumn string, descending bool, dbType string) string {
	direction := "ASC"
	if descending {
		direction = "DESC"
	}

	if orderColumn != "" {
		return fmt.Sprintf("ORDER BY %s %s", orderColumn, direction)
	}

	fallback := fallbackOrderColumn(columns)
	if fallback != "" {
		return fmt.Sprintf("ORDER BY %s %s", fallback, direction)
	}

	if strings.EqualFold(dbType, "mssql") {
		return fmt.Sprintf("ORDER BY 1 %s", direction)
	}

	return ""
}

func buildExplorerCountQuery(tableName, whereClause string) string {
	if whereClause == "" {
		return fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	}
	return fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", tableName, whereClause)
}

func buildExplorerSelectQuery(tableName, whereClause, orderClause string, page, pageSize int, dbType string) string {
	if page < 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := page * pageSize

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("SELECT * FROM %s", tableName))
	if whereClause != "" {
		builder.WriteString(" WHERE ")
		builder.WriteString(whereClause)
	}
	if orderClause != "" {
		builder.WriteString(" ")
		builder.WriteString(orderClause)
	}

	if strings.EqualFold(dbType, "mssql") {
		builder.WriteString(fmt.Sprintf(" OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, pageSize))
	} else {
		builder.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset))
	}

	return builder.String()
}

// getTableRowCount gets the total number of rows in a table
func (ui *Gui) getTableRowCount(tableName string) int {
	if ui.app.DB == nil || !ui.app.DB.IsConnected() {
		return 0
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	rows, err := ui.app.DB.ExecuteQuery(query)
	if err != nil {
		ui.app.Events.Dispatch(events.Debugf("gui", "Error counting rows in %s: %v", tableName, err))
		return 0
	}
	defer rows.Close()

	if rows.Next() {
		var count int
		if err := rows.Scan(&count); err == nil {
			return count
		}
	}

	return 0
}

// getTableColumns gets the column names for a table
func (ui *Gui) getTableColumns(tableName string) []string {
	db := ui.app.DB
	if db == nil || !db.IsConnected() {
		return nil
	}

	if columns, err := db.GetTableColumns(tableName); err == nil && len(columns) > 0 {
		return columns
	}

	var query string
	switch strings.ToLower(db.GetType()) {
	case "mssql":
		query = fmt.Sprintf("SELECT TOP 1 * FROM %s", tableName)
	default:
		query = fmt.Sprintf("SELECT * FROM %s LIMIT 1", tableName)
	}

	rows, err := db.ExecuteQuery(query)
	if err != nil {
		if ui.app != nil && ui.app.Events != nil {
			ui.app.Events.Dispatch(events.Debugf("gui", "Failed to inspect columns for %s: %v", tableName, err))
		}
		return nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		if ui.app != nil && ui.app.Events != nil {
			ui.app.Events.Dispatch(events.Debugf("gui", "Failed to read column metadata for %s: %v", tableName, err))
		}
		return nil
	}

	return columns
}
