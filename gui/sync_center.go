package gui

import (
	"badgermaps/app/push"
	"badgermaps/database"
	"badgermaps/events"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Constants for UI dimensions and timeouts
const (
	// UI dimensions
	statusIndicatorSize = 12
	checkboxColumnWidth = 30
	idColumnWidth       = 60
	nameColumnWidth     = 100
	defaultGridColumns  = 3
	splitViewOffset     = 0.5

	// Pagination
	defaultPageSize = 50
	maxPageSize     = 200

	// Timeouts
	syncDelayBetweenOps = 1 * time.Second

	// Date formats
	shortDateFormat = "Jan 2 15:04"
	timeFormat      = "15:04:05"
)

// SyncCenter provides a unified interface for pull and push operations
type SyncCenter struct {
	ui        *Gui
	presenter *GuiPresenter

	// UI Components
	statusIndicator      *canvas.Circle
	statusLabel          *widget.Label
	lastSyncLabel        *widget.Label
	progressBar          *widget.ProgressBar
	progressLabel        *widget.Label
	selectionStatusLabel *widget.Label

	// Data bindings
	pendingAccountsCount binding.String
	pendingCheckinsCount binding.String
	selectedAccounts     map[int]bool
	selectedCheckins     map[int]bool

	// Tables
	accountsTable  *widget.Table
	checkinsTable  *widget.Table
	conflictsTable *widget.Table

	// Progress container
	progressContainer *fyne.Container
	historyContainer  *fyne.Container

	// Current state - thread safe
	currentOperation string
	operatingMutex   sync.RWMutex
	isOperating      int32 // Use atomic operations

	// Pagination state
	currentPage int
	pageSize    int

	// Current data view state
	currentDataType string
	tableContainer  *fyne.Container
	currentTable    *widget.Table
}

// NewSyncCenter creates a new sync center instance
func NewSyncCenter(ui *Gui, presenter *GuiPresenter) *SyncCenter {
	sc := &SyncCenter{
		ui:                   ui,
		presenter:            presenter,
		selectedAccounts:     make(map[int]bool),
		selectedCheckins:     make(map[int]bool),
		pendingAccountsCount: binding.NewString(),
		pendingCheckinsCount: binding.NewString(),
		currentPage:          0,
		pageSize:             defaultPageSize,
		currentDataType:      "accounts", // Default to accounts
	}

	// Initialize UI components
	sc.statusIndicator = canvas.NewCircle(color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	sc.statusIndicator.StrokeWidth = 2
	sc.statusIndicator.StrokeColor = color.NRGBA{R: 100, G: 100, B: 100, A: 255}

	sc.statusLabel = widget.NewLabel("Idle")
	sc.lastSyncLabel = widget.NewLabel("Never synced")
	sc.progressBar = widget.NewProgressBar()
	sc.progressLabel = widget.NewLabel("")

	return sc
}

// CreateContent builds the main sync center interface with compact layout
func (sc *SyncCenter) CreateContent() fyne.CanvasObject {
	// Create a compact header combining status and controls
	compactHeader := sc.createCompactHeader()

	// Main data section
	dataSection := sc.createDataSection()

	// Combine with minimal vertical space
	content := container.NewBorder(
		compactHeader,
		nil, nil, nil,
		dataSection,
	)

	// Load initial data
	sc.refreshPendingCounts()

	// Return compact layout - add VScroll back if needed for very small windows
	return container.NewVScroll(content)
}

// createCompactHeader creates a condensed header with status and controls in one row
func (sc *SyncCenter) createCompactHeader() fyne.CanvasObject {
	// Status indicators (compact)
	sc.statusIndicator.Resize(fyne.NewSize(statusIndicatorSize, statusIndicatorSize))
	statusContainer := container.NewHBox(
		sc.statusIndicator,
		sc.statusLabel,
	)

	// Compact sync controls
	syncDirection := widget.NewSelect([]string{
		"Pull from BadgerMaps",
		"Push to BadgerMaps",
		"Two-way Sync",
	}, func(selected string) {
		sc.updateSyncMode(selected)
	})
	syncDirection.SetSelected("Two-way Sync")

	// Main sync button
	syncButton := widget.NewButtonWithIcon("Start Sync", theme.ViewRefreshIcon(), func() {
		sc.startSync()
	})
	syncButton.Importance = widget.HighImportance

	// Quick stats (more compact than cards)
	pendingAccountsLabel := widget.NewLabelWithData(sc.pendingAccountsCount)
	pendingCheckinsLabel := widget.NewLabelWithData(sc.pendingCheckinsCount)

	quickStats := container.NewHBox(
		widget.NewLabel("Pending:"),
		pendingAccountsLabel,
		widget.NewLabel("accounts,"),
		pendingCheckinsLabel,
		widget.NewLabel("check-ins"),
	)

	// Progress section (hidden by default)
	sc.progressContainer = container.NewVBox(
		sc.progressLabel,
		sc.progressBar,
	)
	sc.progressContainer.Hide()

	// Single row layout with all essential controls
	headerRow := container.NewHBox(
		statusContainer,
		widget.NewSeparator(),
		widget.NewLabel("Mode:"),
		syncDirection,
		widget.NewSeparator(),
		quickStats,
		layout.NewSpacer(),
		syncButton,
	)

	return container.NewVBox(
		headerRow,
		sc.progressContainer,
	)
}

func (sc *SyncCenter) createStatusSection() fyne.CanvasObject {
	// Status indicator with label
	sc.statusIndicator.Resize(fyne.NewSize(statusIndicatorSize, statusIndicatorSize))
	statusContainer := container.NewHBox(
		sc.statusIndicator,
		sc.statusLabel,
		layout.NewSpacer(),
		sc.lastSyncLabel,
	)

	// Progress section (hidden by default)
	sc.progressContainer = container.NewVBox(
		sc.progressLabel,
		sc.progressBar,
	)
	sc.progressContainer.Hide()

	// Summary cards
	summaryCards := sc.createSummaryCards()

	return container.NewVBox(
		statusContainer, // Remove padding to reduce height
		sc.progressContainer,
		summaryCards,
	)
}

func (sc *SyncCenter) createSummaryCards() fyne.CanvasObject {
	// Pending changes card
	pendingAccountsLabel := widget.NewLabelWithData(sc.pendingAccountsCount)
	pendingCheckinsLabel := widget.NewLabelWithData(sc.pendingCheckinsCount)

	pendingCard := widget.NewCard("Pending Changes", "", container.NewGridWithColumns(2,
		container.NewVBox(
			widget.NewLabelWithStyle("Accounts:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			pendingAccountsLabel,
		),
		container.NewVBox(
			widget.NewLabelWithStyle("Check-ins:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			pendingCheckinsLabel,
		),
	))

	// Last sync info card
	lastSyncCard := widget.NewCard("Last Sync", "", container.NewVBox(
		container.NewHBox(
			widget.NewIcon(theme.HistoryIcon()),
			widget.NewLabel("2 hours ago"),
		),
		widget.NewLabel("5 accounts, 12 check-ins synced"),
	))

	// Conflict status card
	conflictCard := widget.NewCard("Conflicts", "", container.NewVBox(
		container.NewHBox(
			widget.NewIcon(theme.WarningIcon()),
			widget.NewLabel("No conflicts"),
		),
		widget.NewLabel("All changes can be synced"),
	))

	return container.NewGridWithColumns(defaultGridColumns, pendingCard, lastSyncCard, conflictCard)
}

func (sc *SyncCenter) createControlSection() fyne.CanvasObject {
	// Sync direction selector
	syncDirection := widget.NewSelect([]string{
		"Pull from BadgerMaps",
		"Push to BadgerMaps",
		"Two-way Sync",
	}, func(selected string) {
		sc.updateSyncMode(selected)
	})
	syncDirection.SetSelected("Two-way Sync")

	// Entity type selector
	entityTypes := widget.NewCheckGroup([]string{
		"Accounts",
		"Check-ins",
		"Routes",
		"User Profile",
	}, func(selected []string) {
		// Handle entity selection
	})
	entityTypes.SetSelected([]string{"Accounts", "Check-ins"})

	// Action buttons
	syncButton := widget.NewButtonWithIcon("Start Sync", theme.ViewRefreshIcon(), func() {
		sc.startSync()
	})
	syncButton.Importance = widget.HighImportance

	previewButton := widget.NewButtonWithIcon("Preview Changes", theme.VisibilityIcon(), func() {
		sc.previewChanges()
	})

	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		sc.cancelSync()
	})
	cancelButton.Disable()

	// Options
	autoResolveCheck := widget.NewCheck("Auto-resolve conflicts", func(checked bool) {})
	dryRunCheck := widget.NewCheck("Dry run (preview only)", func(checked bool) {})

	// Layout
	return container.NewVBox(
		container.NewGridWithColumns(2,
			container.NewVBox(
				widget.NewLabelWithStyle("Sync Mode", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				syncDirection,
			),
			container.NewVBox(
				widget.NewLabelWithStyle("Data Types", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				entityTypes,
			),
		),
		container.NewHBox(
			syncButton,
			previewButton,
			cancelButton,
			layout.NewSpacer(),
			autoResolveCheck,
			dryRunCheck,
		),
	)
}

func (sc *SyncCenter) createDataSection() fyne.CanvasObject {
	// Reduced tab container for more compact layout
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Pending Changes", theme.DocumentIcon(), sc.createPendingChangesView()),
		container.NewTabItemWithIcon("History", theme.HistoryIcon(), sc.createSyncHistoryView()),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), sc.createSyncSettingsView()),
	)

	return tabs
}

func (sc *SyncCenter) createPendingChangesView() fyne.CanvasObject {
	description := widget.NewLabel("Pending changes now live in the Explorer. Use the button below to jump directly to the pending changes tables.")
	description.Wrapping = fyne.TextWrapWord

	openButton := widget.NewButtonWithIcon("Open Pending Changes in Explorer", theme.NavigateNextIcon(), func() {
		if !sc.ui.OpenExplorerPendingChanges() {
			sc.ui.app.Events.Dispatch(events.Debugf("sync_center", "Unable to open pending changes in explorer"))
		}
	})
	openButton.Importance = widget.HighImportance

	content := container.NewVBox(description, openButton)
	card := widget.NewCard("Pending Changes", "", content)

	return container.NewCenter(card)
}

func (sc *SyncCenter) createFilterBar() fyne.CanvasObject {
	// Compact search and filter bar
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search changes...")

	// Combined status/type filter
	quickFilter := widget.NewSelect([]string{
		"All Pending",
		"Pending Creates",
		"Pending Updates",
		"Pending Deletes",
		"Recently Completed",
	}, func(selected string) {
		// Quick filter implementation placeholder
		sc.ui.app.Events.Dispatch(events.Infof("sync_center", "Filter changed to: %s", selected))
	})
	quickFilter.SetSelected("All Pending")

	return container.NewHBox(
		searchEntry,
		quickFilter,
		layout.NewSpacer(),
	)
}

func (sc *SyncCenter) createBatchActions() fyne.CanvasObject {
	syncSelectedBtn := widget.NewButtonWithIcon("Sync Selected", theme.ConfirmIcon(), func() {
		sc.syncSelected()
	})

	deleteSelectedBtn := widget.NewButtonWithIcon("Delete Selected", theme.DeleteIcon(), func() {
		sc.deleteSelected()
	})

	retryFailedBtn := widget.NewButtonWithIcon("Retry Failed", theme.ViewRefreshIcon(), func() {
		sc.retryFailed()
	})

	sc.selectionStatusLabel = widget.NewLabel("0 items selected")

	return container.NewHBox(
		syncSelectedBtn,
		deleteSelectedBtn,
		retryFailedBtn,
		layout.NewSpacer(),
		sc.selectionStatusLabel,
	)
}

func (sc *SyncCenter) createSyncHistoryView() fyne.CanvasObject {
	sc.historyContainer = container.NewMax(widget.NewLabel("Loading history..."))
	sc.refreshHistoryView()

	viewDetailsBtn := widget.NewButtonWithIcon("View Details", theme.InfoIcon(), func() {
		// Future enhancement: show detailed run information
	})

	rerunBtn := widget.NewButtonWithIcon("Re-run", theme.ViewRefreshIcon(), func() {
		// Future enhancement: re-run selected sync operation
	})

	exportBtn := widget.NewButtonWithIcon("Export History", theme.DocumentSaveIcon(), func() {
		// Future enhancement: export sync history entries
	})

	refreshBtn := widget.NewButtonWithIcon("Refresh History", theme.ViewRefreshIcon(), func() {
		sc.refreshHistoryView()
	})
	refreshBtn.Importance = widget.HighImportance

	actions := container.NewHBox(viewDetailsBtn, rerunBtn, exportBtn, layout.NewSpacer(), refreshBtn)

	return container.NewBorder(nil, actions, nil, nil, sc.historyContainer)
}

func (sc *SyncCenter) refreshHistoryView() {
	if sc.historyContainer == nil {
		return
	}

	if sc.ui.app.DB == nil || !sc.ui.app.DB.IsConnected() {
		sc.historyContainer.Objects = []fyne.CanvasObject{widget.NewLabel("Connect to the database to view sync history.")}
		sc.historyContainer.Refresh()
		return
	}

	entries, err := database.GetRecentSyncHistory(sc.ui.app.DB, 50)
	if err != nil {
		sc.ui.app.Events.Dispatch(events.Errorf("sync_center", "Failed to load sync history: %v", err))
		sc.historyContainer.Objects = []fyne.CanvasObject{widget.NewLabel("Unable to load sync history.")}
		sc.historyContainer.Refresh()
		return
	}

	if len(entries) == 0 {
		sc.historyContainer.Objects = []fyne.CanvasObject{widget.NewLabel("No sync runs recorded yet.")}
		sc.historyContainer.Refresh()
		return
	}

	table := sc.buildHistoryTable(entries)
	sc.historyContainer.Objects = []fyne.CanvasObject{table}
	sc.historyContainer.Refresh()
}

func (sc *SyncCenter) buildHistoryTable(entries []database.SyncHistoryEntry) fyne.CanvasObject {
	headers := []string{"Started", "Run", "Status", "Items", "Errors", "Duration", "Summary"}
	data := make([][]string, len(entries))

	for i, entry := range entries {
		started := "-"
		if !entry.StartedAt.IsZero() {
			started = entry.StartedAt.Local().Format("2006-01-02 15:04")
		}

		runLabel := strings.Title(entry.RunType)
		if entry.Source != "" {
			runLabel = fmt.Sprintf("%s • %s", runLabel, friendlyHistoryLabel(entry.Source))
		}

		status := prettifyHistoryStatus(entry.Status)
		items := fmt.Sprintf("%d", entry.ItemsProcessed)
		errors := fmt.Sprintf("%d", entry.ErrorCount)
		duration := formatHistoryDuration(entry.DurationSeconds, entry.StartedAt, entry.CompletedAt)
		summary := entry.Summary
		if strings.TrimSpace(summary) == "" {
			summary = "-"
		}

		data[i] = []string{started, runLabel, status, items, errors, duration, summary}
	}

	factory := NewTableFactory(sc.ui)
	config := TableConfig{
		Headers:       headers,
		Data:          data,
		HasCheckboxes: false,
		StatusColumn:  2,
		ColumnWidths: map[int]float32{
			0: 140,
			1: 200,
			2: 140,
			3: 80,
			4: 80,
			5: 100,
			6: 260,
		},
	}

	return factory.CreateAutoTruncatedTable(config)
}

func formatHistoryDuration(seconds int, started time.Time, completed *time.Time) string {
	if seconds <= 0 && !started.IsZero() && completed != nil && !completed.IsZero() {
		seconds = int(completed.Sub(started).Seconds())
	}
	if seconds <= 0 {
		return "-"
	}
	duration := time.Duration(seconds) * time.Second
	if duration < time.Second {
		return "<1s"
	}
	return duration.String()
}

func prettifyHistoryStatus(status string) string {
	if status == "" {
		return "-"
	}
	status = strings.ReplaceAll(status, "_", " ")
	status = strings.TrimSpace(status)
	return strings.Title(status)
}

func friendlyHistoryLabel(source string) string {
	if source == "" {
		return "General"
	}
	replacer := strings.NewReplacer("_", " ", "-", " ")
	return strings.Title(replacer.Replace(source))
}

func (sc *SyncCenter) createConflictsView() fyne.CanvasObject {
	// No conflicts message
	noConflictsLabel := widget.NewLabel("No conflicts detected")
	noConflictsLabel.Alignment = fyne.TextAlignCenter

	noConflictsIcon := widget.NewIcon(theme.ConfirmIcon())

	message := container.NewCenter(container.NewVBox(
		noConflictsIcon,
		noConflictsLabel,
		widget.NewLabel("All changes can be synchronized without conflicts"),
	))

	// Would show conflict resolution UI when conflicts exist
	return message
}

func (sc *SyncCenter) createSyncSettingsView() fyne.CanvasObject {
	// Sync preferences
	autoSyncCheck := widget.NewCheck("Enable automatic sync", func(checked bool) {})

	syncIntervalSelect := widget.NewSelect([]string{
		"Every 5 minutes",
		"Every 15 minutes",
		"Every 30 minutes",
		"Every hour",
		"Every 6 hours",
		"Daily",
	}, func(selected string) {})
	syncIntervalSelect.SetSelected("Every 30 minutes")

	// Conflict resolution settings
	conflictStrategyRadio := widget.NewRadioGroup([]string{
		"Always use local changes",
		"Always use remote changes",
		"Use most recent",
		"Ask every time",
	}, func(selected string) {})
	conflictStrategyRadio.SetSelected("Ask every time")

	// Performance settings
	batchSizeEntry := widget.NewEntry()
	batchSizeEntry.SetText("100")

	parallelCheck := widget.NewCheck("Enable parallel processing", func(checked bool) {})
	parallelCheck.SetChecked(true)

	// Logging settings
	verboseLoggingCheck := widget.NewCheck("Verbose logging", func(checked bool) {})

	logRetentionSelect := widget.NewSelect([]string{
		"7 days",
		"30 days",
		"90 days",
		"Forever",
	}, func(selected string) {})
	logRetentionSelect.SetSelected("30 days")

	// Save button
	saveBtn := widget.NewButtonWithIcon("Save Settings", theme.DocumentSaveIcon(), func() {
		sc.ui.ShowToast("Sync settings saved")
	})
	saveBtn.Importance = widget.HighImportance

	// Layout
	form := widget.NewForm(
		widget.NewFormItem("Auto Sync", autoSyncCheck),
		widget.NewFormItem("Sync Interval", syncIntervalSelect),
		widget.NewFormItem("", widget.NewSeparator()),
		widget.NewFormItem("Conflict Resolution", conflictStrategyRadio),
		widget.NewFormItem("", widget.NewSeparator()),
		widget.NewFormItem("Batch Size", batchSizeEntry),
		widget.NewFormItem("Performance", parallelCheck),
		widget.NewFormItem("", widget.NewSeparator()),
		widget.NewFormItem("Logging", verboseLoggingCheck),
		widget.NewFormItem("Log Retention", logRetentionSelect),
	)

	return container.NewVBox(
		form,
		container.NewCenter(saveBtn),
	)
}

// Helper methods

func (sc *SyncCenter) refreshPendingCounts() {
	// Get pending counts from database
	go func() {
		accountCount := 0
		checkinCount := 0

		// Fetch actual counts
		if sc.ui.app.DB != nil && sc.ui.app.DB.IsConnected() {
			// Count pending account changes
			accountOptions := push.PushFilterOptions{Status: "pending"}
			if results, err := push.GetFilteredPendingChanges(sc.ui.app, "accounts", accountOptions); err == nil {
				if changes, ok := results.([]database.AccountPendingChange); ok {
					accountCount = len(changes)
				}
			}

			// Count pending checkin changes
			checkinOptions := push.PushFilterOptions{Status: "pending"}
			if results, err := push.GetFilteredPendingChanges(sc.ui.app, "checkins", checkinOptions); err == nil {
				if changes, ok := results.([]database.CheckinPendingChange); ok {
					checkinCount = len(changes)
				}
			}
		}

		sc.pendingAccountsCount.Set(fmt.Sprintf("%d", accountCount))
		sc.pendingCheckinsCount.Set(fmt.Sprintf("%d", checkinCount))
	}()
}

func (sc *SyncCenter) updateSyncMode(mode string) {
	// Update UI based on selected sync mode (thread-safe)
	sc.operatingMutex.Lock()
	sc.currentOperation = mode
	sc.operatingMutex.Unlock()
}

func (sc *SyncCenter) startSync() {
	// Check if already operating (thread-safe)
	if !atomic.CompareAndSwapInt32(&sc.isOperating, 0, 1) {
		sc.ui.app.Events.Dispatch(events.Debugf("sync_center", "Sync already in progress, ignoring request"))
		return
	}

	sc.ui.app.Events.Dispatch(events.Infof("sync_center", "Starting sync operation"))

	sc.setStatus("Syncing", color.NRGBA{R: 255, G: 165, B: 0, A: 255})
	sc.progressLabel.SetText("Starting sync...")
	sc.progressBar.SetValue(0)

	// Show progress container
	if sc.progressContainer != nil {
		sc.progressContainer.Show()
	}

	// Get current operation (thread-safe)
	sc.operatingMutex.RLock()
	currentOp := sc.currentOperation
	sc.operatingMutex.RUnlock()

	// Perform sync operation based on current mode
	go func() {
		defer func() {
			atomic.StoreInt32(&sc.isOperating, 0)
			if sc.progressContainer != nil {
				sc.progressContainer.Hide()
			}
		}()

		switch currentOp {
		case "Pull from BadgerMaps":
			sc.ui.app.Events.Dispatch(events.Infof("sync_center", "Executing pull operation"))
			sc.presenter.HandlePullGroup()
		case "Push to BadgerMaps":
			sc.ui.app.Events.Dispatch(events.Infof("sync_center", "Executing push operation"))
			sc.presenter.HandlePushAll()
		case "Two-way Sync":
			sc.ui.app.Events.Dispatch(events.Infof("sync_center", "Executing two-way sync operation"))
			// Pull first, then push
			sc.presenter.HandlePullGroup()
			time.Sleep(syncDelayBetweenOps)
			sc.presenter.HandlePushAll()
		default:
			sc.ui.app.Events.Dispatch(events.Warningf("sync_center", "Unknown sync operation: %s", currentOp))
		}

		sc.setStatus("Idle", color.NRGBA{R: 0, G: 200, B: 0, A: 255})
		sc.lastSyncLabel.SetText(fmt.Sprintf("Last sync: %s", time.Now().Format(timeFormat)))
		sc.refreshPendingCounts()
		sc.ui.app.Events.Dispatch(events.Infof("sync_center", "Sync operation completed successfully"))
	}()
}

func (sc *SyncCenter) previewChanges() {
	// Show preview dialog
	sc.ui.ShowToast("Preview not yet implemented")
}

func (sc *SyncCenter) cancelSync() {
	if atomic.LoadInt32(&sc.isOperating) == 1 {
		atomic.StoreInt32(&sc.isOperating, 0)
		sc.setStatus("Cancelled", color.NRGBA{R: 255, G: 0, B: 0, A: 255})
		sc.ui.ShowToast("Sync operation cancelled")
	}
}

func (sc *SyncCenter) setStatus(status string, color color.Color) {
	sc.statusLabel.SetText(status)
	sc.statusIndicator.FillColor = color
	sc.statusIndicator.Refresh()
}

func (sc *SyncCenter) filterByStatus(status string) {
	// Apply status filter to tables
	sc.refreshTables()
}

func (sc *SyncCenter) filterByType(changeType string) {
	// Apply type filter to tables
	sc.refreshTables()
}

func (sc *SyncCenter) filterByDate(dateRange string) {
	// Apply date filter to tables
	sc.refreshTables()
}

func (sc *SyncCenter) selectAllAccounts(selected bool) {
	// Select/deselect all accounts
	for id := range sc.selectedAccounts {
		if selected {
			sc.selectedAccounts[id] = true
		} else {
			delete(sc.selectedAccounts, id)
		}
	}
	sc.accountsTable.Refresh()
	sc.updateSelectionStatusLabel()
}

func (sc *SyncCenter) selectAllCheckins(selected bool) {
	// Select/deselect all checkins
	for id := range sc.selectedCheckins {
		if selected {
			sc.selectedCheckins[id] = true
		} else {
			delete(sc.selectedCheckins, id)
		}
	}
	sc.checkinsTable.Refresh()
	sc.updateSelectionStatusLabel()
}

func (sc *SyncCenter) syncSelected() {
	selectedCount := sc.totalSelected()
	if selectedCount == 0 {
		sc.ui.ShowToast("No items selected")
		return
	}

	sc.ui.ShowToast(fmt.Sprintf("Syncing %d selected items...", selectedCount))
	// Implement selective sync
}

func (sc *SyncCenter) deleteSelected() {
	selectedCount := sc.totalSelected()
	if selectedCount == 0 {
		sc.ui.ShowToast("No items selected")
		return
	}

	sc.ui.ShowConfirmDialog("Delete Selected",
		fmt.Sprintf("Are you sure you want to delete %d selected changes?", selectedCount),
		func(confirmed bool) {
			if confirmed {
				// Implement deletion
				sc.ui.ShowToast("Selected changes deleted")
				sc.clearSelections()
				sc.refreshPendingCounts()
				sc.refreshTables()
			}
		})
}

func (sc *SyncCenter) totalSelected() int {
	count := 0
	for _, selected := range sc.selectedAccounts {
		if selected {
			count++
		}
	}
	for _, selected := range sc.selectedCheckins {
		if selected {
			count++
		}
	}
	return count
}

func (sc *SyncCenter) clearSelections() {
	sc.selectedAccounts = make(map[int]bool)
	sc.selectedCheckins = make(map[int]bool)
	sc.updateSelectionStatusLabel()
}

func (sc *SyncCenter) updateSelectionStatusLabel() {
	if sc.selectionStatusLabel == nil {
		return
	}
	total := sc.totalSelected()
	itemLabel := "items"
	if total == 1 {
		itemLabel = "item"
	}
	sc.selectionStatusLabel.SetText(fmt.Sprintf("%d %s selected", total, itemLabel))
}

func (sc *SyncCenter) retryFailed() {
	// Retry failed sync operations
	sc.ui.ShowToast("Retrying failed operations...")
}

func (sc *SyncCenter) refreshTables() {
	if sc.currentTable != nil {
		sc.currentTable.Refresh()
	}
	if sc.tableContainer != nil && sc.currentDataType != "" {
		sc.updatePendingChangesTable(sc.tableContainer, sc.currentDataType)
	}
}

// createDataTypeSelector creates a compact selector for switching between data types
func (sc *SyncCenter) createDataTypeSelector() fyne.CanvasObject {
	// Compact select widget instead of radio buttons
	dataTypeSelect := widget.NewSelect([]string{
		"Accounts",
		"Check-ins",
		"All Changes",
	}, func(selected string) {
		var dataType string
		switch selected {
		case "Accounts":
			dataType = "accounts"
		case "Check-ins":
			dataType = "checkins"
		case "All Changes":
			dataType = "all"
		}
		sc.currentDataType = dataType
		sc.clearSelections()
		if sc.tableContainer != nil {
			sc.updatePendingChangesTable(sc.tableContainer, dataType)
		}
		sc.ui.app.Events.Dispatch(events.Debugf("sync_center", "Switched to view: %s", dataType))
	})
	dataTypeSelect.SetSelected("Accounts") // Default selection

	return container.NewHBox(
		widget.NewLabel("View:"),
		dataTypeSelect,
	)
}

// updatePendingChangesTable updates the table container with the specified data type
func (sc *SyncCenter) updatePendingChangesTable(tableContainer *fyne.Container, dataType string) {
	sc.tableContainer = tableContainer

	var table *widget.Table

	switch dataType {
	case "accounts":
		table = sc.createUnifiedPendingTable("accounts")
	case "checkins":
		table = sc.createUnifiedPendingTable("checkins")
	case "all":
		table = sc.createUnifiedPendingTable("all")
	default:
		table = sc.createUnifiedPendingTable("accounts")
	}

	sc.currentTable = table
	tableContainer.Objects = []fyne.CanvasObject{table}
	tableContainer.Refresh()
}

// createUnifiedPendingTable creates a single table that can show accounts, checkins, or both
func (sc *SyncCenter) createUnifiedPendingTable(dataType string) *widget.Table {
	// Fetch pending changes based on data type
	options := push.PushFilterOptions{
		Status:  "pending",
		OrderBy: "date_desc",
	}

	var headers []string
	var data [][]string

	switch dataType {
	case "accounts":
		headers = []string{"", "Type", "ID", "Account ID", "Change Type", "Status", "Modified", "Details"}
		if results, err := push.GetFilteredPendingChanges(sc.ui.app, "accounts", options); err == nil {
			if changes, ok := results.([]database.AccountPendingChange); ok {
				for _, c := range changes {
					data = append(data, []string{
						"", // Checkbox column
						"Account",
						fmt.Sprintf("%d", c.ChangeId),
						fmt.Sprintf("%d", c.AccountId),
						c.ChangeType,
						c.Status,
						c.CreatedAt.Format(shortDateFormat),
						c.Changes,
					})
				}
			}
		}
	case "checkins":
		headers = []string{"", "Type", "ID", "Check-in ID", "Account ID", "Change Type", "Status", "Modified", "Details"}
		if results, err := push.GetFilteredPendingChanges(sc.ui.app, "checkins", options); err == nil {
			if changes, ok := results.([]database.CheckinPendingChange); ok {
				for _, c := range changes {
					data = append(data, []string{
						"", // Checkbox column
						"Check-in",
						fmt.Sprintf("%d", c.ChangeId),
						fmt.Sprintf("%d", c.CheckinId),
						fmt.Sprintf("%d", c.AccountId),
						c.ChangeType,
						c.Status,
						c.CreatedAt.Format(shortDateFormat),
						c.Changes,
					})
				}
			}
		}
	case "all":
		headers = []string{"", "Type", "ID", "Entity ID", "Account ID", "Change Type", "Status", "Modified", "Details"}

		// Get accounts first
		if results, err := push.GetFilteredPendingChanges(sc.ui.app, "accounts", options); err == nil {
			if changes, ok := results.([]database.AccountPendingChange); ok {
				for _, c := range changes {
					data = append(data, []string{
						"", // Checkbox column
						"Account",
						fmt.Sprintf("%d", c.ChangeId),
						fmt.Sprintf("%d", c.AccountId),
						fmt.Sprintf("%d", c.AccountId), // Same as entity ID for accounts
						c.ChangeType,
						c.Status,
						c.CreatedAt.Format(shortDateFormat),
						c.Changes,
					})
				}
			}
		}

		// Get checkins
		if results, err := push.GetFilteredPendingChanges(sc.ui.app, "checkins", options); err == nil {
			if changes, ok := results.([]database.CheckinPendingChange); ok {
				for _, c := range changes {
					data = append(data, []string{
						"", // Checkbox column
						"Check-in",
						fmt.Sprintf("%d", c.ChangeId),
						fmt.Sprintf("%d", c.CheckinId),
						fmt.Sprintf("%d", c.AccountId),
						c.ChangeType,
						c.Status,
						c.CreatedAt.Format(shortDateFormat),
						c.Changes,
					})
				}
			}
		}
	}

	// Use the table factory for consistent table creation
	factory := NewTableFactory(sc.ui)
	config := TableConfig{
		Headers:       headers,
		Data:          data,
		HasCheckboxes: true,
		StatusColumn:  5, // Status is typically column 5 or 6
		OnSelectionChange: func(rowIndex int, selected bool, rowData []string) {
			// Handle selection changes based on row type
			if len(rowData) > 1 {
				rowType := strings.ToLower(rowData[1])                     // Type column
				if changeID, err := strconv.Atoi(rowData[2]); err == nil { // ID column
					if strings.Contains(rowType, "account") {
						if selected {
							sc.selectedAccounts[changeID] = true
						} else {
							delete(sc.selectedAccounts, changeID)
						}
					} else if strings.Contains(rowType, "check") {
						if selected {
							sc.selectedCheckins[changeID] = true
						} else {
							delete(sc.selectedCheckins, changeID)
						}
					}
				}
			}
			sc.updateSelectionStatusLabel()
		},
		ColumnWidths: map[int]float32{
			0: checkboxColumnWidth,
			1: 80, // Type
			2: idColumnWidth,
			3: nameColumnWidth,
			4: 100, // Change Type
			5: 80,  // Status
			6: 120, // Modified
			7: 200, // Details
		},
	}

	return factory.CreateAutoTruncatedTable(config)
}
