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
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	syncKindAll      = "All"
	syncKindAccounts = "Accounts"
	syncKindCheckins = "Checkins"
	syncKindRoutes   = "Routes"
	syncKindUser     = "User"

	scopeAll    = "All"
	scopeSingle = "Single"

	fullSyncFollowUpDelay = 1500 * time.Millisecond
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

func (sc *SyncCenter) CreateContent() fyne.CanvasObject {
	sc.controlsBody = container.NewVBox()

	// Push card content: includes Pending Changes shortcut
	pendingChangesRow := container.NewHBox(
		widget.NewButtonWithIcon("Pending Changes", theme.NavigateNextIcon(), func() {
			if !sc.ui.OpenExplorerPendingChanges() {
				sc.ui.app.Events.Dispatch(events.Debugf("sync_center", "unable to navigate to pending changes"))
			}
		}),
		layout.NewSpacer(),
	)

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
			widget.NewSeparator(),
			pendingChangesRow,
		),
	)
	sc.rebuildControlsLayout()
	sc.rebuildPushLayout()
	sc.applyStoredDetail()

	syncHistoryButton := widget.NewButtonWithIcon("Sync History", theme.NavigateNextIcon(), func() {
		if !sc.ui.OpenExplorerTable("SyncHistory") {
			sc.ui.app.Events.Dispatch(events.Debugf("sync_center", "unable to navigate to sync history"))
		}
	})

	title := canvas.NewText("Sync Center", theme.ForegroundColor())
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = theme.TextSize() + 4

	header := container.NewBorder(nil, nil, nil, syncHistoryButton, container.NewHBox(title))

	content := container.NewVScroll(container.NewVBox(
		sc.controlsCard,
		sc.pushCard,
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
		text = "Pull Everything"
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
		sc.presenter.HandlePullGroup()
		time.Sleep(fullSyncFollowUpDelay)
		sc.presenter.HandlePushAll()
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
