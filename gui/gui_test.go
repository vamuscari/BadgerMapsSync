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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestOmniboxSearch(t *testing.T) {
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
	ui.window = test.NewWindow(ui.createContent())
	ui.tabs.SelectTabIndex(1) // Select the "Pull" tab

	// Find the search entry and button using the helper
	searchEntry := findWidget(ui.window.Canvas().Content(), func(o fyne.CanvasObject) bool {
		if e, ok := o.(*widget.Entry); ok {
			return e.PlaceHolder == "Search by name or ID…" || e.PlaceHolder == "Search by name or ID..."
		}
		return false
	})
	if searchEntry == nil {
		t.Fatal("Could not find search entry")
	}

	searchButton := findWidget(ui.window.Canvas().Content(), func(o fyne.CanvasObject) bool {
		if b, ok := o.(*widget.Button); ok {
			return b.Icon == theme.SearchIcon()
		}
		return false
	})
	if searchButton == nil {
		t.Fatal("Could not find search button")
	}

	// Simulate a search
	test.Type(searchEntry.(fyne.Focusable), "Test")
	test.Tap(searchButton.(fyne.Tappable))

	// Let the go routine in the presenter finish
	time.Sleep(500 * time.Millisecond)

	// Verify that the results are displayed in the details view
	resultsListObject := findWidget(ui.detailsView, func(o fyne.CanvasObject) bool {
		_, ok := o.(*widget.List)
		return ok
	})

	if resultsListObject == nil {
		t.Fatalf("Could not find results list in details view. Details view is a %T", ui.detailsView)
	}

	resultsList := resultsListObject.(*widget.List)
	if resultsList.Length() != 2 {
		t.Errorf("Expected 2 results, got %d", resultsList.Length())
	}
}
