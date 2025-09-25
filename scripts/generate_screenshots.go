package main

import (
    "badgermaps/api"
    "badgermaps/app"
    "badgermaps/app/state"
    "badgermaps/database"
    "badgermaps/gui"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/test"
    "image/png"
    "io/ioutil"
    "os"
    "path/filepath"
    "strconv"
)

func must(err error) {
    if err != nil {
        panic(err)
    }
}

func ensureDir(path string) {
    _ = os.MkdirAll(path, 0o755)
}

func savePNG(path string, c fyne.Canvas) {
    shot := c.Capture()
    f, err := os.Create(path)
    must(err)
    defer f.Close()
    must(png.Encode(f, shot))
}

func seedDemoSQLite(db database.DB, demoSQLPath string) error {
    data, err := ioutil.ReadFile(demoSQLPath)
    if err != nil {
        return err
    }
    // Execute the entire file content; SQLite driver supports multi-statement exec
    _, err = db.GetDB().Exec(string(data))
    return err
}

func main() {
    shotsDir := filepath.Join("assets", "screenshots")
    ensureDir(shotsDir)

    // Prepare demo app + DB
    demoDBPath := filepath.Join(shotsDir, "demo.db")
    _ = os.Remove(demoDBPath)

    a := app.NewApp()

    // Configure API as connected so UI shows healthy state
    a.API = api.NewAPIClient(&api.APIConfig{APIKey: "demo-key", BaseURL: "https://example.invalid/api"})
    a.API.SetConnected(true)

    // Setup SQLite DB
    db, _ := database.NewDB(&database.DBConfig{Type: "sqlite3", Path: demoDBPath})
    must(db.Connect())
    defer db.Close()
    must(db.EnforceSchema(&state.State{}))
    must(seedDemoSQLite(db, filepath.Join("docs", "demo.sql")))
    db.SetConnected(true)
    a.DB = db

    // Build UI using test driver (headless)
    ui := gui.NewGuiForScreenshots(a, test.NewApp())
    // Force dark modern theme for consistent visuals
    ui.ApplyThemePreference(app.ThemePreferenceDark)
    // Scale up details pane for high-res output
    gui.SetDefaultRightPaneWidth(720)

    // Main window
    w := test.NewWindow(ui.CreateContent())
    // Retina-style scale (default 2.0; override with SCALE env var)
    scale := float32(2.0)
    if s := os.Getenv("SCALE"); s != "" {
        if f, err := strconv.ParseFloat(s, 32); err == nil && f > 0 {
            scale = float32(f)
        }
    }
    if c, ok := w.Canvas().(test.WindowlessCanvas); ok {
        c.SetScale(scale)
    }
    // Keep logical size near default; scale increases pixel density
    w.Resize(fyne.NewSize(1000, 600))

    // Home
    savePNG(filepath.Join(shotsDir, "home.png"), w.Canvas())

    // Configuration
    for _, item := range ui.Tabs().Items {
        if item.Text == "Configuration" {
            ui.Tabs().Select(item)
            break
        }
    }
    w.Resize(fyne.NewSize(1000, 600))
    savePNG(filepath.Join(shotsDir, "config.png"), w.Canvas())

    // Sync Center - Pull (top of tab)
    for _, item := range ui.Tabs().Items {
        if item.Text == "Sync Center" {
            ui.Tabs().Select(item)
            break
        }
    }
    w.Resize(fyne.NewSize(1000, 600))
    savePNG(filepath.Join(shotsDir, "sync-pull.png"), w.Canvas())

    // Sync Center - Push (capture push card alone for clarity)
    // Ensure content exists
    if ui.SyncCenter() != nil {
        sc := ui.SyncCenter()
        // Recreate content to ensure cards initialized
        _ = sc.CreateContent()
        w2 := ui.FyneApp().NewWindow("Push")
        w2.SetContent(sc.PushCard())
        if c2, ok := w2.Canvas().(test.WindowlessCanvas); ok { c2.SetScale(scale) }
        w2.Resize(fyne.NewSize(900, 500))
        savePNG(filepath.Join(shotsDir, "sync-push.png"), w2.Canvas())
        w2.Close()
    }

    // Explorer — Accounts
    for _, item := range ui.Tabs().Items {
        if item.Text == "Explorer" {
            ui.Tabs().Select(item)
            break
        }
    }
    if ui.ExplorerTableSelect() != nil {
        // Populate options synchronously
        if tables, err := a.DB.GetTables(); err == nil {
            ui.ExplorerTableSelect().Options = tables
            ui.ExplorerTableSelect().Refresh()
        }
        ui.ExplorerTableSelect().SetSelected("Accounts")
    }
    w.Resize(fyne.NewSize(1000, 600))
    savePNG(filepath.Join(shotsDir, "explorer-accounts.png"), w.Canvas())

    // Server
    for _, item := range ui.Tabs().Items {
        if item.Text == "Server" {
            ui.Tabs().Select(item)
            break
        }
    }
    w.Resize(fyne.NewSize(1000, 600))
    savePNG(filepath.Join(shotsDir, "server.png"), w.Canvas())

    w.Close()
}
