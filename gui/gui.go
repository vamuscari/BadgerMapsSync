package gui

import (
	"fmt"
	"image/color"
	"os"
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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"badgermaps/app"
	"badgermaps/app/action"
	"badgermaps/app/push"
	appserver "badgermaps/app/server"
	"badgermaps/database"
	"badgermaps/events"
)

// SecondaryButton is a custom button that can be styled with a secondary color
type SecondaryButton struct {
	widget.Button
	TextColor       color.Color
	VerticalPadding float32
}

// NewSecondaryButton creates a new SecondaryButton
func NewSecondaryButton(label string, icon fyne.Resource, tapped func()) *SecondaryButton {
	b := &SecondaryButton{}
	b.Text = label
	b.Icon = icon
	b.OnTapped = tapped
	b.VerticalPadding = theme.Padding()
	b.ExtendBaseWidget(b)
	return b
}

// CreateRenderer implements the Widget interface
func (b *SecondaryButton) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(theme.ButtonColor())
	background.CornerRadius = theme.InputRadiusSize()
	r := &secondaryButtonRenderer{
		button:     b,
		label:      canvas.NewText(b.Text, theme.ForegroundColor()),
		icon:       widget.NewIcon(b.Icon),
		background: background,
	}
	r.objects = []fyne.CanvasObject{r.background, r.icon, r.label}
	return r
}

type secondaryButtonRenderer struct {
	button     *SecondaryButton
	label      *canvas.Text
	icon       *widget.Icon
	background *canvas.Rectangle
	objects    []fyne.CanvasObject
}

func (r *secondaryButtonRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	hPadding := theme.Padding()
	vPadding := r.button.VerticalPadding
	if vPadding < 0 {
		vPadding = 0
	}
	labelSize := r.label.MinSize()
	if r.button.Icon != nil {
		iconSize := theme.IconInlineSize()
		totalWidth := iconSize + hPadding + labelSize.Width
		if totalWidth > size.Width {
			totalWidth = size.Width
		}
		startX := (size.Width - totalWidth) / 2
		contentTop := vPadding
		contentHeight := size.Height - (vPadding * 2)
		if contentHeight <= 0 {
			contentTop = 0
			contentHeight = size.Height
		}
		iconY := contentTop + (contentHeight-iconSize)/2
		r.icon.Resize(fyne.NewSize(iconSize, iconSize))
		r.icon.Move(fyne.NewPos(startX, iconY))
		r.label.Move(fyne.NewPos(startX+iconSize+hPadding, contentTop+(contentHeight-labelSize.Height)/2))
	} else {
		contentTop := vPadding
		contentHeight := size.Height - (vPadding * 2)
		if contentHeight <= 0 {
			contentTop = 0
			contentHeight = size.Height
		}
		r.label.Move(fyne.NewPos((size.Width-labelSize.Width)/2, contentTop+(contentHeight-labelSize.Height)/2))
	}
}

func (r *secondaryButtonRenderer) MinSize() fyne.Size {
	iconSize := theme.IconInlineSize()
	hPadding := theme.Padding()
	vPadding := r.button.VerticalPadding
	if vPadding < 0 {
		vPadding = 0
	}
	min := r.label.MinSize()
	if r.button.Icon != nil {
		min.Width += iconSize + hPadding
	}
	min.Width += hPadding * 2
	min.Height += vPadding * 2
	return min
}

func (r *secondaryButtonRenderer) Refresh() {
	r.label.Text = r.button.Text
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
	if r.button.Disabled() {
		r.label.Color = theme.DisabledColor()
	} else if r.button.TextColor != nil {
		r.label.Color = r.button.TextColor
	} else {
		r.label.Color = theme.ForegroundColor()
	}
	r.background.CornerRadius = theme.InputRadiusSize()
	r.background.Refresh()
	canvas.Refresh(r.label)
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

var rightPaneWidth float32 = 360

type rightPaneSection string

const (
	rightPaneSectionDetails rightPaneSection = "details"
	rightPaneSectionLog     rightPaneSection = "log"
	rightPaneSectionJobs    rightPaneSection = "jobs"
)

// SetDefaultRightPaneWidth allows external utilities (e.g., screenshot generator)
// to adjust the default width of the right details pane.
func SetDefaultRightPaneWidth(w float32) {
	if w > 0 {
		rightPaneWidth = w
	}
}

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
	rightPaneSection      rightPaneSection
	rightPaneJobsView     fyne.CanvasObject
	rightPaneJobsActivity *widget.Label
	rightPaneJobsList     *widget.List
	rightPaneJobsLines    []string
	rightPaneContent      *fyne.Container
	rightPaneOverlay      *fyne.Container
	rightPaneBackdrop     fyne.CanvasObject
	rightPaneToggleButton *widget.Button
	rightPaneVisible      bool
	rightPaneJobsStopMu   sync.Mutex
	rightPaneJobsStopCh   chan struct{}
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
	return ui.newSectionCardWithBackground(title, subtitle, nil, content...)
}

func (ui *Gui) newSectionCardWithBackground(title, subtitle string, backgroundColor color.Color, content ...fyne.CanvasObject) fyne.CanvasObject {
	return ui.newSectionCardWithHeaderControls(title, subtitle, nil, backgroundColor, content...)
}

func (ui *Gui) newSectionCardWithHeaderControls(title, subtitle string, headerControls fyne.CanvasObject, backgroundColor color.Color, content ...fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header := container.NewHBox(titleLabel)
	if headerControls != nil {
		header = container.NewHBox(titleLabel, layout.NewSpacer(), headerControls)
	}

	body := container.NewVBox(header)
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
	if backgroundColor == nil {
		backgroundColor = ui.themeColor(StatusCardBackgroundColorName)
	}
	background := canvas.NewRectangle(backgroundColor)
	background.CornerRadius = theme.Padding()
	background.StrokeColor = ui.themeColor(StatusCardBorderColorName)
	background.StrokeWidth = 1

	return container.NewStack(
		background,
		container.NewPadded(body),
	)
}

func (ui *Gui) newScheduledJobItemCard(title, subtitle string, controls fyne.CanvasObject, details fyne.CanvasObject, backgroundColor color.Color) fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	headerLeft := container.NewVBox(titleLabel)
	if strings.TrimSpace(subtitle) != "" {
		subtitleLabel := widget.NewLabel(subtitle)
		subtitleLabel.Alignment = fyne.TextAlignLeading
		subtitleLabel.Wrapping = fyne.TextWrapWord
		headerLeft.Add(subtitleLabel)
	}

	var rightHeader fyne.CanvasObject = layout.NewSpacer()
	if controls != nil {
		rightHeader = controls
	}

	header := container.NewBorder(
		nil,
		nil,
		nil,
		rightHeader,
		headerLeft,
	)

	body := container.NewVBox(header)
	if details != nil {
		body.Add(widget.NewSeparator())
		body.Add(details)
	}

	if backgroundColor == nil {
		backgroundColor = ui.themeColor(StatusCardBackgroundColorName)
	}
	background := canvas.NewRectangle(backgroundColor)
	background.CornerRadius = theme.Padding()
	background.StrokeColor = ui.themeColor(StatusCardBorderColorName)
	background.StrokeWidth = 1

	return container.NewStack(
		background,
		container.NewPadded(body),
	)
}

func (ui *Gui) syncCenterCardBackground(index int) color.Color {
	baseColor := ui.themeColor(StatusCardBackgroundColorName)
	if baseColor == nil {
		baseColor = theme.ButtonColor()
	}
	baseRGBA := color.NRGBAModel.Convert(baseColor).(color.NRGBA)

	var variantOffsets [][3]int
	if ui != nil && ui.fyneApp != nil && ui.fyneApp.Settings().ThemeVariant() == theme.VariantLight {
		// Light mode needs stronger negative offsets so nested cards stand out on pale surfaces.
		variantOffsets = [][3]int{
			{0, 0, 0},       // Section background
			{-6, -6, -8},    // Empty-state/alternate
			{-24, -22, -18}, // Job cards
			{-12, -10, -8},
			{-20, -18, -14},
			{-10, -8, -6},
		}
	} else {
		// Dark mode uses gentle hue offsets to avoid flat monochrome surfaces.
		variantOffsets = [][3]int{
			{10, 4, -2},
			{4, 10, -2},
			{-2, 6, 10},
			{8, -2, 6},
			{-4, 4, 8},
			{6, 6, -4},
		}
	}

	offset := variantOffsets[index%len(variantOffsets)]
	return color.NRGBA{
		R: clampColorChannel(int(baseRGBA.R) + offset[0]),
		G: clampColorChannel(int(baseRGBA.G) + offset[1]),
		B: clampColorChannel(int(baseRGBA.B) + offset[2]),
		A: baseRGBA.A,
	}
}

func clampColorChannel(value int) uint8 {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
}

// CreateContent exposes the internal createContent for programmatic consumers (e.g., screenshot generator)
func (ui *Gui) CreateContent() fyne.CanvasObject { return ui.createContent() }

// Tabs exposes the main AppTabs for external utilities
func (ui *Gui) Tabs() *container.AppTabs { return ui.tabs }

// ExplorerTableSelect exposes the table selector for the Explorer tab
func (ui *Gui) ExplorerTableSelect() *widget.Select { return ui.explorerTableSelect }

// SyncCenter returns the SyncCenter instance
func (ui *Gui) SyncCenter() *SyncCenter { return ui.syncCenter }

// FyneApp returns the underlying fyne.App
func (ui *Gui) FyneApp() fyne.App { return ui.fyneApp }

// NewGuiForScreenshots builds a minimal GUI instance suitable for headless rendering (tests/screenshots).
func NewGuiForScreenshots(a *app.App, fyApp fyne.App) *Gui {
	ui := &Gui{
		app:        a,
		fyneApp:    fyApp,
		logBinding: binding.NewStringList(),
	}
	ui.presenter = NewGuiPresenter(a, ui)
	ui.tableFactory = NewTableFactory(ui)
	ui.syncCenter = NewSyncCenter(ui, ui.presenter)
	ui.smartDashboard = NewSmartDashboard(ui, ui.presenter)
	ui.showWelcome = false
	// Ensure our custom theme is applied for headless rendering
	ui.fyneApp.Settings().SetTheme(newModernTheme())
	return ui
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
				// Avoid early crash if list is not yet fully initialised
				if ui.logView != nil {
					defer func() { _ = recover() }()
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

	// Update Home dashboard server status in place when the server starts/stops.
	serverStatusListener := func(e events.Event) {
		fyne.Do(func() {
			if ui.smartDashboard != nil {
				ui.smartDashboard.RefreshDashboard()
			}
			if ui.syncCenter != nil {
				ui.syncCenter.RefreshServerStatusLabel()
			}
		})
	}
	a.Events.Subscribe("server.status.changed", serverStatusListener)

	serverStatusWatcherStop := make(chan struct{})
	startServerStatusWatcher(a, serverStatusWatcherStop)
	defer close(serverStatusWatcherStop)

	// Optional launch scaling for screenshots or HiDPI preview
	baseW, baseH := float32(1000), float32(600)
	scale := float32(1.0)
	if s := strings.TrimSpace(os.Getenv("BM_GUI_SCALE")); s != "" {
		if v, err := strconv.ParseFloat(s, 32); err == nil && v > 0 {
			scale = float32(v)
		}
	} else if s := strings.TrimSpace(os.Getenv("GUI_WINDOW_SCALE")); s != "" {
		if v, err := strconv.ParseFloat(s, 32); err == nil && v > 0 {
			scale = float32(v)
		}
	}

	// Scale details pane proportionally if requested
	if scale > 1.0 {
		SetDefaultRightPaneWidth(360 * scale)
	}

	window.SetContent(ui.createContent())
	// Set initial size and allow resizing
	window.Resize(fyne.NewSize(baseW*scale, baseH*scale))
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

	syncTab := container.NewTabItemWithIcon("Sync Center", theme.HistoryIcon(), syncContent)
	explorerTab := container.NewTabItemWithIcon("Explorer", theme.FolderIcon(), explorerContent)

	tabs := []*container.TabItem{
		homeTab,
		syncTab,
		explorerTab,
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
	ui.rightPaneSection = rightPaneSectionDetails
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

// refreshAllTabs rebuilds the main tabs on the UI thread.
func (ui *Gui) refreshAllTabs() {
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

// RefreshAllTabs rebuilds the main tabs, which is useful when connection status changes.
func (ui *Gui) RefreshAllTabs() {
	fyne.Do(func() {
		ui.refreshAllTabs()
	})
}

// refreshHomeTab rebuilds and refreshes the home tab on the UI thread.
func (ui *Gui) refreshHomeTab() {
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

func (ui *Gui) RefreshHomeTab() {
	fyne.Do(func() {
		ui.refreshHomeTab()
	})
}

func (ui *Gui) refreshActionsTab() {
	ui.refreshSyncCenterTab()
}

func (ui *Gui) RefreshActionsTab() {
	ui.RefreshSyncCenterTab()
}

func (ui *Gui) refreshSyncCenterTab() {
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

	syncContent := ui.createDisabledTabView(configTabItem)
	if ui.app.API != nil && ui.app.API.IsConnected() && ui.app.DB != nil && ui.app.DB.IsConnected() {
		syncContent = ui.syncCenter.CreateContent()
	}

	for _, tab := range ui.tabs.Items {
		if tab.Text == "Sync Center" {
			tab.Content = syncContent
			ui.tabs.Refresh()
			return
		}
	}
}

func (ui *Gui) RefreshSyncCenterTab() {
	fyne.Do(func() {
		ui.refreshSyncCenterTab()
	})
}

func (ui *Gui) createRightPaneHeader() fyne.CanvasObject {
	detailsButton := widget.NewButtonWithIcon("Details", theme.ListIcon(), func() {
		ui.selectRightPaneSection(rightPaneSectionDetails)
		ui.showRightPane()
	})

	logButton := widget.NewButtonWithIcon("Log", theme.ComputerIcon(), func() {
		ui.selectRightPaneSection(rightPaneSectionLog)
		ui.showRightPane()
	})

	jobsButton := widget.NewButtonWithIcon("Jobs", theme.HistoryIcon(), func() {
		ui.selectRightPaneSection(rightPaneSectionJobs)
		ui.showRightPane()
	})

	buttonRow := container.NewHBox(detailsButton, logButton, jobsButton)
	return container.NewBorder(nil, nil, nil, nil, buttonRow)
}

func (ui *Gui) selectRightPaneSection(section rightPaneSection) {
	ui.rightPaneSection = section
	switch section {
	case rightPaneSectionLog:
		ui.terminalVisible = true
		if ui.logView != nil {
			ui.setRightPaneContent(ui.logView)
		}
	case rightPaneSectionJobs:
		ui.terminalVisible = false
		ui.setRightPaneContent(ui.ensureRightPaneJobsView())
		ui.refreshRightPaneJobs()
	default:
		ui.terminalVisible = false
		if ui.detailsView != nil {
			ui.setRightPaneContent(ui.detailsView)
		}
	}
	ui.manageRightPaneJobsAutoRefresh()
}

func (ui *Gui) ensureRightPaneJobsView() fyne.CanvasObject {
	if ui.rightPaneJobsView != nil {
		return ui.rightPaneJobsView
	}

	ui.rightPaneJobsLines = []string{"Start the server to view active and queued jobs."}
	ui.rightPaneJobsList = widget.NewList(
		func() int {
			return len(ui.rightPaneJobsLines)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("template")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			object.(*widget.Label).SetText(ui.rightPaneJobsLines[id])
		},
	)
	jobsListContainer := container.NewVScroll(ui.rightPaneJobsList)
	jobsListContainer.SetMinSize(fyne.NewSize(0, 220))

	ui.rightPaneJobsActivity = widget.NewLabel("Server jobs will appear here once the server is active.")
	ui.rightPaneJobsActivity.Wrapping = fyne.TextWrapWord

	ui.rightPaneJobsView = ui.newSectionCard(
		"Jobs",
		"View active and queued server sync jobs.",
		ui.rightPaneJobsActivity,
		jobsListContainer,
	)

	ui.refreshRightPaneJobs()
	return ui.rightPaneJobsView
}

func (ui *Gui) refreshRightPaneJobs() {
	if ui.rightPaneJobsActivity == nil || ui.rightPaneJobsList == nil {
		return
	}

	displayLoc := ui.app.ServerTimezoneLocation()
	timezoneLabel := displayLoc.String()
	snapshot, err := ui.presenter.FetchServerJobsSnapshot()
	if err != nil {
		ui.rightPaneJobsActivity.SetText(fmt.Sprintf("Unable to load server jobs: %v (display TZ: %s)", err, timezoneLabel))
		ui.rightPaneJobsLines = []string{"Start the server to view active and queued jobs."}
		ui.rightPaneJobsList.SetItemHeight(0, rightPaneJobsItemHeight(ui.rightPaneJobsLines[0]))
		ui.rightPaneJobsList.Refresh()
		return
	}

	ui.rightPaneJobsActivity.SetText(formatServerActivityLine(snapshot.Activity, displayLoc))
	filtered := filterActiveAndQueuedJobs(snapshot.Jobs)
	if len(filtered) == 0 {
		ui.rightPaneJobsLines = []string{"No active or queued jobs."}
		ui.rightPaneJobsList.SetItemHeight(0, rightPaneJobsItemHeight(ui.rightPaneJobsLines[0]))
		ui.rightPaneJobsList.Refresh()
		return
	}

	lines := make([]string, 0, len(filtered))
	for _, job := range filtered {
		lines = append(lines, formatServerJobDisplayLine(formatServerJobLine(job, displayLoc)))
	}
	ui.rightPaneJobsLines = lines
	for idx, line := range ui.rightPaneJobsLines {
		ui.rightPaneJobsList.SetItemHeight(idx, rightPaneJobsItemHeight(line))
	}
	ui.rightPaneJobsList.Refresh()
}

func (ui *Gui) manageRightPaneJobsAutoRefresh() {
	if ui.rightPaneVisible && ui.rightPaneSection == rightPaneSectionJobs {
		ui.startRightPaneJobsAutoRefresh()
		return
	}
	ui.stopRightPaneJobsAutoRefresh()
}

func (ui *Gui) startRightPaneJobsAutoRefresh() {
	ui.rightPaneJobsStopMu.Lock()
	defer ui.rightPaneJobsStopMu.Unlock()

	if ui.rightPaneJobsStopCh != nil {
		return
	}

	stopCh := make(chan struct{})
	ui.rightPaneJobsStopCh = stopCh

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				fyne.Do(func() {
					ui.refreshRightPaneJobs()
				})
			}
		}
	}()
}

func (ui *Gui) stopRightPaneJobsAutoRefresh() {
	ui.rightPaneJobsStopMu.Lock()
	defer ui.rightPaneJobsStopMu.Unlock()

	if ui.rightPaneJobsStopCh == nil {
		return
	}
	close(ui.rightPaneJobsStopCh)
	ui.rightPaneJobsStopCh = nil
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
		ui.selectRightPaneSection(ui.rightPaneSection)
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
	if ui.rightPaneSection == rightPaneSectionJobs {
		ui.refreshRightPaneJobs()
	}
	ui.manageRightPaneJobsAutoRefresh()
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
	ui.manageRightPaneJobsAutoRefresh()
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
	fyne.Do(func() {
		ui.setDetails(details, true)
	})
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
	ui.rightPaneSection = rightPaneSectionDetails
	ui.terminalVisible = false
	ui.setRightPaneContent(ui.detailsView)
	ui.manageRightPaneJobsAutoRefresh()
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

func (ui *Gui) refreshPushTab() {
	ui.refreshSyncCenterTab()
}

func (ui *Gui) RefreshPushTab() {
	fyne.Do(func() {
		ui.refreshPushTab()
	})
}

func (ui *Gui) formatTimestampInDisplayTimezone(ts time.Time, layout string) string {
	if ui != nil && ui.app != nil {
		return ui.app.FormatTimestampInDisplayTimezone(ts, layout)
	}

	trimmedLayout := strings.TrimSpace(layout)
	if trimmedLayout == "" {
		trimmedLayout = time.RFC3339
	}
	return fmt.Sprintf("%s [%s]", ts.In(time.Local).Format(trimmedLayout), time.Local.String())
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
				ui.formatTimestampInDisplayTimezone(c.CreatedAt, time.RFC3339),
				c.Changes,
			})
		}
	case "checkins":
		headers = []string{"ID", "Checkin ID", "Account ID", "Change Type", "Endpoint", "Checkin Type", "Status", "Created At", "Comments"}
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
				c.EndpointType.String,
				c.Type.String,
				c.Status,
				ui.formatTimestampInDisplayTimezone(c.CreatedAt, time.RFC3339),
				c.Comments.String,
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
	notice := widget.NewLabel("Global Actions are retired. Configure action steps under Jobs workflow steps.")
	notice.Wrapping = fyne.TextWrapWord
	return ui.newSectionCard("Actions", "Deprecated", notice)
}

func (ui *Gui) createScheduledJobsSection() fyne.CanvasObject {
	jobsContent := container.NewVBox()
	sectionBackground := ui.syncCenterCardBackground(0)
	emptyStateBackground := ui.syncCenterCardBackground(1)

	jobs, err := ui.app.ListScheduledJobs()
	if err != nil {
		errorLabel := widget.NewLabel(fmt.Sprintf("Unable to load scheduled jobs: %v", err))
		errorLabel.Wrapping = fyne.TextWrapWord
		jobsContent.Add(ui.newSectionCardWithBackground("Jobs", "Failed to load jobs", emptyStateBackground, errorLabel))
	} else {
		if len(jobs) == 0 {
			jobsContent.Add(ui.newSectionCardWithBackground(
				"Jobs",
				"No scheduled jobs configured yet.",
				emptyStateBackground,
				widget.NewLabel("Use the button below to add scheduled pull/push automation."),
			))
		}

		jobCardBackground := ui.syncCenterCardBackground(2)
		for _, job := range jobs {
			jb := job
			stateLabel := "Active"
			if !jb.Enabled {
				stateLabel = "Paused"
			}
			effectiveTimezone, timezoneSource := ui.app.EffectiveScheduledJobTimezone(jb)
			timezoneSummary := fmt.Sprintf("%s (%s)", effectiveTimezone, timezoneSource)
			stepSummaries := make([]string, 0, len(jb.Steps))
			for _, step := range jb.Steps {
				stepLabel := step.EffectiveName()
				if step.Type == appserver.WorkflowStepTypeSync {
					stepLabel = fmt.Sprintf("%s (%s)", stepLabel, step.SyncMode)
				}
				stepSummaries = append(stepSummaries, stepLabel)
			}
			stepsText := strings.Join(stepSummaries, "\n- ")
			if stepsText != "" {
				stepsText = "- " + stepsText
			} else {
				stepsText = "(no steps)"
			}
			summary := widget.NewLabel(fmt.Sprintf(
				"ID: %s\nSchedule: %s\nSteps (%d):\n%s\nRetries: %d (retry_on_error=%t)\nTimezone: %s\nState: %s",
				jb.ID,
				jb.Schedule,
				len(jb.Steps),
				stepsText,
				jb.MaxRetries,
				jb.RetryOnError,
				timezoneSummary,
				stateLabel,
			))
			summary.Wrapping = fyne.TextWrapWord

			pauseIcon := theme.MediaPauseIcon()
			pauseVerb := "Paused"
			nextEnabled := false
			pauseTooltip := "Pause job"
			if !jb.Enabled {
				pauseIcon = theme.MediaPlayIcon()
				pauseVerb = "Resumed"
				nextEnabled = true
				pauseTooltip = "Resume job"
			}

			hoverHint := widget.NewLabel(" ")
			hintHeight := hoverHint.MinSize().Height
			if hintHeight < theme.TextSize()+theme.Padding() {
				hintHeight = theme.TextSize() + theme.Padding()
			}
			hoverHintSlot := container.NewGridWrap(fyne.NewSize(96, hintHeight), hoverHint)
			setHoverHint := func(text string) {
				if strings.TrimSpace(text) == "" {
					hoverHint.SetText(" ")
					return
				}
				hoverHint.SetText(text)
			}

			pauseButton := newTooltipIconButton(pauseIcon, pauseTooltip, setHoverHint, func() {
				applyJobEnabled := func() error {
					return ui.app.SetScheduledJobEnabled(jb.ID, nextEnabled)
				}

				if err := applyJobEnabled(); err != nil {
					if strings.Contains(err.Error(), "stop the server first") {
						actionVerb := strings.ToLower(pauseVerb)
						dialog.ShowConfirm(
							"Stop Server First",
							fmt.Sprintf("The server is active. Stop it now and %s job '%s'?", actionVerb, jb.Name),
							func(confirm bool) {
								if !confirm {
									return
								}

								if stopErr := ui.app.Server.StopServer(); stopErr != nil {
									ui.app.Events.Dispatch(events.Errorf("gui", "Error stopping server before job update: %v", stopErr))
									ui.ShowToast(fmt.Sprintf("Error: %v", stopErr))
									return
								}
								ui.app.Events.Dispatch(events.Event{Type: "server.status.changed", Source: "gui"})

								if retryErr := applyJobEnabled(); retryErr != nil {
									ui.app.Events.Dispatch(events.Errorf("gui", "Error updating job state after stop: %v", retryErr))
									ui.ShowToast(fmt.Sprintf("Error: %v", retryErr))
									return
								}
								ui.ShowToast(fmt.Sprintf("%s job '%s'.", pauseVerb, jb.Name))
								ui.refreshSyncCenterTab()
							},
							ui.window,
						)
						return
					}

					ui.app.Events.Dispatch(events.Errorf("gui", "Error updating job state: %v", err))
					ui.ShowToast(fmt.Sprintf("Error: %v", err))
					return
				}
				ui.ShowToast(fmt.Sprintf("%s job '%s'. Restart server if already running.", pauseVerb, jb.Name))
				ui.refreshSyncCenterTab()
			})

			runNowButton := newTooltipIconButton(theme.MediaSkipNextIcon(), "Run job now", setHoverHint, func() {
				if ui.presenter == nil {
					ui.ShowToast("Error: Unable to run job right now.")
					return
				}
				ui.presenter.HandleRunScheduledJob(jb.ID, jb.Name)
			})

			editButton := newTooltipIconButton(theme.DocumentCreateIcon(), "Edit job", setHoverHint, func() {
				ui.createJobPopup(jb)
			})

			deleteButton := newTooltipIconButton(theme.DeleteIcon(), "Delete job", setHoverHint, func() {
				dialog.ShowConfirm("Delete Job", "Delete this scheduled job?", func(confirm bool) {
					if !confirm {
						return
					}
					if err := ui.app.DeleteScheduledJob(jb.ID); err != nil {
						ui.app.Events.Dispatch(events.Errorf("gui", "Error deleting job: %v", err))
						ui.ShowToast(fmt.Sprintf("Error: %v", err))
						return
					}
					ui.ShowToast(fmt.Sprintf("Deleted job '%s'.", jb.Name))
					ui.refreshSyncCenterTab()
				}, ui.window)
			})

			actionsRow := container.NewHBox(
				hoverHintSlot,
				NewSpacer(fyne.NewSize(theme.Padding(), 0)),
				pauseButton,
				runNowButton,
				editButton,
				deleteButton,
			)

			title := jb.Name
			if strings.TrimSpace(title) == "" {
				title = jb.ID
			}
			if !jb.Enabled {
				title = fmt.Sprintf("[Paused] %s", title)
			}

			jobsContent.Add(ui.newScheduledJobItemCard(
				title,
				"",
				actionsRow,
				summary,
				jobCardBackground,
			))
		}
	}

	addButton := NewSecondaryButton("Add Job", theme.NewColoredResource(theme.ContentAddIcon(), AddJobAccentColorName), func() {
		ui.createJobPopup(nil)
	})
	addButton.VerticalPadding = theme.Padding() * 0.35
	addButton.TextColor = ui.themeColor(AddJobAccentColorName)
	headerControls := container.NewVBox(
		NewSpacer(fyne.NewSize(0, theme.Padding()*0.25)),
		container.NewHBox(
			addButton,
			NewSpacer(fyne.NewSize(theme.Padding(), 0)),
		),
	)

	notice := widget.NewLabel("Stop the server before editing jobs. Changes are written immediately and loaded on next start.")
	notice.Wrapping = fyne.TextWrapWord

	return ui.newSectionCardWithHeaderControls(
		"Scheduled Jobs",
		"Manage recurring sync workflows.",
		headerControls,
		sectionBackground,
		container.NewVBox(
			jobsContent,
			widget.NewSeparator(),
			notice,
		),
	)
}

func (ui *Gui) createJobPopup(existing *appserver.ScheduledJob) {
	job := appserver.ScheduledJob{
		Enabled:      true,
		MaxRetries:   1,
		RetryOnError: false,
	}
	if existing != nil {
		job = *existing
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Nightly pull")
	nameEntry.SetText(job.Name)

	cronWidget := NewCronWidget()
	advancedScheduleEntry := widget.NewEntry()
	advancedScheduleEntry.SetPlaceHolder("Advanced schedule, e.g. @daily")
	advancedScheduleEntry.SetText(strings.TrimSpace(job.Schedule))

	scheduleParseMode, parsedFields, rawScheduleValue, scheduleParseErr := ResolveCronScheduleForEditor(job.Schedule)
	if scheduleParseMode == ParseModeFields {
		cronWidget.SetFields(parsedFields)
	} else if strings.TrimSpace(rawScheduleValue) != "" {
		advancedScheduleEntry.SetText(rawScheduleValue)
	}
	currentScheduleMode := scheduleParseMode
	scheduleModeSelector := widget.NewRadioGroup([]string{"Fields", "Advanced"}, nil)
	scheduleModeSelector.Horizontal = true
	scheduleFieldsContainer := container.NewVBox(cronWidget.Object())
	scheduleAdvancedContainer := container.NewVBox(advancedScheduleEntry)

	updateScheduleEditorVisibility := func() {
		if currentScheduleMode == ParseModeRaw {
			scheduleFieldsContainer.Hide()
			scheduleAdvancedContainer.Show()
		} else {
			scheduleFieldsContainer.Show()
			scheduleAdvancedContainer.Hide()
		}
	}

	suppressScheduleModeChange := false
	setScheduleMode := func(mode ParseMode) {
		currentScheduleMode = mode
		suppressScheduleModeChange = true
		if mode == ParseModeRaw {
			scheduleModeSelector.SetSelected("Advanced")
		} else {
			scheduleModeSelector.SetSelected("Fields")
		}
		suppressScheduleModeChange = false
		updateScheduleEditorVisibility()
	}
	scheduleModeSelector.OnChanged = func(selected string) {
		if suppressScheduleModeChange {
			return
		}
		switch strings.TrimSpace(selected) {
		case "Advanced":
			currentScheduleMode = ParseModeRaw
		default:
			currentScheduleMode = ParseModeFields
		}
		updateScheduleEditorVisibility()
	}

	profileSelect := widget.NewSelect([]string{}, nil)
	profileSelect.PlaceHolder = "Select profile template (optional)"

	enabledCheck := widget.NewCheck("Enabled", nil)
	enabledCheck.SetChecked(job.Enabled)

	retryCheck := widget.NewCheck("Retry on error", nil)
	retryCheck.SetChecked(job.RetryOnError)

	maxRetriesEntry := widget.NewEntry()
	if job.MaxRetries <= 0 {
		maxRetriesEntry.SetText("1")
	} else {
		maxRetriesEntry.SetText(strconv.Itoa(job.MaxRetries))
	}

	timezoneEntry := widget.NewEntry()
	timezoneEntry.SetPlaceHolder("Optional override, e.g. America/New_York")
	timezoneEntry.SetText(job.Timezone)

	scheduleHelp := widget.NewLabel("Use six cron fields (second minute hour day month weekday), or set an advanced raw schedule like @daily/@every. Advanced raw takes precedence when provided.")
	if scheduleParseMode == ParseModeRaw {
		if scheduleParseErr != nil {
			scheduleHelp.SetText(fmt.Sprintf(
				"Existing schedule is treated as advanced raw because it could not be mapped to cron fields: %v",
				scheduleParseErr,
			))
		} else {
			scheduleHelp.SetText("Existing schedule is in advanced raw mode. Keep the advanced value, or clear it to use cron fields.")
		}
	}
	scheduleHelp.Wrapping = fyne.TextWrapWord
	setScheduleMode(currentScheduleMode)
	scheduleEditorContainer := container.NewVBox(scheduleFieldsContainer, scheduleAdvancedContainer)

	canonicalSteps := cloneWorkflowSteps(job.Steps)
	advancedJSON, err := workflowStepsToJSON(canonicalSteps)
	if err != nil {
		advancedJSON = "[]"
	}
	stepsEntry := widget.NewMultiLineEntry()
	stepsEntry.SetPlaceHolder("Workflow steps JSON. Example: [{\"id\":\"pull_accounts\",\"type\":\"sync\",\"sync_type\":\"pull_accounts\"}]")
	stepsEntry.Wrapping = fyne.TextWrapWord
	stepsEntry.SetMinRowsVisible(12)
	stepsEntry.SetText(advancedJSON)

	builderDrafts, builderErr := workflowBuilderDraftsFromSteps(canonicalSteps)
	initialBuilderCompatible := builderErr == nil
	if !initialBuilderCompatible {
		builderDrafts = []workflowBuilderDraftStep{}
	}

	currentEditorMode := workflowEditorModeBuilder
	if existing != nil && !initialBuilderCompatible {
		currentEditorMode = workflowEditorModeAdvanced
	}
	modeSelector := widget.NewRadioGroup([]string{workflowEditorModeBuilder, workflowEditorModeAdvanced}, nil)
	modeSelector.Horizontal = true

	editorStatus := widget.NewLabel("")
	editorStatus.Wrapping = fyne.TextWrapWord
	editorStatus.Hide()

	builderCompatibilityWarning := widget.NewLabel("")
	builderCompatibilityWarning.Wrapping = fyne.TextWrapWord
	builderCompatibilityWarning.Hide()
	if existing != nil && !initialBuilderCompatible {
		builderCompatibilityWarning.SetText(fmt.Sprintf("Builder unavailable for current steps: %v", builderErr))
		builderCompatibilityWarning.Show()
	}

	builderPanel := container.NewVBox()
	advancedPanel := container.NewVBox(
		widget.NewLabel("Advanced mode: edit workflow steps as JSON."),
		stepsEntry,
	)
	editorContainer := container.NewVBox(modeSelector, builderCompatibilityWarning, editorStatus, builderPanel, advancedPanel)

	setEditorStatus := func(message string) {
		message = strings.TrimSpace(message)
		if message == "" {
			editorStatus.SetText("")
			editorStatus.Hide()
			return
		}
		editorStatus.SetText(message)
		editorStatus.Show()
	}

	syncBuilderToCanonical := func(showStatus bool) error {
		jsonText, steps, err := workflowBuilderDraftsToJSON(builderDrafts)
		if err != nil {
			if showStatus {
				setEditorStatus(err.Error())
			}
			return err
		}
		canonicalSteps = cloneWorkflowSteps(steps)
		stepsEntry.SetText(jsonText)
		if showStatus {
			setEditorStatus("")
		}
		return nil
	}

	parseAdvancedJSON := func(requireBuilderShape bool, showStatus bool) error {
		steps, err := parseWorkflowStepsJSON(stepsEntry.Text)
		if err != nil {
			if showStatus {
				setEditorStatus(err.Error())
			}
			return err
		}
		if requireBuilderShape {
			drafts, convErr := workflowBuilderDraftsFromSteps(steps)
			if convErr != nil {
				if showStatus {
					setEditorStatus(fmt.Sprintf("Cannot switch to Builder: %v", convErr))
				}
				return convErr
			}
			builderDrafts = drafts
		}
		canonicalSteps = cloneWorkflowSteps(steps)
		jsonText, jsonErr := workflowStepsToJSON(steps)
		if jsonErr == nil {
			stepsEntry.SetText(jsonText)
		}
		if showStatus {
			setEditorStatus("")
		}
		return nil
	}

	syncModeOptions := workflowBuilderSyncModeOptions()

	var renderBuilderPanel func()
	renderBuilderPanel = func() {
		rows := make([]fyne.CanvasObject, 0, len(builderDrafts)+2)
		rows = append(rows, widget.NewLabel("Builder mode: create and reorder workflow steps visually."))

		if len(builderDrafts) == 0 {
			empty := widget.NewLabel("No steps yet. Add a step to begin.")
			empty.Wrapping = fyne.TextWrapWord
			rows = append(rows, empty)
		}

		for i := range builderDrafts {
			idx := i
			draft := &builderDrafts[idx]

			stepIndexLabel := widget.NewLabel(fmt.Sprintf("Step %d", idx+1))
			moveUpBtn := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {
				if idx <= 0 {
					return
				}
				builderDrafts[idx-1], builderDrafts[idx] = builderDrafts[idx], builderDrafts[idx-1]
				_ = syncBuilderToCanonical(true)
				renderBuilderPanel()
			})
			moveDownBtn := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
				if idx >= len(builderDrafts)-1 {
					return
				}
				builderDrafts[idx+1], builderDrafts[idx] = builderDrafts[idx], builderDrafts[idx+1]
				_ = syncBuilderToCanonical(true)
				renderBuilderPanel()
			})
			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				builderDrafts = append(builderDrafts[:idx], builderDrafts[idx+1:]...)
				_ = syncBuilderToCanonical(true)
				renderBuilderPanel()
			})
			header := container.NewHBox(stepIndexLabel, layout.NewSpacer(), moveUpBtn, moveDownBtn, deleteBtn)

			stepNameEntry := widget.NewEntry()
			stepNameEntry.SetPlaceHolder("Optional name")
			stepNameEntry.SetText(draft.Name)
			stepNameEntry.OnChanged = func(value string) {
				draft.Name = value
				_ = syncBuilderToCanonical(false)
			}

			typeSelect := widget.NewSelect([]string{string(appserver.WorkflowStepTypeSync), string(appserver.WorkflowStepTypeAction)}, nil)
			typeSelect.SetSelected(string(draft.Type))
			typeSelect.OnChanged = func(value string) {
				if strings.TrimSpace(value) == "" {
					return
				}
				draft.Type = appserver.WorkflowStepType(value)
				if draft.Type == appserver.WorkflowStepTypeSync {
					if !appserver.IsWorkflowSyncMode(draft.SyncMode) {
						draft.SyncMode = appserver.SyncModePullAccounts
					}
				} else {
					if strings.TrimSpace(draft.ActionType) == "" {
						draft.ActionType = workflowActionTypeExec
					}
				}
				draft.ID = nextWorkflowBuilderStepID(builderDrafts, *draft, idx)
				_ = syncBuilderToCanonical(true)
				renderBuilderPanel()
			}

			commonForm := widget.NewForm(
				widget.NewFormItem("Name", stepNameEntry),
				widget.NewFormItem("Type", typeSelect),
			)

			var detail fyne.CanvasObject
			switch draft.Type {
			case appserver.WorkflowStepTypeSync:
				syncModeSelect := widget.NewSelect(syncModeOptions, nil)
				syncModeSelect.SetSelected(string(draft.SyncMode))
				syncModeSelect.OnChanged = func(value string) {
					draft.SyncMode = appserver.SyncMode(value)
					draft.ID = nextWorkflowBuilderStepID(builderDrafts, *draft, idx)
					_ = syncBuilderToCanonical(false)
					renderBuilderPanel()
				}
				detail = widget.NewForm(widget.NewFormItem("Sync Type", syncModeSelect))
			case appserver.WorkflowStepTypeAction:
				enabledCheck := widget.NewCheck("Enabled", nil)
				enabledCheck.SetChecked(draft.ActionEnabled)
				enabledCheck.OnChanged = func(enabled bool) {
					draft.ActionEnabled = enabled
					_ = syncBuilderToCanonical(false)
				}

				actionTypeSelect := widget.NewSelect([]string{workflowActionTypeExec, workflowActionTypeDB}, nil)
				selectedActionType := strings.TrimSpace(draft.ActionType)
				if selectedActionType == "" {
					selectedActionType = workflowActionTypeExec
					draft.ActionType = selectedActionType
				}
				actionTypeSelect.SetSelected(selectedActionType)
				actionTypeSelect.OnChanged = func(value string) {
					draft.ActionType = strings.TrimSpace(value)
					draft.ID = nextWorkflowBuilderStepID(builderDrafts, *draft, idx)
					_ = syncBuilderToCanonical(true)
					renderBuilderPanel()
				}

				actionForm := widget.NewForm(
					widget.NewFormItem("Enabled", enabledCheck),
					widget.NewFormItem("Action Type", actionTypeSelect),
				)

				var actionDetails fyne.CanvasObject
				switch strings.TrimSpace(draft.ActionType) {
				case workflowActionTypeExec:
					commandEntry := widget.NewEntry()
					commandEntry.SetPlaceHolder("Command")
					commandEntry.SetText(draft.ExecCommand)
					commandEntry.OnChanged = func(value string) {
						draft.ExecCommand = value
						_ = syncBuilderToCanonical(false)
					}

					useShellCheck := widget.NewCheck("Use shell", nil)
					useShellCheck.SetChecked(draft.ExecUseShell)
					useShellCheck.OnChanged = func(enabled bool) {
						draft.ExecUseShell = enabled
						_ = syncBuilderToCanonical(true)
						renderBuilderPanel()
					}

					execFormItems := []*widget.FormItem{
						widget.NewFormItem("Command", commandEntry),
						widget.NewFormItem("Use Shell", useShellCheck),
					}
					if !draft.ExecUseShell {
						argsEntry := widget.NewMultiLineEntry()
						argsEntry.SetPlaceHolder("One arg per line")
						argsEntry.SetText(draft.ExecArgsText)
						argsEntry.Wrapping = fyne.TextWrapWord
						argsEntry.OnChanged = func(value string) {
							draft.ExecArgsText = value
							_ = syncBuilderToCanonical(false)
						}
						execFormItems = append(execFormItems, widget.NewFormItem("Args", argsEntry))
					}
					actionDetails = widget.NewForm(execFormItems...)
				default:
					opSelect := widget.NewSelect(workflowDBOperationKeys, nil)
					selectedOp := strings.TrimSpace(draft.DBOperation)
					if selectedOp == "" {
						selectedOp = "command"
						draft.DBOperation = selectedOp
					}
					opSelect.SetSelected(selectedOp)
					opSelect.OnChanged = func(value string) {
						draft.DBOperation = strings.TrimSpace(value)
						_ = syncBuilderToCanonical(false)
					}

					dbValueEntry := widget.NewEntry()
					dbValueEntry.SetPlaceHolder("DB operation value")
					dbValueEntry.SetText(draft.DBValue)
					dbValueEntry.OnChanged = func(value string) {
						draft.DBValue = value
						_ = syncBuilderToCanonical(false)
					}

					dbArgsEntry := widget.NewMultiLineEntry()
					dbArgsEntry.SetPlaceHolder("Optional args JSON")
					dbArgsEntry.SetText(draft.DBArgsText)
					dbArgsEntry.Wrapping = fyne.TextWrapWord
					dbArgsEntry.OnChanged = func(value string) {
						draft.DBArgsText = value
						_ = syncBuilderToCanonical(false)
					}

					actionDetails = widget.NewForm(
						widget.NewFormItem("Operation", opSelect),
						widget.NewFormItem("Value", dbValueEntry),
						widget.NewFormItem("Args JSON (optional)", dbArgsEntry),
					)
				}

				detail = container.NewVBox(actionForm, actionDetails)
			}

			stepCard := ui.newSectionCard(fmt.Sprintf("Step %d", idx+1), "", container.NewVBox(header, commonForm, detail))
			rows = append(rows, stepCard)
		}

		addStepBtn := widget.NewButtonWithIcon("Add Step", theme.ContentAddIcon(), func() {
			builderDrafts = append(builderDrafts, defaultBuilderDraftStep(builderDrafts))
			_ = syncBuilderToCanonical(true)
			renderBuilderPanel()
		})
		rows = append(rows, addStepBtn)

		builderPanel.Objects = rows
		builderPanel.Refresh()
	}

	updateEditorModeVisibility := func() {
		if currentEditorMode == workflowEditorModeBuilder {
			builderPanel.Show()
			advancedPanel.Hide()
		} else {
			builderPanel.Hide()
			advancedPanel.Show()
		}
	}

	suppressModeChange := false
	setModeSelection := func(mode string) {
		currentEditorMode = mode
		suppressModeChange = true
		modeSelector.SetSelected(mode)
		suppressModeChange = false
		updateEditorModeVisibility()
	}

	modeSelector.OnChanged = func(selection string) {
		if suppressModeChange {
			return
		}
		selection = strings.TrimSpace(selection)
		if selection == "" || selection == currentEditorMode {
			return
		}
		if selection == workflowEditorModeBuilder {
			if err := parseAdvancedJSON(true, true); err != nil {
				ui.ShowToast(fmt.Sprintf("Cannot switch to Builder: %v", err))
				setModeSelection(currentEditorMode)
				return
			}
			builderCompatibilityWarning.Hide()
			renderBuilderPanel()
			setModeSelection(workflowEditorModeBuilder)
			return
		}

		if err := syncBuilderToCanonical(false); err != nil {
			setEditorStatus(err.Error())
		}
		setModeSelection(workflowEditorModeAdvanced)
	}

	profileOptions := sortedWorkflowProfileOptions(ui.app.Config.WorkflowProfiles)
	profileSelect.Options = profileOptions
	suppressProfileChange := false
	profileSelect.OnChanged = func(value string) {
		if suppressProfileChange {
			return
		}
		selected := strings.TrimSpace(value)
		if selected == "" {
			return
		}

		dialog.ShowConfirm(
			"Insert Workflow Template",
			fmt.Sprintf("Replace current workflow steps with template '%s'?", selected),
			func(confirm bool) {
				nextSteps, replaced, applyErr := applyWorkflowProfileTemplateSelection(
					selected,
					canonicalSteps,
					ui.app.Config.WorkflowProfiles,
					confirm,
				)
				if applyErr != nil {
					ui.ShowToast(fmt.Sprintf("Error applying template: %v", applyErr))
				}

				if !replaced {
					suppressProfileChange = true
					profileSelect.ClearSelected()
					suppressProfileChange = false
					return
				}

				canonicalSteps = cloneWorkflowSteps(nextSteps)
				jsonText, jsonErr := workflowStepsToJSON(canonicalSteps)
				if jsonErr == nil {
					stepsEntry.SetText(jsonText)
				}
				drafts, convErr := workflowBuilderDraftsFromSteps(canonicalSteps)
				if convErr == nil {
					builderDrafts = drafts
					builderCompatibilityWarning.Hide()
					if currentEditorMode == workflowEditorModeBuilder {
						renderBuilderPanel()
					}
				} else {
					builderCompatibilityWarning.SetText(fmt.Sprintf("Builder unavailable for selected profile steps: %v", convErr))
					builderCompatibilityWarning.Show()
					setModeSelection(workflowEditorModeAdvanced)
				}
				suppressProfileChange = true
				profileSelect.ClearSelected()
				suppressProfileChange = false
				setEditorStatus("")
			},
			ui.window,
		)
	}

	profileSelect.Refresh()

	renderBuilderPanel()
	setModeSelection(currentEditorMode)

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Schedule Mode", scheduleModeSelector),
		widget.NewFormItem("Schedule Editor", scheduleEditorContainer),
		widget.NewFormItem("", scheduleHelp),
		widget.NewFormItem("Workflow Template Insert (optional)", profileSelect),
		widget.NewFormItem("Workflow Steps", editorContainer),
		widget.NewFormItem("Timezone Override (optional)", timezoneEntry),
		widget.NewFormItem("", widget.NewLabel("Leave blank to inherit the global server timezone (or OS local if unset).")),
		widget.NewFormItem("Max Retries", maxRetriesEntry),
		widget.NewFormItem("", enabledCheck),
		widget.NewFormItem("", retryCheck),
	)

	title := "Add Job"
	if existing != nil {
		title = "Edit Job"
	}

	formScroll := container.NewVScroll(form)
	formScroll.SetMinSize(fyne.NewSize(500, 560))

	d := dialog.NewCustomConfirm(title, "Save", "Cancel", formScroll, func(confirm bool) {
		if !confirm {
			return
		}

		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			ui.ShowToast("Job name is required.")
			return
		}

		scheduleRaw := ""
		if currentScheduleMode == ParseModeRaw {
			scheduleRaw = advancedScheduleEntry.Text
		}
		schedule, err := ConsolidateScheduleFromEditor(currentScheduleMode, cronWidget, scheduleRaw)
		if err != nil {
			ui.ShowToast(fmt.Sprintf("Invalid cron expression: %v", err))
			return
		}

		maxRetries := 1
		if strings.TrimSpace(maxRetriesEntry.Text) != "" {
			value, err := strconv.Atoi(strings.TrimSpace(maxRetriesEntry.Text))
			if err != nil || value <= 0 {
				ui.ShowToast("Max retries must be a positive number.")
				return
			}
			maxRetries = value
		}

		switch currentEditorMode {
		case workflowEditorModeBuilder:
			if err := syncBuilderToCanonical(true); err != nil {
				ui.ShowToast(fmt.Sprintf("Invalid workflow steps: %v", err))
				return
			}
		default:
			if err := parseAdvancedJSON(false, true); err != nil {
				ui.ShowToast(fmt.Sprintf("Invalid workflow steps JSON: %v", err))
				return
			}
		}

		updated := job
		updated.Name = name
		updated.Schedule = schedule
		updated.Steps = cloneWorkflowSteps(canonicalSteps)
		updated.Timezone = strings.TrimSpace(timezoneEntry.Text)
		updated.Enabled = enabledCheck.Checked
		updated.RetryOnError = retryCheck.Checked
		updated.MaxRetries = maxRetries

		if err := ui.app.UpsertScheduledJob(&updated); err != nil {
			ui.app.Events.Dispatch(events.Errorf("gui", "Error saving job: %v", err))
			ui.ShowToast(fmt.Sprintf("Error: %v", err))
			return
		}
		ui.ShowToast("Job saved. Restart server if already running.")
		ui.refreshSyncCenterTab()
	}, ui.window)

	d.Resize(fyne.NewSize(560, 700))
	d.Show()
}

func (ui *Gui) createActionPopup(eventAction *action.EventAction, actionIndex int) {
	var event, source string
	var actionConfig action.ActionConfig
	actionEnabled := true

	if eventAction != nil {
		event = eventAction.Event
		source = eventAction.Source
		if actionIndex != -1 {
			actionConfig = eventAction.Run[actionIndex]
			actionEnabled = actionConfig.IsEnabled()
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

	enabledCheck := widget.NewCheck("Enabled", nil)
	enabledCheck.SetChecked(actionEnabled)
	dialogBody.Add(enabledCheck)

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
		newAction.SetEnabled(enabledCheck.Checked)

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
				return
			}
		} else {
			err := ui.app.UpdateEventAction(eventAction.Name, actionIndex, newAction)
			if err != nil {
				ui.app.Events.Dispatch(events.Errorf("gui", "Error updating action: %v", err))
				return
			}
		}

		ui.refreshActionsTab()
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
		updateFilterRowOptions   func()
		rebuildFilterRows        func()
		presetLabelForFilters    func() string
		updateQuickPresetOptions func(tableName string)
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
		updateQuickPresetOptions(currentTableName)

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
			currentTableName = ""
			updateQuickPresetOptions("")
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

	type explorerPresetOption struct {
		Label   string
		Filters []ExplorerFilterClause
	}

	tableQuickPresets := map[string][]explorerPresetOption{
		"AccountsPendingChanges": {
			{Label: "Pending Accounts", Filters: []ExplorerFilterClause{{Column: "Status", Mode: FilterModeEquals, Value: "pending"}}},
			{Label: "Failed Accounts", Filters: []ExplorerFilterClause{{Column: "Status", Mode: FilterModeEquals, Value: "failed"}}},
			{Label: "Completed Accounts", Filters: []ExplorerFilterClause{{Column: "Status", Mode: FilterModeEquals, Value: "completed"}}},
		},
		"AccountCheckinsPendingChanges": {
			{Label: "Pending Check-ins", Filters: []ExplorerFilterClause{{Column: "Status", Mode: FilterModeEquals, Value: "pending"}}},
			{Label: "Failed Check-ins", Filters: []ExplorerFilterClause{{Column: "Status", Mode: FilterModeEquals, Value: "failed"}}},
			{Label: "Completed Check-ins", Filters: []ExplorerFilterClause{{Column: "Status", Mode: FilterModeEquals, Value: "completed"}}},
		},
		"JobLog": {
			{Label: "Pull Runs", Filters: []ExplorerFilterClause{{Column: "Direction", Mode: FilterModeEquals, Value: "pull"}}},
			{Label: "Push Runs", Filters: []ExplorerFilterClause{{Column: "Direction", Mode: FilterModeEquals, Value: "push"}}},
			{Label: "Failed Runs", Filters: []ExplorerFilterClause{{Column: "Status", Mode: FilterModeEquals, Value: "failed"}}},
			{Label: "Action Steps", Filters: []ExplorerFilterClause{{Column: "JobKind", Mode: FilterModeEquals, Value: "action"}}},
		},
	}

	filtersMatchPreset := func(current []ExplorerFilterClause, preset []ExplorerFilterClause) bool {
		if len(current) != len(preset) {
			return false
		}
		matched := make([]bool, len(current))
		for _, target := range preset {
			found := false
			for i, clause := range current {
				if matched[i] {
					continue
				}
				if strings.EqualFold(clause.Column, target.Column) && clause.Mode == target.Mode && strings.EqualFold(clause.Value, target.Value) {
					matched[i] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	applyPresetFilters := func(filters []ExplorerFilterClause) {
		pendingFilters = cloneExplorerFilters(filters)
		rebuildFilterRows()
		updateFilterRowOptions()
	}

	updateQuickPresetOptions = func(tableName string) {
		if quickPresetSelect == nil {
			return
		}

		presets := tableQuickPresets[tableName]
		if len(presets) == 0 {
			quickPresetSelect.Options = []string{}
			quickPresetSelect.ClearSelected()
			quickPresetSelect.Hide()
			return
		}

		options := make([]string, 0, len(presets)+1)
		options = append(options, "No Preset")
		for _, preset := range presets {
			options = append(options, preset.Label)
		}

		quickPresetSelect.Options = options
		quickPresetSelect.Show()
		suppressPresetChange = true
		if label := presetLabelForFilters(); label == "" {
			quickPresetSelect.ClearSelected()
		} else {
			quickPresetSelect.SetSelected(label)
		}
		suppressPresetChange = false
		quickPresetSelect.Refresh()
	}

	presetLabelForFilters = func() string {
		presets := tableQuickPresets[currentTableName]
		for _, preset := range presets {
			if filtersMatchPreset(pendingFilters, preset.Filters) {
				return preset.Label
			}
		}
		return ""
	}

	quickPresetSelect = widget.NewSelect([]string{}, func(label string) {
		if suppressPresetChange {
			return
		}
		if label == "" || label == "No Preset" {
			applyPresetFilters(nil)
			return
		}

		presets := tableQuickPresets[currentTableName]
		for _, preset := range presets {
			if preset.Label == label {
				applyPresetFilters(preset.Filters)
				return
			}
		}
	})
	quickPresetSelect.PlaceHolder = "Quick preset"
	quickPresetSelect.Hide()

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
			fyne.Do(func() {
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
			})
		}()
	})

	// Initial table list load
	go func() {
		tables, err := ui.app.DB.GetTables()
		if err != nil {
			ui.app.Events.Dispatch(events.Errorf("gui", "Error getting tables: %v", err))
			return
		}
		fyne.Do(func() {
			tableSelect.Options = tables
			tableSelect.Refresh()
		})
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
		updateQuickPresetOptions(currentTableName)

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

	combinedRight := container.NewHBox(centerSection, widget.NewLabel("  "), rightSection)

	// Create a comprehensive pagination toolbar
	paginationBar := container.NewBorder(
		nil, nil,
		leftSection,
		combinedRight,
		nil,
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

func formatServerJobDisplayLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "-"
	}
	return strings.ReplaceAll(trimmed, " | ", "\n")
}

func rightPaneJobsItemHeight(line string) float32 {
	lineCount := strings.Count(line, "\n") + 1
	if lineCount < 1 {
		lineCount = 1
	}
	lineHeight := theme.TextSize() + (theme.Padding() * 1.6)
	return (float32(lineCount) * lineHeight) + (theme.Padding() * 1.5)
}

func formatServerActivityLine(activity appserver.RuntimeActivity, location *time.Location) string {
	if location == nil {
		location = time.Local
	}
	timezoneLabel := time.Local.String()
	if location != nil {
		timezoneLabel = location.String()
	}

	heartbeat := activity.LastHeartbeat
	if heartbeat.IsZero() {
		return "Activity unavailable."
	}
	heartbeatText := formatTimestampInLocation(heartbeat, location, "2006-01-02 15:04:05")

	if activity.ActiveJobID != "" {
		actionSummary := "n/a"
		if trimmedAction := strings.TrimSpace(activity.ActiveJobAction); trimmedAction != "" {
			actionSummary = trimmedAction
		}
		return fmt.Sprintf(
			"Active: %s (%s, %s) | Action: %s | Queue depth: %d | Heartbeat: %s | TZ: %s",
			activity.ActiveJobName,
			activity.ActiveJobMode,
			activity.ActiveJobID,
			actionSummary,
			activity.QueueDepth,
			heartbeatText,
			timezoneLabel,
		)
	}

	return fmt.Sprintf(
		"No active job | Queue depth: %d | Heartbeat: %s | TZ: %s",
		activity.QueueDepth,
		heartbeatText,
		timezoneLabel,
	)
}

func formatServerJobLine(job *appserver.SyncJob, location *time.Location) string {
	if location == nil {
		location = time.Local
	}
	if job == nil {
		return "Unknown job"
	}

	processRole := "Parent"
	parentJobID := strings.TrimSpace(job.ParentJobID)
	if parentJobID != "" {
		processRole = "Subprocess"
	}

	jobIdentifier := job.ID
	if trimmedName := strings.TrimSpace(job.Name); trimmedName != "" {
		jobIdentifier = fmt.Sprintf("%s (%s)", job.ID, trimmedName)
	}

	start := formatSyncTimestamp(job.StartedAt, location)
	end := formatSyncTimestamp(job.CompletedAt, location)
	line := fmt.Sprintf(
		"[%s] %s: %s | kind=%s | mode=%s | source=%s | queued=%s | started=%s | completed=%s",
		strings.ToUpper(string(job.Status)),
		processRole,
		jobIdentifier,
		job.Kind,
		job.Mode,
		job.Source,
		formatTimestampInLocation(job.QueuedAt, location, "2006-01-02 15:04:05"),
		start,
		end,
	)
	if parentJobID != "" {
		line = fmt.Sprintf("%s | parent=%s", line, parentJobID)
	}
	if job.StepIndex > 0 {
		stepInfo := fmt.Sprintf("%d", job.StepIndex)
		if job.TotalSteps > 0 {
			stepInfo = fmt.Sprintf("%d/%d", job.StepIndex, job.TotalSteps)
		}
		if trimmedStepID := strings.TrimSpace(job.StepID); trimmedStepID != "" {
			stepInfo = fmt.Sprintf("%s (%s)", stepInfo, trimmedStepID)
		}
		line = fmt.Sprintf("%s | step=%s", line, stepInfo)
	}
	if strings.TrimSpace(job.Error) != "" {
		line = fmt.Sprintf("%s | error=%s", line, job.Error)
	}
	if trimmedAction := strings.TrimSpace(job.CurrentAction); trimmedAction != "" {
		line = fmt.Sprintf("%s | action=%s", line, trimmedAction)
	}
	return line
}

func filterActiveAndQueuedJobs(jobs []*appserver.SyncJob) []*appserver.SyncJob {
	filtered := make([]*appserver.SyncJob, 0, len(jobs))
	for _, job := range jobs {
		if job == nil {
			continue
		}
		if job.Status == appserver.SyncJobRunning || job.Status == appserver.SyncJobQueued {
			filtered = append(filtered, job)
		}
	}
	return filtered
}

func formatSyncTimestamp(value *time.Time, location *time.Location) string {
	if location == nil {
		location = time.Local
	}
	if value == nil || value.IsZero() {
		return "-"
	}
	return formatTimestampInLocation(*value, location, "2006-01-02 15:04:05")
}

func formatTimestampInLocation(value time.Time, location *time.Location, layout string) string {
	if location == nil {
		location = time.Local
	}

	trimmedLayout := strings.TrimSpace(layout)
	if trimmedLayout == "" {
		trimmedLayout = time.RFC3339
	}
	return fmt.Sprintf("%s [%s]", value.In(location).Format(trimmedLayout), location.String())
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
	baseURLEntry := widget.NewEntry()
	apiKey := ""
	baseURL := ""
	apiConnected := false
	if ui.app != nil {
		if ui.app.API != nil {
			apiKey = ui.app.API.APIKey
			baseURL = ui.app.API.BaseURL
			apiConnected = ui.app.API.IsConnected()
		}
		if ui.app.Config != nil {
			if apiKey == "" {
				apiKey = ui.app.Config.API.APIKey
			}
			if baseURL == "" {
				baseURL = ui.app.Config.API.BaseURL
			}
		}
	}
	apiKeyEntry.SetText(apiKey)
	baseURLEntry.SetText(baseURL)

	apiIcon := theme.HelpIcon()
	if apiConnected {
		apiIcon = theme.ConfirmIcon()
	} else if strings.TrimSpace(apiKey) != "" { // If key exists but not connected, show error
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
	dbTypeValue := "sqlite3"
	if ui.app != nil && ui.app.Config != nil {
		if strings.TrimSpace(ui.app.Config.DB.Type) != "" {
			dbTypeValue = ui.app.Config.DB.Type
		}
		dbPathEntry.SetText(ui.app.Config.DB.Path)
		dbHostEntry.SetText(ui.app.Config.DB.Host)
		if ui.app.Config.DB.Port > 0 {
			dbPortEntry.SetText(fmt.Sprintf("%d", ui.app.Config.DB.Port))
		}
		dbUserEntry.SetText(ui.app.Config.DB.Username)
		dbPassEntry.SetText(ui.app.Config.DB.Password)
		dbNameEntry.SetText(ui.app.Config.DB.Database)
	}
	if ui.app != nil && ui.app.DB != nil {
		switch config := ui.app.DB.(type) {
		case *database.SQLiteConfig:
			dbPathEntry.SetText(config.Path)
			dbTypeValue = "sqlite3"
		case *database.PostgreSQLConfig:
			dbHostEntry.SetText(config.Host)
			dbPortEntry.SetText(fmt.Sprintf("%d", config.Port))
			dbUserEntry.SetText(config.Username)
			dbPassEntry.SetText(config.Password)
			dbNameEntry.SetText(config.Database)
			dbTypeValue = "postgres"
		case *database.MSSQLConfig:
			dbHostEntry.SetText(config.Host)
			dbPortEntry.SetText(fmt.Sprintf("%d", config.Port))
			dbUserEntry.SetText(config.Username)
			dbPassEntry.SetText(config.Password)
			dbNameEntry.SetText(config.Database)
			dbTypeValue = "mssql"
		default:
			if t := strings.TrimSpace(ui.app.DB.GetType()); t != "" {
				dbTypeValue = t
			}
		}
	}
	if strings.TrimSpace(dbTypeValue) == "" {
		dbTypeValue = "sqlite3"
	}
	dbTypeSelect.SetSelected(dbTypeValue)

	dbIcon := theme.HelpIcon()
	dbConnected := ui.app != nil && ui.app.DB != nil && ui.app.DB.IsConnected()
	if dbConnected {
		dbIcon = theme.ConfirmIcon()
	} else if strings.TrimSpace(dbTypeValue) != "" {
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

	syncPrefsHint := widget.NewLabel("Controls apply to concurrent sync execution.")
	syncPrefsHint.Wrapping = fyne.TextWrapWord

	syncPreferencesCard := ui.newSectionCard(
		"Sync Preferences",
		"Configure supported sync execution settings.",
		widget.NewForm(
			widget.NewFormItem("Parallel Processing", parallelProcessingCheck),
			widget.NewFormItem("Max Concurrent", maxConcurrentEntry),
		),
		syncPrefsHint,
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
	testCustomCheckinsCheck := widget.NewCheck("Enable custom checkin API", nil)
	testCustomCheckinsCheck.SetChecked(ui.app.Config.CustomCheckins)
	otherCard := ui.newSectionCard("Other Settings", "", container.NewVBox(
		testCustomCheckinsCheck,
	))

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

	webhookCard := ui.newSectionCard(
		"Webhook Routing",
		"Select which webhooks the embedded server should handle.",
		accountWebhookCheck,
		checkinWebhookCheck,
		webhookStatusLabel,
	)

	serverHostEntry := widget.NewEntry()
	serverHostEntry.SetText(ui.app.Config.Server.Host)
	serverPortEntry := widget.NewEntry()
	serverPortEntry.SetText(fmt.Sprintf("%d", ui.app.Config.Server.Port))
	serverTimezoneEntry := widget.NewEntry()
	serverTimezoneEntry.SetPlaceHolder("Optional, e.g. America/New_York")
	serverTimezoneEntry.SetText(ui.app.Config.Server.Timezone)
	tlsCertEntry := widget.NewEntry()
	tlsCertEntry.SetText(ui.app.Config.Server.TLSCert)
	tlsKeyEntry := widget.NewEntry()
	tlsKeyEntry.SetText(ui.app.Config.Server.TLSKey)
	webhookSecretEntry := widget.NewPasswordEntry()
	webhookSecretEntry.SetText(ui.app.Config.Server.WebhookSecret)
	internalTokenEntry := widget.NewEntry()
	internalTokenEntry.SetText(ui.app.Config.Server.InternalAPIToken)
	logRequestsCheck := widget.NewCheck("Log incoming requests", nil)
	logRequestsCheck.SetChecked(ui.app.Config.Server.LogRequests)

	regenerateTokenButton := widget.NewButtonWithIcon("Regenerate", theme.ViewRefreshIcon(), func() {
		generatedToken, err := appserver.GenerateInternalAPIToken()
		if err != nil {
			ui.app.Events.Dispatch(events.Errorf("gui", "Failed to regenerate internal API token: %v", err))
			ui.ShowToast("Error: Failed to regenerate internal API token.")
			return
		}
		internalTokenEntry.SetText(generatedToken)
		ui.ShowToast("Generated a new internal API token. Save server settings to apply it.")
	})

	var serverForm *widget.Form
	tlsCertFormItem := widget.NewFormItem("TLS Cert Path", tlsCertEntry)
	tlsKeyFormItem := widget.NewFormItem("TLS Key Path", tlsKeyEntry)
	webhookSecretFormItem := widget.NewFormItem("Webhook Secret", webhookSecretEntry)
	internalTokenFormItem := widget.NewFormItem("Internal API Token", container.NewBorder(nil, nil, nil, regenerateTokenButton, internalTokenEntry))
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
		widget.NewFormItem("Global Timezone (IANA)", serverTimezoneEntry),
		webhookSecretFormItem,
		internalTokenFormItem,
	)

	tlsEnabledCheck.SetChecked(ui.app.Config.Server.TLSEnabled)

	saveServerButton := NewSecondaryButton("Save Server Settings", theme.DocumentSaveIcon(), func() {
		ui.presenter.HandleSaveServerConfig(
			serverHostEntry.Text,
			serverPortEntry.Text,
			serverTimezoneEntry.Text,
			tlsEnabledCheck.Checked,
			tlsCertEntry.Text,
			tlsKeyEntry.Text,
			webhookSecretEntry.Text,
			internalTokenEntry.Text,
			logRequestsCheck.Checked,
		)
	})

	serverSettingsCard := ui.newSectionCard(
		"Server Configuration",
		"Configure host, global timezone, TLS, and request logging for the embedded server.",
		tlsEnabledCheck,
		serverForm,
		logRequestsCheck,
		container.NewCenter(saveServerButton),
	)

	configSaveLocationSelect := widget.NewSelect(configSaveLocationOptions(), nil)
	configSaveLocationSelect.PlaceHolder = "Keep current config path"
	if selected := detectConfigSaveLocation(ui.app); selected != "" {
		configSaveLocationSelect.SetSelected(selected)
	}

	configSavePathLabel := widget.NewLabel("")
	configSavePathLabel.Wrapping = fyne.TextWrapWord
	updateConfigSavePathLabel := func(selected string) {
		selected = strings.TrimSpace(selected)
		if selected != "" {
			if path, err := configSaveLocationPath(selected); err == nil {
				configSavePathLabel.SetText(fmt.Sprintf("Will save to: %s", path))
				return
			}
		}
		if current := currentConfigPath(ui.app); current != "" {
			configSavePathLabel.SetText(fmt.Sprintf("Current: %s", current))
			return
		}
		globalPath, err := configSaveLocationPath(configSaveLocationUserID)
		if err != nil {
			configSavePathLabel.SetText("Current: none")
			return
		}
		configSavePathLabel.SetText(fmt.Sprintf("Will save to default: %s", globalPath))
	}
	configSaveLocationSelect.OnChanged = func(selected string) {
		updateConfigSavePathLabel(selected)
	}
	updateConfigSavePathLabel(configSaveLocationSelect.Selected)

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
			configSaveLocationSelect.Selected,
			selectedThemePreference,
			maxConcurrentEntry.Text,
			parallelProcessingCheck.Checked,
			testCustomCheckinsCheck.Checked,
		)
	})

	viewButton := widget.NewButtonWithIcon("View", theme.VisibilityIcon(), ui.presenter.HandleViewConfig)

	actionsCard := ui.newSectionCard(
		"Configuration Actions",
		"Review or persist the current configuration.",
		widget.NewForm(
			widget.NewFormItem("Save Location", configSaveLocationSelect),
		),
		configSavePathLabel,
		container.NewGridWithColumns(2, viewButton, saveButton),
	)

	scrollContent := container.NewVScroll(container.NewVBox(
		NewSpacer(fyne.NewSize(0, 10)),
		apiCard,
		dbCard,
		serverSettingsCard,
		webhookCard,
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
	fyne.Do(func() {
		if ui == nil || ui.window == nil || ui.window.Canvas() == nil {
			return
		}

		if !ui.toastMutex.TryLock() {
			return // Don't show a new toast if one is already visible
		}

		toastContent := container.NewPadded(widget.NewLabel(content))
		popup := widget.NewPopUp(toastContent, ui.window.Canvas())
		popup.Show()
		popup.Resize(popup.MinSize())

		winSize := ui.window.Canvas().Size()
		popupSize := popup.Size()
		popup.Move(fyne.NewPos(
			winSize.Width-popupSize.Width-theme.Padding(),
			winSize.Height-popupSize.Height-theme.Padding(),
		))

		// Hide the popup after a short duration
		go func() {
			time.Sleep(3 * time.Second)
			fyne.Do(func() {
				popup.Hide()
				ui.toastMutex.Unlock()
			})
		}()
	})
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

// refreshConfigTab rebuilds and refreshes the configuration tab on the UI thread.
func (ui *Gui) refreshConfigTab() {
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

// RefreshConfigTab rebuilds and refreshes the configuration tab.
func (ui *Gui) RefreshConfigTab() {
	fyne.Do(func() {
		ui.refreshConfigTab()
	})
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

	dbType := ui.app.DB.GetType()
	whereClause, filterArgs := buildExplorerWhereClause(resolvedFilters, dbType)
	orderClause := buildExplorerOrderClause(columns, orderColumn, normalized.OrderDescending, dbType)

	countQuery := buildExplorerCountQuery(tableName, whereClause, dbType)

	countRows, err := ui.app.DB.GetDB().Query(countQuery, filterArgs...)
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

	rows, err := ui.app.DB.GetDB().Query(selectQuery, filterArgs...)
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
			rowData[i] = ui.formatExplorerCellValue(tableName, resultColumns[i], *v)
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

func (ui *Gui) formatExplorerCellValue(tableName, columnName string, value interface{}) string {
	if value == nil {
		return ""
	}

	// Render JobLog absolute timestamps in the configured server timezone.
	if strings.EqualFold(tableName, "JobLog") &&
		(strings.EqualFold(columnName, "StartedAt") || strings.EqualFold(columnName, "CompletedAt")) {
		if parsed, ok := parseExplorerTimestamp(value); ok {
			return ui.formatTimestampInDisplayTimezone(parsed, "2006-01-02 15:04:05")
		}
	}

	if b, ok := value.([]byte); ok {
		return string(b)
	}
	return fmt.Sprintf("%v", value)
}

func parseExplorerTimestamp(value interface{}) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed, true
	case *time.Time:
		if typed != nil {
			return *typed, true
		}
	case string:
		return parseExplorerTimestampString(typed)
	case []byte:
		return parseExplorerTimestampString(string(typed))
	}
	return time.Time{}, false
}

func parseExplorerTimestampString(raw string) (time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, false
	}

	layoutsWithZone := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05Z07:00",
	}
	for _, layout := range layoutsWithZone {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed, true
		}
	}

	layoutsWithoutZone := []string{
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layoutsWithoutZone {
		if parsed, err := time.ParseInLocation(layout, trimmed, time.UTC); err == nil {
			return parsed, true
		}
	}

	return time.Time{}, false
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

func quoteExplorerIdentifier(identifier, dbType string) string {
	if strings.EqualFold(dbType, "mssql") {
		return "[" + strings.ReplaceAll(identifier, "]", "]]") + "]"
	}
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func likeOperator(dbType string) string {
	if strings.EqualFold(dbType, "postgres") {
		return "ILIKE"
	}
	return "LIKE"
}

func explorerPlaceholder(dbType string, position int) string {
	if strings.EqualFold(dbType, "postgres") {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

func buildFilterCondition(column string, mode ExplorerFilterMode, value string, dbType string, position int) (string, any) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}

	quotedColumn := quoteExplorerIdentifier(column, dbType)
	placeholder := explorerPlaceholder(dbType, position)

	switch mode {
	case FilterModeEquals:
		return fmt.Sprintf("%s = %s", quotedColumn, placeholder), trimmed
	case FilterModeNotEquals:
		return fmt.Sprintf("%s <> %s", quotedColumn, placeholder), trimmed
	case FilterModeStartsWith:
		return fmt.Sprintf("%s %s %s", quotedColumn, likeOperator(dbType), placeholder), trimmed + "%"
	case FilterModeEndsWith:
		return fmt.Sprintf("%s %s %s", quotedColumn, likeOperator(dbType), placeholder), "%" + trimmed
	default:
		return fmt.Sprintf("%s %s %s", quotedColumn, likeOperator(dbType), placeholder), "%" + trimmed + "%"
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

func buildExplorerWhereClause(filters []ExplorerFilterClause, dbType string) (string, []any) {
	if len(filters) == 0 {
		return "", nil
	}

	clauses := make([]string, 0, len(filters))
	args := make([]any, 0, len(filters))
	for _, clause := range filters {
		if clause.Column == "" || clause.Mode == FilterModeNone {
			continue
		}
		condition, arg := buildFilterCondition(clause.Column, clause.Mode, clause.Value, dbType, len(args)+1)
		if condition != "" {
			clauses = append(clauses, condition)
			args = append(args, arg)
		}
	}

	if len(clauses) == 0 {
		return "", nil
	}

	if len(clauses) == 1 {
		return clauses[0], args
	}

	return strings.Join(clauses, " AND "), args
}

func buildExplorerOrderClause(columns []string, orderColumn string, descending bool, dbType string) string {
	direction := "ASC"
	if descending {
		direction = "DESC"
	}

	if orderColumn != "" {
		return fmt.Sprintf("ORDER BY %s %s", quoteExplorerIdentifier(orderColumn, dbType), direction)
	}

	fallback := fallbackOrderColumn(columns)
	if fallback != "" {
		return fmt.Sprintf("ORDER BY %s %s", quoteExplorerIdentifier(fallback, dbType), direction)
	}

	if strings.EqualFold(dbType, "mssql") {
		return fmt.Sprintf("ORDER BY 1 %s", direction)
	}

	return ""
}

func buildExplorerCountQuery(tableName, whereClause, dbType string) string {
	quotedTable := quoteExplorerIdentifier(tableName, dbType)
	if whereClause == "" {
		return fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTable)
	}
	return fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", quotedTable, whereClause)
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
	builder.WriteString(fmt.Sprintf("SELECT * FROM %s", quoteExplorerIdentifier(tableName, dbType)))
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

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteExplorerIdentifier(tableName, ui.app.DB.GetType()))
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
	quotedTable := quoteExplorerIdentifier(tableName, db.GetType())
	switch strings.ToLower(db.GetType()) {
	case "mssql":
		query = fmt.Sprintf("SELECT TOP 1 * FROM %s", quotedTable)
	default:
		query = fmt.Sprintf("SELECT * FROM %s LIMIT 1", quotedTable)
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
