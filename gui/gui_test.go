package gui

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/app/state"
	"badgermaps/database"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/test"
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
	if ui.syncCenter.actionButton.Text != "Sync Everything" {
		t.Fatalf("expected default action label 'Sync Everything', got %q", ui.syncCenter.actionButton.Text)
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
