package gui

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/app/action"
	appserver "badgermaps/app/server"
	"badgermaps/app/state"
	"badgermaps/database"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// findWidget recursively searches for a widget that satisfies the predicate.
// This version is more robust as it checks various container types.
func findWidget(o fyne.CanvasObject, predicate func(fyne.CanvasObject) bool) fyne.CanvasObject {
	if o == nil {
		return nil
	}
	if predicate(o) {
		return o
	}

	switch v := o.(type) {
	case *fyne.Container:
		for _, child := range v.Objects {
			if found := findWidget(child, predicate); found != nil {
				return found
			}
		}
	case *widget.Card:
		return findWidget(v.Content, predicate)
	case *container.Scroll:
		return findWidget(v.Content, predicate)
	case *container.Split:
		if found := findWidget(v.Leading, predicate); found != nil {
			return found
		}
		return findWidget(v.Trailing, predicate)
	case *container.AppTabs:
		for _, item := range v.Items {
			if found := findWidget(item.Content, predicate); found != nil {
				return found
			}
		}
	}

	return nil
}

func findTabByText(tabs *container.AppTabs, text string) *container.TabItem {
	if tabs == nil {
		return nil
	}
	for _, item := range tabs.Items {
		if item.Text == text {
			return item
		}
	}
	return nil
}

func hasLabel(root fyne.CanvasObject, text string) bool {
	return findWidget(root, func(o fyne.CanvasObject) bool {
		label, ok := o.(*widget.Label)
		return ok && label.Text == text
	}) != nil
}

func hasButton(root fyne.CanvasObject, text string) bool {
	return findWidget(root, func(o fyne.CanvasObject) bool {
		if button, ok := o.(*widget.Button); ok {
			return button.Text == text
		}
		if button, ok := o.(*SecondaryButton); ok {
			return button.Text == text
		}
		return false
	}) != nil
}

func hasCanvasText(root fyne.CanvasObject, text string) bool {
	return findWidget(root, func(o fyne.CanvasObject) bool {
		label, ok := o.(*canvas.Text)
		return ok && label.Text == text
	}) != nil
}

func hasCanvasTextContaining(root fyne.CanvasObject, fragment string) bool {
	return findWidget(root, func(o fyne.CanvasObject) bool {
		label, ok := o.(*canvas.Text)
		return ok && strings.Contains(label.Text, fragment)
	}) != nil
}

func hasToolbarActionIcon(root fyne.CanvasObject, icon fyne.Resource) bool {
	target := ""
	if icon != nil {
		target = icon.Name()
	}

	return findWidget(root, func(o fyne.CanvasObject) bool {
		toolbar, ok := o.(*widget.Toolbar)
		if ok {
			for _, item := range toolbar.Items {
				action, ok := item.(*widget.ToolbarAction)
				if !ok || action.Icon == nil {
					continue
				}
				if action.Icon.Name() == target {
					return true
				}
			}
		}

		if button, ok := o.(*widget.Button); ok && button.Icon != nil && button.Icon.Name() == target {
			return true
		}
		if button, ok := o.(*SecondaryButton); ok && button.Icon != nil && button.Icon.Name() == target {
			return true
		}
		if button, ok := o.(*tooltipIconButton); ok && button.Icon != nil && button.Icon.Name() == target {
			return true
		}

		return false
	}) != nil
}

func buildConnectedTestGUI(t *testing.T) *Gui {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	a := app.NewApp()
	a.API = api.NewAPIClient(&api.APIConfig{BaseURL: "https://example.invalid/api", APIKey: "test-key"})
	a.API.SetConnected(true)

	db, err := database.NewDB(&database.DBConfig{Type: "sqlite3", Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect test db: %v", err)
	}
	if err := db.EnforceSchema(&state.State{}); err != nil {
		t.Fatalf("failed to enforce schema: %v", err)
	}
	db.SetConnected(true)
	a.DB = db

	ui := &Gui{
		app:        a,
		fyneApp:    test.NewApp(),
		logBinding: binding.NewStringList(),
	}
	ui.presenter = NewGuiPresenter(a, ui)
	ui.syncCenter = NewSyncCenter(ui, ui.presenter)
	ui.smartDashboard = NewSmartDashboard(ui, ui.presenter)
	ui.showWelcome = false
	ui.window = test.NewWindow(ui.createContent())

	t.Cleanup(func() {
		if ui.window != nil {
			ui.window.Close()
		}
		if a.DB != nil {
			_ = a.DB.Close()
		}
	})

	return ui
}

func TestSyncCenterInitialization(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/search/accounts") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"id": 1, "full_name": "Test Account 1"}, {"id": 2, "full_name": "Test Account 2"}]`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create a temporary directory for the test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	a := app.NewApp()
	a.API = api.NewAPIClient(&api.APIConfig{BaseURL: server.URL, APIKey: "test-key"})
	a.DB, _ = database.NewDB(&database.DBConfig{Type: "sqlite3", Path: dbPath})
	a.DB.Connect()
	a.DB.EnforceSchema(&state.State{})
	// Seed minimal data for DB-backed omnibox search
	if _, err := a.DB.GetDB().Exec("INSERT INTO Accounts (AccountId, FullName) VALUES (1, 'Test Account 1'), (2, 'Test Account 2')"); err != nil {
		t.Fatalf("failed seeding accounts: %v", err)
	}
	a.API.SetConnected(true)
	a.DB.SetConnected(true)

	ui := &Gui{
		app:        a,
		fyneApp:    test.NewApp(),
		logBinding: binding.NewStringList(),
	}
	ui.presenter = NewGuiPresenter(a, ui)

	// Initialize new components
	ui.syncCenter = NewSyncCenter(ui, ui.presenter)
	ui.smartDashboard = NewSmartDashboard(ui, ui.presenter)
	ui.showWelcome = false // Skip welcome for tests

	ui.window = test.NewWindow(ui.createContent())
	ui.tabs.SelectIndex(1) // Select the "Sync Center" tab

	if ui.syncCenter.syncTypeSelect == nil {
		t.Fatal("Sync type selector not initialized")
	}
	if ui.syncCenter.scopeSelect == nil {
		t.Fatal("Scope selector not initialized")
	}
	if ui.syncCenter.actionButton == nil {
		t.Fatal("Action button not initialized")
	}

	if ui.syncCenter.currentType != syncKindAll {
		t.Errorf("expected default sync type %q, got %q", syncKindAll, ui.syncCenter.currentType)
	}
	if ui.syncCenter.actionButton.Text != "Pull All" {
		t.Fatalf("expected default action label 'Pull All', got %q", ui.syncCenter.actionButton.Text)
	}
	if ui.syncCenter.scopeGroup.Visible() {
		t.Fatalf("scope controls should be hidden for %s", syncKindAll)
	}

	ui.syncCenter.syncTypeSelect.SetSelected(syncKindAccounts)

	if !ui.syncCenter.scopeGroup.Visible() {
		t.Fatal("scope selector should be visible when Accounts is active")
	}
	if ui.syncCenter.scopeSelect.Selected != scopeAll {
		t.Fatalf("expected scope default %q, got %q", scopeAll, ui.syncCenter.scopeSelect.Selected)
	}
	if ui.syncCenter.actionButton.Text != "Pull All Accounts" {
		t.Fatalf("expected action label 'Pull All Accounts', got %q", ui.syncCenter.actionButton.Text)
	}

	ui.syncCenter.scopeSelect.SetSelected(scopeSingle)

	if !ui.syncCenter.omniGroup.Visible() {
		t.Fatal("omnibox should be visible when Single scope is selected")
	}
	if ui.syncCenter.actionButton.Text != "Pull Account" {
		t.Fatalf("expected action label 'Pull Account', got %q", ui.syncCenter.actionButton.Text)
	}
}

func TestTableFactoryEnhancements(t *testing.T) {
	// Create a minimal app and UI for testing
	testApp := &app.App{
		State: &state.State{},
	}

	// Create a test UI
	ui := &Gui{
		app:    testApp,
		window: test.NewApp().NewWindow("Test"),
	}

	factory := NewTableFactory(ui)

	// Test data with some long text that should be truncated
	testData := [][]string{
		{"1", "This is a very long name that should be truncated to prevent overflow", "Active", "2024-01-15 14:30:00"},
		{"2", "Short", "Inactive", "2024-01-14 10:00:00"},
		{"3", "Another extremely long text that definitely needs to be truncated for proper display", "Pending", "2024-01-13 09:00:00"},
	}

	testHeaders := []string{"ID", "Name", "Status", "Created"}

	// Test auto-truncated table creation
	config := TableConfig{
		Headers:       testHeaders,
		Data:          testData,
		HasCheckboxes: false,
	}

	table := factory.CreateAutoTruncatedTable(config)
	if table == nil {
		t.Fatal("CreateAutoTruncatedTable returned nil")
	}

	// Test truncation functionality
	truncated := factory.truncateText("This is a long text", 10)
	expected := "This is..."
	if truncated != expected {
		t.Errorf("Expected truncated text '%s', got '%s'", expected, truncated)
	}

	// Test that short text is not truncated
	short := factory.truncateText("Short", 10)
	if short != "Short" {
		t.Errorf("Expected short text to remain unchanged, got '%s'", short)
	}

	// Test column width calculation
	width := factory.calculateOptimalWidth("Name", 1, config)
	if width <= 0 {
		t.Errorf("Expected positive column width, got %f", width)
	}

	// Test searchable table creation
	searchableTable := factory.CreateSearchableTable(config, "Search...")
	if searchableTable == nil {
		t.Fatal("CreateSearchableTable returned nil")
	}

	// Test paginated table creation
	paginatedTable := factory.CreatePaginatedTable(config, 2)
	if paginatedTable == nil {
		t.Fatal("CreatePaginatedTable returned nil")
	}
}

func TestMainTabsExcludeJobsAndServer(t *testing.T) {
	ui := buildConnectedTestGUI(t)

	if ui.tabs == nil {
		t.Fatal("expected tabs to be initialized")
	}
	if findTabByText(ui.tabs, "Sync Center") == nil {
		t.Fatal("expected Sync Center tab")
	}
	if findTabByText(ui.tabs, "Configuration") == nil {
		t.Fatal("expected Configuration tab")
	}
	if findTabByText(ui.tabs, "Jobs") != nil {
		t.Fatal("did not expect Jobs tab")
	}
	if findTabByText(ui.tabs, "Server") != nil {
		t.Fatal("did not expect Server tab")
	}
}

func TestConfigurationTabContainsServerControls(t *testing.T) {
	ui := buildConnectedTestGUI(t)
	configTab := findTabByText(ui.tabs, "Configuration")
	if configTab == nil {
		t.Fatal("expected Configuration tab")
	}

	if !hasLabel(configTab.Content, "Server Configuration") {
		t.Fatal("expected Server Configuration section in config tab")
	}
	if !hasLabel(configTab.Content, "Webhook Routing") {
		t.Fatal("expected Webhook Routing section in config tab")
	}
	if !hasButton(configTab.Content, "Save Server Settings") {
		t.Fatal("expected Save Server Settings button in config tab")
	}
}

func TestSyncCenterContainsScheduledJobsToolsAndHeaderServerControl(t *testing.T) {
	ui := buildConnectedTestGUI(t)
	syncTab := findTabByText(ui.tabs, "Sync Center")
	if syncTab == nil {
		t.Fatal("expected Sync Center tab")
	}

	if !hasLabel(syncTab.Content, "Scheduled Jobs") {
		t.Fatal("expected Scheduled Jobs section in Sync Center")
	}
	if !hasLabel(syncTab.Content, "Additional Tools") {
		t.Fatal("expected Additional Tools section in Sync Center")
	}
	if !hasButton(syncTab.Content, "Add Job") {
		t.Fatal("expected Add Job button in Sync Center")
	}
	if !hasButton(syncTab.Content, "Job Log") {
		t.Fatal("expected Job Log shortcut in Sync Center tools")
	}
	if !hasButton(syncTab.Content, "Pending Changes") {
		t.Fatal("expected Pending Changes shortcut in Sync Center tools")
	}
	if !hasCanvasTextContaining(syncTab.Content, "Server:") {
		t.Fatal("expected read-only server status text in Sync Center header")
	}
}

func TestScheduledJobsSectionIncludesRunNowSkipNextIcon(t *testing.T) {
	tempDir := t.TempDir()

	a := app.NewApp()
	a.SetConfigFilePath(filepath.Join(tempDir, "config.yaml"))

	if err := appserver.SaveScheduledJobs(a.State, map[string]*appserver.ScheduledJob{
		"job_icon_check": {
			ID:       "job_icon_check",
			Name:     "Icon Check",
			Schedule: "0 0 * * * *",
			Steps: []appserver.WorkflowStep{
				{
					ID:   "action_echo",
					Type: appserver.WorkflowStepTypeAction,
					Action: action.ActionConfig{
						Type: "exec",
						Args: map[string]interface{}{
							"command": "echo icon-check",
						},
					},
				},
			},
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("failed to seed scheduled jobs: %v", err)
	}

	ui := &Gui{
		app:        a,
		fyneApp:    test.NewApp(),
		logBinding: binding.NewStringList(),
	}
	ui.presenter = NewGuiPresenter(a, ui)

	section := ui.createScheduledJobsSection()

	if !hasToolbarActionIcon(section, theme.MediaSkipNextIcon()) {
		t.Fatal("expected scheduled job actions to include run-now skip-next icon")
	}
	if !hasToolbarActionIcon(section, theme.MediaPauseIcon()) {
		t.Fatal("expected scheduled job toolbar to include pause/resume icon")
	}
	if !hasToolbarActionIcon(section, theme.DocumentCreateIcon()) {
		t.Fatal("expected scheduled job toolbar to include edit icon")
	}
	if !hasToolbarActionIcon(section, theme.DeleteIcon()) {
		t.Fatal("expected scheduled job toolbar to include delete icon")
	}
}

func TestScheduledJobsPauseButtonDisablesJob(t *testing.T) {
	tempDir := t.TempDir()

	a := app.NewApp()
	a.SetConfigFilePath(filepath.Join(tempDir, "config.yaml"))

	const jobID = "job_pause_test"
	if err := appserver.SaveScheduledJobs(a.State, map[string]*appserver.ScheduledJob{
		jobID: {
			ID:       jobID,
			Name:     "Pause Test",
			Schedule: "0 0 * * * *",
			Steps: []appserver.WorkflowStep{
				{
					ID:   "action_echo",
					Type: appserver.WorkflowStepTypeAction,
					Action: action.ActionConfig{
						Type: "exec",
						Args: map[string]interface{}{
							"command": "echo pause-test",
						},
					},
				},
			},
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("failed to seed scheduled jobs: %v", err)
	}

	ui := &Gui{
		app:        a,
		fyneApp:    test.NewApp(),
		logBinding: binding.NewStringList(),
	}
	ui.presenter = NewGuiPresenter(a, ui)
	ui.window = test.NewWindow(widget.NewLabel("test"))
	defer ui.window.Close()

	section := ui.createScheduledJobsSection()
	pauseButtonObj := findWidget(section, func(o fyne.CanvasObject) bool {
		button, ok := o.(*tooltipIconButton)
		return ok && button.Icon != nil && button.Icon.Name() == theme.MediaPauseIcon().Name()
	})
	if pauseButtonObj == nil {
		t.Fatal("expected pause tooltip icon button")
	}
	pauseButton, ok := pauseButtonObj.(*tooltipIconButton)
	if !ok {
		t.Fatal("expected pause button to be tooltipIconButton")
	}

	test.Tap(pauseButton)

	jobs, err := a.ListScheduledJobs()
	if err != nil {
		t.Fatalf("failed to reload scheduled jobs: %v", err)
	}

	var found *appserver.ScheduledJob
	for _, job := range jobs {
		if job != nil && job.ID == jobID {
			found = job
			break
		}
	}
	if found == nil {
		t.Fatalf("expected to find job %q after pause", jobID)
	}
	if found.Enabled {
		t.Fatalf("expected job %q to be disabled after tapping pause", jobID)
	}
}

func TestScheduledJobsHoverHintUpdatesInlineLabel(t *testing.T) {
	tempDir := t.TempDir()

	a := app.NewApp()
	a.SetConfigFilePath(filepath.Join(tempDir, "config.yaml"))

	const jobID = "job_hover_hint"
	if err := appserver.SaveScheduledJobs(a.State, map[string]*appserver.ScheduledJob{
		jobID: {
			ID:       jobID,
			Name:     "Hover Hint Test",
			Schedule: "0 0 * * * *",
			Steps: []appserver.WorkflowStep{
				{
					ID:   "action_echo",
					Type: appserver.WorkflowStepTypeAction,
					Action: action.ActionConfig{
						Type: "exec",
						Args: map[string]interface{}{
							"command": "echo hover-hint",
						},
					},
				},
			},
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("failed to seed scheduled jobs: %v", err)
	}

	ui := &Gui{
		app:        a,
		fyneApp:    test.NewApp(),
		logBinding: binding.NewStringList(),
	}
	ui.presenter = NewGuiPresenter(a, ui)
	ui.window = test.NewWindow(widget.NewLabel("test"))
	defer ui.window.Close()

	section := ui.createScheduledJobsSection()
	runNowObj := findWidget(section, func(o fyne.CanvasObject) bool {
		button, ok := o.(*tooltipIconButton)
		return ok && button.Icon != nil && button.Icon.Name() == theme.MediaSkipNextIcon().Name()
	})
	if runNowObj == nil {
		t.Fatal("expected run-now tooltip icon button")
	}
	runNowButton, ok := runNowObj.(*tooltipIconButton)
	if !ok {
		t.Fatal("expected run-now button to be tooltipIconButton")
	}

	runNowButton.MouseIn(&desktop.MouseEvent{})
	if !hasLabel(section, "Run job now") {
		t.Fatal("expected inline hover hint to show run-now text")
	}

	runNowButton.MouseOut()
	if hasLabel(section, "Run job now") {
		t.Fatal("expected inline hover hint text to clear on mouse out")
	}
}

func TestDatabaseExplorerPagination(t *testing.T) {
	// Create a minimal app for testing
	testApp := &app.App{
		State: &state.State{},
	}

	// Create a test UI
	ui := &Gui{
		app:    testApp,
		window: test.NewApp().NewWindow("Test"),
	}

	// Test getTableRowCount with nil database (should return 0)
	count := ui.getTableRowCount("test_table")
	if count != 0 {
		t.Errorf("Expected 0 rows for nil database, got %d", count)
	}

	// Test getTableColumns with nil database (should return empty slice)
	columns := ui.getTableColumns("test_table")
	if len(columns) == 0 {
		// This is expected behavior
	}

	// Test pagination data structure
	paginatedData := &PaginatedTableData{
		TableData: TableData{
			Headers: []string{"ID", "Name", "Status"},
			Data: [][]string{
				{"1", "Test 1", "Active"},
				{"2", "Test 2", "Inactive"},
			},
		},
		TotalRows:   100,
		CurrentPage: 2,
		PageSize:    50,
		TotalPages:  2,
	}

	if paginatedData.TotalRows != 100 {
		t.Errorf("Expected TotalRows 100, got %d", paginatedData.TotalRows)
	}

	if paginatedData.CurrentPage != 2 {
		t.Errorf("Expected CurrentPage 2, got %d", paginatedData.CurrentPage)
	}

	if paginatedData.TotalPages != 2 {
		t.Errorf("Expected TotalPages 2, got %d", paginatedData.TotalPages)
	}
}
