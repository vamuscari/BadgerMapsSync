package gui

import (
	"badgermaps/database"
	"badgermaps/events"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
	"github.com/guregu/null/v6"
	"image/color"
	"strconv"
	"strings"
	"sync/atomic"
)

const (
	syncKindAll      = "All"
	syncKindAccounts = "Accounts"
	syncKindCheckins = "Checkins"
	syncKindRoutes   = "Routes"
	syncKindUser     = "User"

	scopeAll    = "All"
	scopeSingle = "Single"
)

type omniSuggestion struct {
	id      int
	display string
	summary string
}

type SyncCenter struct {
	ui        *Gui
	presenter *GuiPresenter

	syncTypeSelect *widget.Select
	scopeSelect    *widget.Select
	omniEntry      *xwidget.CompletionEntry
	actionButton   *widget.Button
	typeGroup      *fyne.Container
	scopeGroup     *fyne.Container
	omniGroup      *fyne.Container
	actionRow      *fyne.Container
	controlsBody   *fyne.Container
	controlsCard   fyne.CanvasObject

	// Push controls
	pushTypeSelect   *widget.Select
	pushScopeSelect  *widget.Select
	pushActionButton *widget.Button
	pushTypeGroup    *fyne.Container
	pushScopeGroup   *fyne.Container
	pushBody         *fyne.Container
	pushCard         fyne.CanvasObject

	suggestions   map[string]omniSuggestion
	suppressOmni  int32
	currentType   string
	currentScope  string
	currentRecord int
	syncGate      int32
	lastDetail    string

	serverStatusLabel *canvas.Text
}

type serverControlVisualState struct {
	StatusLabel     string
	ActionLabel     string
	StatusColorName fyne.ThemeColorName
	ActionColorName fyne.ThemeColorName
}

func resolveServerControlVisualState(running bool) serverControlVisualState {
	if running {
		return serverControlVisualState{
			StatusLabel:     "Running",
			ActionLabel:     "Stop",
			StatusColorName: StatusPositiveColorName,
			ActionColorName: StatusNegativeColorName,
		}
	}
	return serverControlVisualState{
		StatusLabel:     "Stopped",
		ActionLabel:     "Start",
		StatusColorName: StatusNegativeColorName,
		ActionColorName: StatusPositiveColorName,
	}
}

func neutralServerControlVisualState() serverControlVisualState {
	return serverControlVisualState{
		StatusLabel:     "Unknown",
		ActionLabel:     "Checking...",
		StatusColorName: StatusCardBackgroundColorName,
		ActionColorName: StatusCardBackgroundColorName,
	}
}

type splitPillServerControl struct {
	widget.BaseWidget

	statusText        string
	actionText        string
	statusTextColor   color.Color
	actionTextColor   color.Color
	statusBackground  color.Color
	actionBackground  color.Color
	borderColor       color.Color
	actionEnabled     bool
	onActionRequested func()
}

const (
	splitPillInset          float32 = 1
	splitPillDividerWidth   float32 = 1
	splitPillMinStatusWidth float32 = 92
	splitPillMinActionWidth float32 = 80
)

func newSplitPillServerControl() *splitPillServerControl {
	ctrl := &splitPillServerControl{
		statusText:       "Unknown",
		actionText:       "Checking...",
		statusTextColor:  theme.ForegroundColor(),
		actionTextColor:  theme.ForegroundColor(),
		statusBackground: theme.ButtonColor(),
		actionBackground: theme.ButtonColor(),
		borderColor:      theme.SeparatorColor(),
		actionEnabled:    false,
	}
	ctrl.ExtendBaseWidget(ctrl)
	return ctrl
}

func (c *splitPillServerControl) SetState(
	statusText string,
	actionText string,
	statusBackground color.Color,
	actionBackground color.Color,
	statusTextColor color.Color,
	actionTextColor color.Color,
	borderColor color.Color,
	actionEnabled bool,
) {
	c.statusText = statusText
	c.actionText = actionText
	c.statusBackground = statusBackground
	c.actionBackground = actionBackground
	c.statusTextColor = statusTextColor
	c.actionTextColor = actionTextColor
	c.borderColor = borderColor
	c.actionEnabled = actionEnabled
	c.Refresh()
}

func (c *splitPillServerControl) SetOnActionRequested(handler func()) {
	c.onActionRequested = handler
}

func (c *splitPillServerControl) CreateRenderer() fyne.WidgetRenderer {
	border := canvas.NewRectangle(color.NRGBA{A: 0})
	border.StrokeWidth = 1
	border.CornerRadius = theme.InputRadiusSize()
	statusBackground := canvas.NewRectangle(theme.ButtonColor())
	actionBackground := canvas.NewRectangle(theme.ButtonColor())
	statusCap := canvas.NewCircle(theme.ButtonColor())
	actionCap := canvas.NewCircle(theme.ButtonColor())
	divider := canvas.NewRectangle(theme.SeparatorColor())
	statusLabel := canvas.NewText("", theme.ForegroundColor())
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	actionLabel := canvas.NewText("", theme.ForegroundColor())
	actionLabel.TextStyle = fyne.TextStyle{Bold: true}

	r := &splitPillServerControlRenderer{
		control:          c,
		border:           border,
		statusBackground: statusBackground,
		actionBackground: actionBackground,
		statusCap:        statusCap,
		actionCap:        actionCap,
		divider:          divider,
		statusLabel:      statusLabel,
		actionLabel:      actionLabel,
	}
	r.objects = []fyne.CanvasObject{border, statusBackground, actionBackground, statusCap, actionCap, divider, statusLabel, actionLabel}
	r.Refresh()
	return r
}

func (c *splitPillServerControl) Tapped(pe *fyne.PointEvent) {
	if !c.actionEnabled || c.onActionRequested == nil {
		return
	}
	size := c.Size()
	if size.Width <= 0 {
		return
	}
	actionStart := splitPillActionStartX(size.Width)
	if pe.Position.X < actionStart {
		return
	}
	c.onActionRequested()
}

func (c *splitPillServerControl) TappedSecondary(*fyne.PointEvent) {}

type splitPillServerControlRenderer struct {
	control          *splitPillServerControl
	border           *canvas.Rectangle
	statusBackground *canvas.Rectangle
	actionBackground *canvas.Rectangle
	statusCap        *canvas.Circle
	actionCap        *canvas.Circle
	divider          *canvas.Rectangle
	statusLabel      *canvas.Text
	actionLabel      *canvas.Text
	objects          []fyne.CanvasObject
}

func (r *splitPillServerControlRenderer) Layout(size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	r.border.Resize(size)
	r.border.Move(fyne.NewPos(0, 0))

	innerWidth := size.Width - (splitPillInset * 2)
	innerHeight := size.Height - (splitPillInset * 2)
	if innerWidth <= 0 || innerHeight <= 0 {
		return
	}

	splitX := splitPillServerControlSplit(innerWidth)
	statusWidth := splitX
	actionWidth := innerWidth - splitX - splitPillDividerWidth
	if statusWidth < 1 {
		statusWidth = 1
	}
	if actionWidth < 1 {
		actionWidth = 1
	}

	// Keep the rounded corners only on the outer edges.
	capDiameter := innerHeight
	statusRectStart := splitPillInset + capDiameter/2
	statusRectWidth := statusWidth - capDiameter/2
	if statusRectWidth < 0 {
		statusRectWidth = 0
	}
	actionRectStart := splitPillInset + splitX + splitPillDividerWidth
	actionRectWidth := actionWidth - capDiameter/2
	if actionRectWidth < 0 {
		actionRectWidth = 0
	}

	r.statusCap.Move(fyne.NewPos(splitPillInset, splitPillInset))
	r.statusCap.Resize(fyne.NewSize(capDiameter, capDiameter))
	r.actionCap.Move(fyne.NewPos(splitPillInset+innerWidth-capDiameter, splitPillInset))
	r.actionCap.Resize(fyne.NewSize(capDiameter, capDiameter))

	r.statusBackground.Move(fyne.NewPos(statusRectStart, splitPillInset))
	r.statusBackground.Resize(fyne.NewSize(statusRectWidth, innerHeight))
	r.divider.Move(fyne.NewPos(splitPillInset+splitX, splitPillInset))
	r.divider.Resize(fyne.NewSize(splitPillDividerWidth, innerHeight))
	r.actionBackground.Move(fyne.NewPos(actionRectStart, splitPillInset))
	r.actionBackground.Resize(fyne.NewSize(actionRectWidth, innerHeight))

	statusSize := r.statusLabel.MinSize()
	r.statusLabel.Resize(statusSize)
	r.statusLabel.Move(fyne.NewPos(
		splitPillInset+(statusWidth-statusSize.Width)/2,
		splitPillInset+(innerHeight-statusSize.Height)/2,
	))
	actionSize := r.actionLabel.MinSize()
	r.actionLabel.Resize(actionSize)
	r.actionLabel.Move(fyne.NewPos(
		splitPillInset+splitX+splitPillDividerWidth+(actionWidth-actionSize.Width)/2,
		splitPillInset+(innerHeight-actionSize.Height)/2,
	))
}

func (r *splitPillServerControlRenderer) MinSize() fyne.Size {
	statusSize := r.statusLabel.MinSize()
	actionSize := r.actionLabel.MinSize()
	width := statusSize.Width + actionSize.Width + theme.Padding()*6 + 3
	minInteractiveWidth := (splitPillInset * 2) + splitPillDividerWidth + splitPillMinStatusWidth + splitPillMinActionWidth
	if width < minInteractiveWidth {
		width = minInteractiveWidth
	}
	height := statusSize.Height
	if actionSize.Height > height {
		height = actionSize.Height
	}
	height += theme.Padding() * 2
	return fyne.NewSize(width, height)
}

func (r *splitPillServerControlRenderer) Refresh() {
	r.statusLabel.Text = r.control.statusText
	r.actionLabel.Text = r.control.actionText
	r.statusLabel.Color = r.control.statusTextColor
	r.actionLabel.Color = r.control.actionTextColor
	r.statusBackground.FillColor = r.control.statusBackground
	r.actionBackground.FillColor = r.control.actionBackground
	r.statusCap.FillColor = r.control.statusBackground
	r.actionCap.FillColor = r.control.actionBackground
	r.divider.FillColor = r.control.borderColor
	r.border.StrokeColor = r.control.borderColor

	if !r.control.actionEnabled {
		r.actionBackground.FillColor = theme.DisabledButtonColor()
		r.actionCap.FillColor = theme.DisabledButtonColor()
		r.actionLabel.Color = theme.DisabledColor()
	}

	r.Layout(r.control.Size())
	canvas.Refresh(r.statusLabel)
	canvas.Refresh(r.actionLabel)
	r.statusCap.Refresh()
	r.actionCap.Refresh()
	r.statusBackground.Refresh()
	r.actionBackground.Refresh()
	r.divider.Refresh()
	r.border.Refresh()
}

func (r *splitPillServerControlRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *splitPillServerControlRenderer) Destroy() {}

func splitPillServerControlSplit(width float32) float32 {
	if width <= 0 {
		return 0
	}
	split := width * 0.6
	requiredMinWidth := splitPillMinStatusWidth + splitPillMinActionWidth + splitPillDividerWidth
	if width >= requiredMinWidth {
		if width-split < splitPillMinActionWidth {
			split = width - splitPillMinActionWidth
		}
		if split < splitPillMinStatusWidth {
			split = splitPillMinStatusWidth
		}
	} else {
		// In very tight widths, keep a visible action area by preserving a ratio.
		split = width * 0.58
	}
	if split > width-2 {
		split = width - 2
	}
	if split < 1 {
		split = 1
	}
	return split
}

func splitPillActionStartX(totalWidth float32) float32 {
	if totalWidth <= 0 {
		return 0
	}
	innerWidth := totalWidth - (splitPillInset * 2)
	if innerWidth <= 0 {
		return totalWidth
	}
	return splitPillInset + splitPillServerControlSplit(innerWidth) + splitPillDividerWidth
}

func serverControlThemeColor(ui *Gui, name fyne.ThemeColorName) color.Color {
	switch name {
	case StatusPositiveColorName, StatusNegativeColorName, StatusCardBackgroundColorName, StatusCardBorderColorName:
		variant := theme.VariantDark
		if ui != nil && ui.fyneApp != nil {
			variant = ui.fyneApp.Settings().ThemeVariant()
		}
		return newModernThemeForVariant(variant).Color(name, variant)
	}

	if ui != nil {
		if c := ui.themeColor(name); c != nil {
			return c
		}
	}

	return theme.ForegroundColor()
}

func NewSyncCenter(ui *Gui, presenter *GuiPresenter) *SyncCenter {
	sc := &SyncCenter{
		ui:           ui,
		presenter:    presenter,
		suggestions:  make(map[string]omniSuggestion),
		currentType:  syncKindAll,
		currentScope: scopeAll,
		lastDetail:   "Choose a pull type to get started.",
	}

	sc.syncTypeSelect = widget.NewSelect([]string{
		syncKindAll,
		syncKindAccounts,
		syncKindCheckins,
		syncKindRoutes,
		syncKindUser,
	}, sc.onSyncTypeChanged)

	sc.scopeSelect = widget.NewSelect([]string{scopeAll, scopeSingle}, sc.onScopeChanged)

	sc.omniEntry = xwidget.NewCompletionEntry(nil)
	sc.omniEntry.SetPlaceHolder("Search by name or enter an ID")
	sc.omniEntry.OnChanged = sc.onOmniChanged
	sc.omniEntry.OnSubmitted = sc.onOmniSubmit

	sc.actionButton = widget.NewButtonWithIcon("Pull All", theme.DownloadIcon(), func() {
		sc.handleAction()
	})
	sc.actionButton.Importance = widget.HighImportance

	sc.typeGroup = container.NewVBox(
		widget.NewLabel("Pull Type"),
		sc.syncTypeSelect,
	)

	sc.scopeGroup = container.NewVBox(
		widget.NewLabel("Scope"),
		sc.scopeSelect,
	)

	sc.omniGroup = container.NewVBox(
		widget.NewLabel("Record"),
		sc.omniEntry,
		widget.NewLabel("Select from search results or enter the ID directly."),
	)

	sc.actionRow = container.NewHBox(sc.actionButton, layout.NewSpacer())

	sc.controlsBody = container.NewVBox()

	// --- Push controls ---
	sc.pushTypeSelect = widget.NewSelect([]string{"All", "Accounts", "Checkins"}, func(string) { sc.updatePushControls() })
	sc.pushScopeSelect = widget.NewSelect([]string{"All", "Single"}, func(string) { sc.updatePushControls() })
	sc.pushActionButton = widget.NewButtonWithIcon("Push All Changes", theme.UploadIcon(), func() {
		sc.handlePushAction()
	})
	sc.pushActionButton.Importance = widget.HighImportance
	sc.pushTypeGroup = container.NewVBox(widget.NewLabel("Push Type"), sc.pushTypeSelect)
	sc.pushScopeGroup = container.NewVBox(widget.NewLabel("Scope"), sc.pushScopeSelect)
	sc.pushBody = container.NewVBox()

	sc.scopeGroup.Hide()
	sc.omniGroup.Hide()

	sc.syncTypeSelect.SetSelected(syncKindAll)
	sc.scopeSelect.SetSelected(scopeAll)
	sc.pushTypeSelect.SetSelected("All")
	sc.pushScopeSelect.SetSelected("All")
	sc.updateControls()
	sc.updatePushControls()

	return sc
}

func (sc *SyncCenter) RefreshServerStatusLabel() {
	if sc == nil || sc.serverStatusLabel == nil {
		return
	}

	statusText := "Server: Unknown"
	statusColor := theme.ForegroundColor()
	if sc.ui != nil && sc.ui.app != nil && sc.ui.app.Server != nil {
		pid, running := sc.ui.app.Server.GetServerStatus()
		if running {
			statusText = fmt.Sprintf("Server: Running (PID %d)", pid)
			statusColor = sc.ui.themeColor(StatusPositiveColorName)
		} else {
			statusText = "Server: Stopped"
			statusColor = sc.ui.themeColor(StatusNegativeColorName)
		}
	}

	sc.serverStatusLabel.Text = statusText
	sc.serverStatusLabel.Color = statusColor
	canvas.Refresh(sc.serverStatusLabel)
}

// PushCard exposes the push card for external rendering (e.g., screenshots)
func (sc *SyncCenter) PushCard() fyne.CanvasObject { return sc.pushCard }

func (sc *SyncCenter) CreateContent() fyne.CanvasObject {
	sc.controlsBody = container.NewVBox()

	sc.controlsCard = sc.ui.newSectionCard(
		"Pull",
		"Pull from BadgerMaps",
		container.NewVBox(
			sc.controlsBody,
		),
	)

	// Build push card body
	sc.pushCard = sc.ui.newSectionCard(
		"Push",
		"Push pending changes to BadgerMaps",
		container.NewVBox(
			sc.pushBody,
		),
	)
	sc.rebuildControlsLayout()
	sc.rebuildPushLayout()
	sc.applyStoredDetail()

	sc.serverStatusLabel = canvas.NewText("Server: Unknown", theme.ForegroundColor())
	sc.serverStatusLabel.TextStyle = fyne.TextStyle{Bold: true}
	sc.serverStatusLabel.TextSize = theme.TextSize()
	sc.RefreshServerStatusLabel()

	syncHistoryButton := widget.NewButtonWithIcon("Sync History", theme.NavigateNextIcon(), func() {
		if !sc.ui.OpenExplorerTable("SyncHistory") {
			sc.ui.app.Events.Dispatch(events.Debugf("sync_center", "unable to navigate to sync history"))
		}
	})
	pendingChangesButton := widget.NewButtonWithIcon("Pending Changes", theme.NavigateNextIcon(), func() {
		if !sc.ui.OpenExplorerPendingChanges() {
			sc.ui.app.Events.Dispatch(events.Debugf("sync_center", "unable to navigate to pending changes"))
		}
	})
	toolsTabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Pull", theme.DownloadIcon(), sc.controlsCard),
		container.NewTabItemWithIcon("Push", theme.UploadIcon(), sc.pushCard),
		container.NewTabItemWithIcon("Navigation", theme.NavigateNextIcon(), container.NewVBox(
			syncHistoryButton,
			pendingChangesButton,
		)),
	)
	toolsCard := sc.ui.newSectionCard(
		"Additional Tools",
		"Manual sync actions and explorer shortcuts.",
		toolsTabs,
	)

	title := canvas.NewText("Sync Center", theme.ForegroundColor())
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = theme.TextSize() + 4

	headerControls := container.NewVBox(
		NewSpacer(fyne.NewSize(0, theme.Padding()*0.5)),
		container.NewHBox(
			sc.serverStatusLabel,
			NewSpacer(fyne.NewSize(theme.Padding()*2, 0)),
		),
	)
	header := container.NewBorder(nil, nil, nil, headerControls, container.NewHBox(title))

	content := container.NewVScroll(container.NewVBox(
		sc.ui.createScheduledJobsSection(),
		toolsCard,
	))

	return container.NewBorder(header, nil, nil, nil, content)
}

func (sc *SyncCenter) onSyncTypeChanged(value string) {
	if value == "" {
		return
	}
	sc.currentType = value
	sc.clearSingleSelection()
	if value == syncKindAccounts || value == syncKindCheckins || value == syncKindRoutes {
		sc.scopeSelect.SetSelected(scopeAll)
	} else {
		sc.scopeSelect.SetSelected(scopeAll)
	}
	sc.updateControls()
}

func (sc *SyncCenter) onScopeChanged(value string) {
	if value == "" {
		return
	}
	sc.currentScope = value
	if value == scopeSingle {
		sc.clearSingleSelection()
	}
	sc.updateControls()
}

func (sc *SyncCenter) updateControls() {
	scoped := sc.currentType == syncKindAccounts || sc.currentType == syncKindCheckins || sc.currentType == syncKindRoutes

	if scoped {
		sc.scopeGroup.Show()
	} else {
		sc.scopeGroup.Hide()
		sc.currentScope = scopeAll
	}

	showOmni := scoped && sc.currentScope == scopeSingle
	if showOmni {
		sc.omniGroup.Show()
		sc.omniEntry.Enable()
		sc.setDetail("Enter an ID to view record details.")
	} else {
		sc.omniGroup.Hide()
		sc.omniEntry.Disable()
		sc.omniEntry.SetOptions(nil)
		sc.omniEntry.HideCompletion()
		sc.setDetail(sc.detailMessageForType(sc.currentType))
	}

	sc.updateActionLabel()
	sc.updateActionState()

	sc.rebuildControlsLayout()
	if sc.canShowDetails() && sc.lastDetail != "" {
		sc.ui.setDetails(NewWrappingLabel(sc.lastDetail), sc.ui.rightPaneVisible)
	}
}

func (sc *SyncCenter) updateActionLabel() {
	if sc.actionButton == nil {
		return
	}

	text := "Run"
	switch sc.currentType {
	case syncKindAll:
		text = "Pull All"
	case syncKindUser:
		text = "Pull User Profile"
	case syncKindAccounts:
		if sc.currentScope == scopeSingle {
			text = "Pull Account"
		} else {
			text = "Pull All Accounts"
		}
	case syncKindCheckins:
		if sc.currentScope == scopeSingle {
			text = "Pull Check-in"
		} else {
			text = "Pull All Check-ins"
		}
	case syncKindRoutes:
		if sc.currentScope == scopeSingle {
			text = "Pull Route"
		} else {
			text = "Pull All Routes"
		}
	}

	sc.actionButton.SetText(text)
}

func (sc *SyncCenter) updateActionState() {
	if sc.actionButton == nil {
		return
	}

	if sc.currentScope == scopeSingle {
		if sc.currentRecord <= 0 {
			sc.actionButton.Disable()
			return
		}
	}
	sc.actionButton.Enable()
}

func (sc *SyncCenter) rebuildControlsLayout() {
	if sc.controlsBody == nil {
		return
	}

	rows := []fyne.CanvasObject{sc.typeGroup}

	if sc.scopeGroup.Visible() {
		rows = append(rows, sc.scopeGroup)
	}

	if sc.omniGroup.Visible() {
		rows = append(rows, sc.omniGroup)
	}

	rows = append(rows, sc.actionRow)

	sc.controlsBody.Objects = rows
	sc.controlsBody.Refresh()
	if sc.controlsCard != nil {
		sc.controlsCard.Refresh()
	}
}

// --- Push helpers ---
func (sc *SyncCenter) rebuildPushLayout() {
	if sc.pushBody == nil {
		return
	}
	rows := []fyne.CanvasObject{sc.pushTypeGroup}
	if sc.pushScopeGroup != nil {
		rows = append(rows, sc.pushScopeGroup)
	}
	rows = append(rows, container.NewHBox(sc.pushActionButton, layout.NewSpacer()))
	sc.pushBody.Objects = rows
	sc.pushBody.Refresh()
	if sc.pushCard != nil {
		// if card is a container, refresh via parent owning code; using fyne CanvasObject interface
	}
}

func (sc *SyncCenter) updatePushControls() {
	// Update button label and enablement based on selects
	if sc.pushActionButton == nil {
		return
	}
	pType := sc.pushTypeSelect.Selected
	pScope := sc.pushScopeSelect.Selected
	// Label
	switch pType {
	case "Accounts":
		if pScope == "Single" {
			sc.pushActionButton.SetText("Push Account (single)")
		} else {
			sc.pushActionButton.SetText("Push Account Changes")
		}
	case "Checkins":
		if pScope == "Single" {
			sc.pushActionButton.SetText("Push Check-in (single)")
		} else {
			sc.pushActionButton.SetText("Push Check-in Changes")
		}
	default:
		sc.pushActionButton.SetText("Push All Changes")
	}
	// Enable only supported combinations (All scope supported). Single not yet implemented
	if pScope == "Single" {
		sc.pushActionButton.Disable()
	} else {
		sc.pushActionButton.Enable()
	}
	sc.rebuildPushLayout()
}

func (sc *SyncCenter) handlePushAction() {
	if !sc.ensureConnections() {
		return
	}
	pType := sc.pushTypeSelect.Selected
	pScope := sc.pushScopeSelect.Selected
	if pScope == "Single" {
		sc.ui.ShowToast("Single-item push not yet supported.")
		return
	}
	switch pType {
	case "Accounts":
		sc.presenter.HandlePushAccounts()
	case "Checkins":
		sc.presenter.HandlePushCheckins()
	default:
		sc.presenter.HandlePushAll()
	}
}

func (sc *SyncCenter) clearSingleSelection() {
	if sc.omniEntry == nil {
		return
	}
	sc.currentRecord = 0
	sc.suggestions = make(map[string]omniSuggestion)
	sc.omniEntry.SetOptions(nil)
	sc.omniEntry.HideCompletion()
	sc.setOmniText("")
	sc.updateActionState()
}

func (sc *SyncCenter) onOmniChanged(value string) {
	if sc.currentScope != scopeSingle {
		return
	}

	if atomic.CompareAndSwapInt32(&sc.suppressOmni, 1, 0) {
		return
	}

	trimmed := strings.TrimSpace(value)
	sc.currentRecord = 0
	sc.updateActionState()

	if trimmed == "" {
		sc.omniEntry.SetOptions(nil)
		sc.omniEntry.HideCompletion()
		sc.setDetail("Enter an ID to view record details.")
		return
	}

	if suggestion, ok := sc.suggestions[trimmed]; ok {
		sc.applySuggestion(suggestion)
		return
	}

	if id, err := strconv.Atoi(trimmed); err == nil {
		sc.currentRecord = id
		sc.updateActionState()
		sc.omniEntry.SetOptions(nil)
		sc.omniEntry.HideCompletion()
		sc.loadAccountDetails(id)
		return
	}

	if len(trimmed) < 2 {
		sc.omniEntry.SetOptions(nil)
		sc.omniEntry.HideCompletion()
		sc.setDetail("Type at least two characters to search.")
		return
	}

	sc.loadSuggestions(trimmed)
}

func (sc *SyncCenter) onOmniSubmit(value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		sc.omniEntry.SetOptions(nil)
		sc.omniEntry.HideCompletion()
		sc.setDetail("Enter an ID to view record details.")
		return
	}

	if suggestion, ok := sc.suggestions[trimmed]; ok {
		sc.applySuggestion(suggestion)
		return
	}

	if id, err := strconv.Atoi(trimmed); err == nil {
		sc.currentRecord = id
		sc.updateActionState()
		sc.omniEntry.SetOptions(nil)
		sc.omniEntry.HideCompletion()
		sc.loadDetailsForID(id)
		return
	}

	sc.loadSuggestions(trimmed)
}

func (sc *SyncCenter) applySuggestion(s omniSuggestion) {
	sc.currentRecord = s.id
	sc.updateActionState()
	sc.setOmniText(strconv.Itoa(s.id))
	sc.setDetail(s.summary)
	sc.omniEntry.SetOptions(nil)
	sc.omniEntry.HideCompletion()
}

func (sc *SyncCenter) setOmniText(text string) {
	if sc.omniEntry == nil {
		return
	}
	atomic.StoreInt32(&sc.suppressOmni, 1)
	sc.omniEntry.SetText(text)
}

func (sc *SyncCenter) loadSuggestions(query string) {
	if !sc.databaseReady() {
		sc.suggestions = make(map[string]omniSuggestion)
		sc.omniEntry.SetOptions(nil)
		sc.omniEntry.HideCompletion()
		sc.setDetail("Connect to the database to search records.")
		return
	}

	switch sc.currentType {
	case syncKindAccounts:
		sc.loadAccountSuggestions(query)
	case syncKindCheckins:
		sc.loadCheckinSuggestions(query)
	case syncKindRoutes:
		sc.loadRouteSuggestions(query)
	}
}

func (sc *SyncCenter) loadAccountSuggestions(query string) {
	rows, err := database.SearchAccounts(sc.ui.app.DB, query)
	if err != nil {
		sc.ui.app.Events.Dispatch(events.Errorf("sync_center", "account search failed: %v", err))
		sc.suggestions = make(map[string]omniSuggestion)
		sc.omniEntry.SetOptions(nil)
		sc.omniEntry.HideCompletion()
		sc.setDetail("Unable to search accounts right now.")
		return
	}

	items := make([]string, 0, len(rows))
	lookup := make(map[string]omniSuggestion)

	for _, row := range rows {
		if !row.AccountId.Valid {
			continue
		}
		id := int(row.AccountId.Int64)
		name := cleanString(row.FullName.ValueOrZero())
		label := fmt.Sprintf("%s (#%d)", fallback(name, "Account"), id)

		summary := fmt.Sprintf("Account #%d\nName: %s", id, fallback(name, "-"))
		lookup[label] = omniSuggestion{id: id, display: label, summary: summary}
		items = append(items, label)
	}

	sc.suggestions = lookup
	sc.omniEntry.SetOptions(items)
	sc.omniEntry.ShowCompletion()
	if len(items) == 0 {
		sc.setDetail("No matching accounts found.")
	}
}

func (sc *SyncCenter) loadCheckinSuggestions(query string) {
	rows, err := database.SearchAccounts(sc.ui.app.DB, query)
	if err != nil {
		sc.ui.app.Events.Dispatch(events.Errorf("sync_center", "account search failed for check-ins: %v", err))
		sc.suggestions = make(map[string]omniSuggestion)
		sc.omniEntry.SetOptions(nil)
		sc.omniEntry.HideCompletion()
		sc.setDetail("Unable to search check-ins right now.")
		return
	}

	items := make([]string, 0, len(rows))
	lookup := make(map[string]omniSuggestion)

	for _, row := range rows {
		if !row.AccountId.Valid {
			continue
		}
		id := int(row.AccountId.Int64)
		name := cleanString(row.FullName.ValueOrZero())
		label := fmt.Sprintf("%s (#%d)", fallback(name, "Account"), id)

		summary := fmt.Sprintf("Account #%d\nName: %s\nPulls all check-ins for this account.", id, fallback(name, "-"))

		lookup[label] = omniSuggestion{id: id, display: label, summary: summary}
		items = append(items, label)
	}

	sc.suggestions = lookup
	sc.omniEntry.SetOptions(items)
	sc.omniEntry.ShowCompletion()
	if len(items) == 0 {
		sc.setDetail("No matching accounts found.")
	}
}

func (sc *SyncCenter) loadRouteSuggestions(query string) {
	rows, err := database.SearchRoutes(sc.ui.app.DB, query)
	if err != nil {
		sc.ui.app.Events.Dispatch(events.Errorf("sync_center", "route search failed: %v", err))
		sc.suggestions = make(map[string]omniSuggestion)
		sc.omniEntry.SetOptions(nil)
		sc.omniEntry.HideCompletion()
		sc.setDetail("Unable to search routes right now.")
		return
	}

	items := make([]string, 0, len(rows))
	lookup := make(map[string]omniSuggestion)

	for _, row := range rows {
		if !row.RouteId.Valid {
			continue
		}
		id := int(row.RouteId.Int64)
		name := cleanString(row.Name.ValueOrZero())
		label := fmt.Sprintf("%s (#%d)", fallback(name, "Route"), id)
		date := cleanString(row.RouteDate.ValueOrZero())
		if date != "" {
			label = fmt.Sprintf("%s (#%d) @ %s", fallback(name, "Route"), id, date)
		}

		summary := fmt.Sprintf("Route #%d\nName: %s\nDate: %s", id, fallback(name, "-"), fallback(date, "-"))

		lookup[label] = omniSuggestion{id: id, display: label, summary: summary}
		items = append(items, label)
	}

	sc.suggestions = lookup
	sc.omniEntry.SetOptions(items)
	sc.omniEntry.ShowCompletion()
	if len(items) == 0 {
		sc.setDetail("No matching routes found.")
	}
}

func (sc *SyncCenter) loadDetailsForID(id int) {
	if !sc.databaseReady() {
		sc.setDetail("Connect to the database to view record details.")
		return
	}

	switch sc.currentType {
	case syncKindAccounts:
		sc.loadAccountDetails(id)
	case syncKindCheckins:
		sc.loadCheckinDetails(id)
	case syncKindRoutes:
		sc.loadRouteDetails(id)
	}
}

func (sc *SyncCenter) loadAccountDetails(id int) {
	account, err := database.GetAccountByID(sc.ui.app.DB, id)
	if err != nil {
		sc.setDetail(fmt.Sprintf("Account #%d not found.", id))
		return
	}

	name := fallback(account.FullName.ValueOrZero(), "-")
	owner := valueFromPointer(account.AccountOwner)
	email := fallback(account.Email.ValueOrZero(), "-")
	last := valueFromPointer(account.LastCheckinDate)

	summary := fmt.Sprintf("Account #%d\nName: %s\nOwner: %s\nEmail: %s\nLast Check-in: %s", id, name, fallback(owner, "-"), email, fallback(last, "-"))
	sc.setDetail(summary)
}

func (sc *SyncCenter) loadCheckinDetails(id int) {
	accountID := id
	account, err := database.GetAccountByID(sc.ui.app.DB, accountID)
	if err != nil {
		sc.setDetail(fmt.Sprintf("Account #%d not found.", accountID))
		return
	}

	name := fallback(account.FullName.ValueOrZero(), "-")
	owner := valueFromPointer(account.AccountOwner)
	last := valueFromPointer(account.LastCheckinDate)

	summary := fmt.Sprintf("Account #%d\nName: %s\nOwner: %s\nLast Check-in: %s", accountID, name, fallback(owner, "-"), fallback(last, "-"))
	sc.setDetail(summary)
}

func (sc *SyncCenter) loadRouteDetails(id int) {
	route, err := database.GetRouteByID(sc.ui.app.DB, id)
	if err != nil {
		sc.setDetail(fmt.Sprintf("Route #%d not found.", id))
		return
	}

	name := fallback(route.Name.ValueOrZero(), "-")
	date := fallback(route.RouteDate.ValueOrZero(), "-")
	start := fallback(route.StartAddress.ValueOrZero(), "-")
	dest := fallback(route.DestinationAddress.ValueOrZero(), "-")

	summary := fmt.Sprintf("Route #%d\nName: %s\nDate: %s\nStart: %s\nDestination: %s", id, name, date, start, dest)
	sc.setDetail(summary)
}

func (sc *SyncCenter) handleAction() {
	switch sc.currentType {
	case syncKindAll:
		sc.runFullSync()
	case syncKindUser:
		if sc.ensureConnections() {
			sc.presenter.HandlePullProfile()
		}
	case syncKindAccounts:
		sc.runAccountOperation()
	case syncKindCheckins:
		sc.runCheckinOperation()
	case syncKindRoutes:
		sc.runRouteOperation()
	}
}

func (sc *SyncCenter) runFullSync() {
	if !sc.ensureConnections() {
		return
	}

	if !atomic.CompareAndSwapInt32(&sc.syncGate, 0, 1) {
		sc.ui.ShowToast("Full sync is already running.")
		return
	}

	sc.ui.app.Events.Dispatch(events.Infof("sync_center", "starting full sync (pull + push)"))
	sc.ui.ShowToast("Starting full sync...")

	go func() {
		defer atomic.StoreInt32(&sc.syncGate, 0)
		if err := sc.presenter.RunFullSyncBlocking(); err != nil {
			sc.ui.app.Events.Dispatch(events.Errorf("sync_center", "full sync failed: %v", err))
			sc.ui.ShowToast("Error: Full sync failed.")
			return
		}
		sc.ui.ShowToast("Success: Full sync complete.")
	}()
}

func (sc *SyncCenter) runAccountOperation() {
	if !sc.ensureConnections() {
		return
	}
	if sc.currentScope == scopeSingle {
		if sc.currentRecord <= 0 {
			sc.ui.ShowToast("Select an account to pull.")
			return
		}
		sc.presenter.HandlePullAccount(strconv.Itoa(sc.currentRecord))
		return
	}
	sc.presenter.HandlePullAccounts()
}

func (sc *SyncCenter) runCheckinOperation() {
	if !sc.ensureConnections() {
		return
	}
	if sc.currentScope == scopeSingle {
		if sc.currentRecord <= 0 {
			sc.ui.ShowToast("Select an account to pull check-ins for.")
			return
		}
		sc.presenter.HandlePullCheckinsForAccount(sc.currentRecord)
		return
	}
	sc.presenter.HandlePullCheckins()
}

func (sc *SyncCenter) runRouteOperation() {
	if !sc.ensureConnections() {
		return
	}
	if sc.currentScope == scopeSingle {
		if sc.currentRecord <= 0 {
			sc.ui.ShowToast("Select a route to pull.")
			return
		}
		sc.presenter.HandlePullRoute(strconv.Itoa(sc.currentRecord))
		return
	}
	sc.presenter.HandlePullRoutes()
}

func (sc *SyncCenter) detailMessageForType(syncType string) string {
	switch syncType {
	case syncKindAll:
		return "Run a full pull and push of pending changes."
	case syncKindAccounts:
		return "Pull all accounts or target a specific account by ID."
	case syncKindCheckins:
		return "Pull every check-in or focus on a single log entry."
	case syncKindRoutes:
		return "Pull the complete routes list or a specific route."
	case syncKindUser:
		return "Refresh your user profile information."
	default:
		return "Choose a sync type to get started."
	}
}

func (sc *SyncCenter) setDetail(message string) {
	sc.lastDetail = message
	if !sc.canShowDetails() {
		return
	}
	sc.ui.setDetails(NewWrappingLabel(message), sc.ui.rightPaneVisible)
}

func (sc *SyncCenter) canShowDetails() bool {
	if sc.ui == nil || sc.ui.rightPaneContent == nil {
		return false
	}
	return len(sc.ui.rightPaneContent.Objects) > 0
}

func (sc *SyncCenter) applyStoredDetail() {
	if sc.lastDetail == "" {
		sc.lastDetail = sc.detailMessageForType(sc.currentType)
	}
	if !sc.canShowDetails() {
		return
	}
	sc.ui.setDetails(NewWrappingLabel(sc.lastDetail), sc.ui.rightPaneVisible)
}

func (sc *SyncCenter) ensureConnections() bool {
	if sc.ui == nil || sc.ui.app == nil {
		return false
	}

	if sc.ui.app.API == nil || !sc.ui.app.API.IsConnected() {
		sc.ui.ShowToast("Connect to the BadgerMaps API to run this action.")
		return false
	}

	if sc.ui.app.DB == nil || !sc.ui.app.DB.IsConnected() {
		sc.ui.ShowToast("Connect to the database to run this action.")
		return false
	}

	return true
}

func (sc *SyncCenter) databaseReady() bool {
	return sc.ui != nil && sc.ui.app != nil && sc.ui.app.DB != nil && sc.ui.app.DB.IsConnected()
}

func cleanString(value string) string {
	return strings.TrimSpace(value)
}

func fallback(value, alt string) string {
	if cleanString(value) == "" {
		return alt
	}
	return value
}

func valueFromPointer(ns *null.String) string {
	if ns == nil {
		return ""
	}
	return cleanString(ns.ValueOrZero())
}
