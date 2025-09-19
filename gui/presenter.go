package gui

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/app/pull"
	"badgermaps/app/push"
	"badgermaps/database"
	"badgermaps/events"
	"fmt"
	"gopkg.in/yaml.v2"
	"strconv"
)

// GuiPresenter handles the presentation logic for the GUI.
// It mediates between the View (the UI) and the Model (the application logic).
type GuiPresenter struct {
	app *app.App
	// view is an interface, allowing us to swap out the UI implementation or mock it for testing.
	view GuiView
}

// NewGuiPresenter creates a new presenter.
func NewGuiPresenter(a *app.App, v GuiView) *GuiPresenter {
	return &GuiPresenter{app: a, view: v}
}

// --- Pull Handlers ---

// HandlePullGroup initiates a full data pull for all data types.
func (p *GuiPresenter) HandlePullGroup() {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePullGroup called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Starting full data pull..."))
	p.view.ShowProgressBar("Running Full Pull...")
	p.view.SetProgress(0)

	go func() {
		defer p.view.HideProgressBar()

		totalMajorSteps := 4.0
		majorStepWeight := 1.0 / totalMajorSteps

		p.app.Events.Dispatch(events.Infof("presenter", "Pulling accounts..."))
		accountsCallback := func(current, total int) {
			progress := (float64(current) / float64(total)) * majorStepWeight
			p.view.SetProgress(progress)
		}
		if err := pull.PullGroupAccounts(p.app, 0, accountsCallback); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "Error pulling accounts: %v", err))
			p.view.ShowToast("Error: The data pull failed.")
			return
		}
		p.view.SetProgress(majorStepWeight)

		p.app.Events.Dispatch(events.Infof("presenter", "Pulling checkins..."))
		checkinsCallback := func(current, total int) {
			progress := majorStepWeight + (float64(current)/float64(total))*majorStepWeight
			p.view.SetProgress(progress)
		}
		if err := pull.PullGroupCheckins(p.app, checkinsCallback); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "Error pulling checkins: %v", err))
			p.view.ShowToast("Error: The data pull failed.")
			return
		}
		p.view.SetProgress(2 * majorStepWeight)

		p.app.Events.Dispatch(events.Infof("presenter", "Pulling routes..."))
		routesCallback := func(current, total int) {
			progress := 2*majorStepWeight + (float64(current)/float64(total))*majorStepWeight
			p.view.SetProgress(progress)
		}
		if err := pull.PullGroupRoutes(p.app, routesCallback); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "Error pulling routes: %v", err))
			p.view.ShowToast("Error: The data pull failed.")
			return
		}
		p.view.SetProgress(3 * majorStepWeight)

		p.app.Events.Dispatch(events.Infof("presenter", "Pulling user profile..."))
		profileCallback := func(current, total int) {
			progress := 3*majorStepWeight + (float64(current)/float64(total))*majorStepWeight
			p.view.SetProgress(progress)
		}
		if err := pull.PullProfile(p.app, profileCallback); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "Error pulling user profile: %v", err))
			p.view.ShowToast("Error: The data pull failed.")
			return
		}
		p.view.SetProgress(4 * majorStepWeight)

		p.app.Events.Dispatch(events.Infof("presenter", "Finished pulling all data."))
		p.view.ShowToast("Success: Full data pull complete.")
	}()
}

// HandlePullAccount pulls a single account by its ID.
func (p *GuiPresenter) HandlePullAccount(idStr string) {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePullAccount called with id: %s", idStr))
	id, err := strconv.Atoi(idStr)
	if err != nil {
		p.app.Events.Dispatch(events.Errorf("presenter", "Invalid Account ID: '%s'", idStr))
		return
	}
	p.app.Events.Dispatch(events.Infof("presenter", "Starting pull for account ID: %d...", id))
	go func() {
		if err := pull.PullAccount(p.app, id); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
			p.view.ShowToast(fmt.Sprintf("Error: Failed to pull account %d.", id))
			return
		}
		p.view.ShowToast(fmt.Sprintf("Success: Pulled account %d.", id))
	}()
}

// HandlePullAccounts pulls all accounts.
func (p *GuiPresenter) HandlePullAccounts() {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePullAccounts called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Starting pull for all accounts..."))
	p.view.ShowProgressBar("Pulling Accounts...")
	p.view.SetProgress(0)

	go func() {
		defer p.view.HideProgressBar()
		callback := func(current, total int) {
			p.view.SetProgress(float64(current) / float64(total))
		}
		if err := pull.PullGroupAccounts(p.app, 0, callback); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
			p.view.ShowToast("Error: Failed to pull all accounts.")
			return
		}
		p.view.SetProgress(1)
		p.view.ShowToast("Success: Pulled all accounts.")
	}()
}

// HandlePullCheckin pulls a single check-in by its ID.
func (p *GuiPresenter) HandlePullCheckin(idStr string) {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePullCheckin called with id: %s", idStr))
	id, err := strconv.Atoi(idStr)
	if err != nil {
		p.app.Events.Dispatch(events.Errorf("presenter", "Invalid Check-in ID: '%s'", idStr))
		return
	}
	p.app.Events.Dispatch(events.Infof("presenter", "Starting pull for check-in ID: %d...", id))
	go func() {
		if err := pull.PullCheckin(p.app, id); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
			p.view.ShowToast(fmt.Sprintf("Error: Failed to pull check-in %d.", id))
			return
		}
		p.view.ShowToast(fmt.Sprintf("Success: Pulled check-in %d.", id))
	}()
}

// HandlePullCheckins pulls all check-ins.
func (p *GuiPresenter) HandlePullCheckins() {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePullCheckins called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Starting pull for all check-ins..."))
	p.view.ShowProgressBar("Pulling Check-ins...")
	p.view.SetProgress(0)
	go func() {
		defer p.view.HideProgressBar()
		callback := func(current, total int) {
			p.view.SetProgress(float64(current) / float64(total))
		}
		if err := pull.PullGroupCheckins(p.app, callback); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
			p.view.ShowToast("Error: Failed to pull all check-ins.")
			return
		}
		p.view.SetProgress(1)
		p.view.ShowToast("Success: Pulled all check-ins.")
	}()
}

// HandlePullRoute pulls a single route by its ID.
func (p *GuiPresenter) HandlePullRoute(idStr string) {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePullRoute called with id: %s", idStr))
	id, err := strconv.Atoi(idStr)
	if err != nil {
		p.app.Events.Dispatch(events.Errorf("presenter", "Invalid Route ID: '%s'", idStr))
		return
	}
	p.app.Events.Dispatch(events.Infof("presenter", "Starting pull for route ID: %d...", id))
	go func() {
		if err := pull.PullRoute(p.app, id); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
			p.view.ShowToast(fmt.Sprintf("Error: Failed to pull route %d.", id))
			return
		}
		p.view.ShowToast(fmt.Sprintf("Success: Pulled route %d.", id))
	}()
}

// HandlePullRoutes pulls all routes.
func (p *GuiPresenter) HandlePullRoutes() {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePullRoutes called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Starting pull for all routes..."))
	p.view.ShowProgressBar("Pulling Routes...")
	p.view.SetProgress(0)
	go func() {
		defer p.view.HideProgressBar()
		callback := func(current, total int) {
			p.view.SetProgress(float64(current) / float64(total))
		}
		if err := pull.PullGroupRoutes(p.app, callback); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
			p.view.ShowToast("Error: Failed to pull all routes.")
			return
		}
		p.view.SetProgress(1)
		p.view.ShowToast("Success: Pulled all routes.")
	}()
}

// HandlePullProfile pulls the user profile.
func (p *GuiPresenter) HandlePullProfile() {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePullProfile called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Starting pull for user profile..."))
	p.view.ShowProgressBar("Pulling User Profile...")
	p.view.SetProgress(0)
	go func() {
		defer p.view.HideProgressBar()
		if p.app.DB == nil || p.app.DB.GetDB() == nil {
			if err := p.app.ReloadDB(); err != nil {
				p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: Failed to connect to database: %v", err))
				p.view.ShowToast("Error: Failed to connect to database.")
				return
			}
		}
		callback := func(current, total int) {
			p.view.SetProgress(float64(current) / float64(total))
		}
		if err := pull.PullProfile(p.app, callback); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
			p.view.ShowToast("Error: Failed to pull user profile.")
			return
		}
		p.view.SetProgress(1)
		p.view.ShowToast("Success: Pulled user profile.")
	}()
}

// --- Push Handlers ---

// HandlePushAccounts pushes pending account changes.
func (p *GuiPresenter) HandlePushAccounts() {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePushAccounts called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Starting push for account changes..."))
	go func() {
		if err := push.RunPushAccounts(p.app); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
			p.view.ShowToast("Error: Failed to push account changes.")
			return
		}
		p.view.ShowToast("Success: Account changes pushed.")
		p.view.RefreshPushTab()
	}()
}

// HandlePushCheckins pushes pending check-in changes.
func (p *GuiPresenter) HandlePushCheckins() {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePushCheckins called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Starting push for check-in changes..."))
	go func() {
		if err := push.RunPushCheckins(p.app); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
			p.view.ShowToast("Error: Failed to push check-in changes.")
			return
		}
		p.view.ShowToast("Success: Check-in changes pushed.")
		p.view.RefreshPushTab()
	}()
}

// HandlePushAll pushes all pending changes.
func (p *GuiPresenter) HandlePushAll() {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandlePushAll called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Starting push for all changes..."))
	go func() {
		if err := push.RunPushAccounts(p.app); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR during account push: %v", err))
		}
		if err := push.RunPushCheckins(p.app); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "ERROR during check-in push: %v", err))
		}
		p.view.ShowToast("Success: All pending changes pushed.")
		p.view.RefreshPushTab()
	}()
}

// --- Config Handlers ---

// HandleSaveConfig saves the application configuration.
func (p *GuiPresenter) HandleSaveConfig(
	apiKey, baseURL, dbType, dbPath, dbHost, dbPortStr, dbUser, dbPass, dbName,
	serverHost, serverPortStr string, tlsEnabled bool, tlsCert, tlsKey string,
	verbose bool,
) {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandleSaveConfig called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Saving configuration..."))

	// Update API config in memory
	p.app.Config.API.APIKey = apiKey
	p.app.Config.API.BaseURL = baseURL

	// Update Server config in memory
	p.app.State.ServerHost = serverHost
	serverPort, _ := strconv.Atoi(serverPortStr)
	p.app.State.ServerPort = serverPort
	p.app.State.TLSEnabled = tlsEnabled
	p.app.State.TLSCert = tlsCert
	p.app.State.TLSKey = tlsKey
	p.app.State.Verbose = verbose

	port, _ := strconv.Atoi(dbPortStr)

	// Clear old DB config values
	p.app.Config.DB = database.DBConfig{}

	p.app.Config.DB.Type = dbType
	switch dbType {
	case "sqlite3":
		p.app.Config.DB.Path = dbPath
	case "postgres":
		p.app.Config.DB.Host = dbHost
		p.app.Config.DB.Port = port
		p.app.Config.DB.Username = dbUser
		p.app.Config.DB.Password = dbPass
		p.app.Config.DB.Database = dbName
		p.app.Config.DB.SSLMode = "disable"
	case "mssql":
		p.app.Config.DB.Host = dbHost
		p.app.Config.DB.Port = port
		p.app.Config.DB.Username = dbUser
		p.app.Config.DB.Password = dbPass
		p.app.Config.DB.Database = dbName
	}

	// Write the accumulated viper config to file
	if err := p.app.SaveConfig(); err != nil {
		p.app.Events.Dispatch(events.Errorf("presenter", "ERROR saving config file: %v", err))
		p.view.ShowToast("Error: Failed to save configuration.")
		return
	}

	// Reload the application with the new config
	if err := p.app.LoadConfig(); err != nil {
		p.app.Events.Dispatch(events.Errorf("presenter", "ERROR reloading config: %v", err))
		p.view.ShowToast("Error: Failed to reload new configuration.")
		return
	}
	if err := p.app.ReloadDB(); err != nil {
		p.app.Events.Dispatch(events.Errorf("presenter", "ERROR reloading database: %v", err))
		p.view.ShowToast("Error: Failed to reload database.")
	}

	p.view.ShowToast("Success: Configuration saved successfully.")
	p.app.Events.Dispatch(events.Event{Type: events.ConnectionStatusChanged})
	p.view.RefreshAllTabs()
}

// HandleTestAPIConnection tests the API connection.
func (p *GuiPresenter) HandleTestAPIConnection(apiKey, baseURL string) {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandleTestAPIConnection called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Testing API connection..."))

	go func() {
		// Create a temporary API client for testing
		apiClient := api.NewAPIClient(&api.APIConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
		})

		if !apiClient.IsConnected() {
			p.app.Events.Dispatch(events.Errorf("presenter", "API connection failed"))
			p.app.API.SetConnected(false)
			p.app.Events.Dispatch(events.Event{Type: events.ConnectionStatusChanged})
			return
		}

		p.app.Events.Dispatch(events.Infof("presenter", "API connection successful!"))
		p.app.API.SetConnected(true)
		p.app.Events.Dispatch(events.Event{Type: events.ConnectionStatusChanged})
	}()
}

// HandleTestDBConnection tests the database connection.
func (p *GuiPresenter) HandleTestDBConnection(dbType, dbPath, dbHost, dbPortStr, dbUser, dbPass, dbName string) {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandleTestDBConnection called"))
	p.app.Events.Dispatch(events.Infof("presenter", "Testing connection for %s...", dbType))

	go func() {
		port, _ := strconv.Atoi(dbPortStr)

		// Create a temporary DB object for testing
		var db database.DB
		switch dbType {
		case "sqlite3":
			db = &database.SQLiteConfig{Path: dbPath}
		case "postgres":
			db = &database.PostgreSQLConfig{
				Host: dbHost, Port: port, Username: dbUser, Password: dbPass, Database: dbName, SSLMode: "disable",
			}
		case "mssql":
			db = &database.MSSQLConfig{
				Host: dbHost, Port: port, Username: dbUser, Password: dbPass, Database: dbName,
			}
		default:
			p.app.Events.Dispatch(events.Errorf("presenter", "Unknown database type for testing: %s", dbType))
			p.app.DB.SetConnected(false)
			p.app.Events.Dispatch(events.Event{Type: events.ConnectionStatusChanged})
			return
		}

		if err := db.Connect(); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "Failed to create connection: %v", err))
			p.app.DB.SetConnected(false)
			p.app.Events.Dispatch(events.Event{Type: events.ConnectionStatusChanged})
			return
		}
		defer db.Close()

		if err := db.TestConnection(); err != nil {
			p.app.Events.Dispatch(events.Errorf("presenter", "Connection failed: %v", err))
			p.app.DB.SetConnected(false)
			p.app.Events.Dispatch(events.Event{Type: events.ConnectionStatusChanged})
			return
		}

		p.app.Events.Dispatch(events.Infof("presenter", "Connection successful!"))
		p.app.DB.SetConnected(true)
		p.app.Events.Dispatch(events.Event{Type: events.ConnectionStatusChanged})
	}()
}

// HandleSchemaEnforcement initializes or re-initializes the database schema.
func (p *GuiPresenter) HandleSchemaEnforcement() {
	p.app.Events.Dispatch(events.Debugf("presenter", "HandleSchemaEnforcement called"))
	if err := p.app.DB.ValidateSchema(p.app.State); err == nil {
		// Schema exists, confirm re-initialization
		p.view.ShowConfirmDialog("Re-initialize Schema?", "This will delete all existing data. Are you sure?", func(ok bool) {
			if !ok {
				return
			}
			p.app.Events.Dispatch(events.Infof("presenter", "Re-initializing database schema..."))
			go func() {
				if err := p.app.DB.EnforceSchema(p.app.State); err != nil {
					p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
					p.view.ShowToast("Error: Failed to re-initialize schema.")
					return
				}
				p.app.Events.Dispatch(events.Infof("presenter", "Schema re-initialized successfully."))
				p.view.ShowToast("Success: Schema re-initialized.")
				p.view.RefreshConfigTab()
				p.view.RefreshHomeTab()
			}()
		})
	} else {
		// Schema doesn't exist, just initialize it
		p.app.Events.Dispatch(events.Infof("presenter", "Initializing database schema..."))
		go func() {
			if err := p.app.DB.EnforceSchema(p.app.State); err != nil {
				p.app.Events.Dispatch(events.Errorf("presenter", "ERROR: %v", err))
				p.view.ShowToast("Error: Failed to initialize schema.")
				return
			}
			p.app.Events.Dispatch(events.Infof("presenter", "Schema initialized successfully."))
			p.view.ShowToast("Success: Schema initialized.")
			p.view.RefreshConfigTab()
			p.view.RefreshHomeTab()
		}()
	}
}

// HandleViewConfig marshals the current config to YAML and shows it in the details view.
func (p *GuiPresenter) HandleViewConfig() {
	configData, err := yaml.Marshal(p.app.Config)
	if err != nil {
		p.app.Events.Dispatch(events.Errorf("presenter", "Error marshaling config: %v", err))
		return
	}
	// This is a bit of a violation of MVP, as the presenter is creating a view component.
	// However, it's a simple case and avoids adding another method to the view interface.
	// For a stricter MVP, the view would have a method like `ShowConfig(config string)`.
	detailsLabel := NewWrappingLabel(string(configData))
	p.view.ShowDetails(detailsLabel)
}

// --- Server Handlers ---

// HandleStartServer starts the webhook server.
func (p *GuiPresenter) HandleStartServer() {
	if err := p.app.Server.StartServer(); err != nil {
		p.app.Events.Dispatch(events.Errorf("presenter", "Error starting server: %v", err))
		p.view.ShowErrorDialog(err)
	}
	p.view.RefreshHomeTab()
}

// HandleStopServer stops the webhook server.
func (p *GuiPresenter) HandleStopServer() {
	if err := p.app.Server.StopServer(); err != nil {
		p.app.Events.Dispatch(events.Errorf("presenter", "Error stopping server: %v", err))
		p.view.ShowErrorDialog(err)
	}
	p.view.RefreshHomeTab()
}

// --- Status Handlers ---

// HandleRefreshStatus triggers a refresh of connection statuses.
func (p *GuiPresenter) HandleRefreshStatus() {
	p.app.Events.Dispatch(events.Infof("presenter", "Refreshing connection statuses..."))

	// Test API Connection
	if p.app.API != nil {
		go func() {
			if err := p.app.API.TestAPIConnection(); err != nil {
				p.app.Events.Dispatch(events.Errorf("presenter", "API connection test failed: %v", err))
			}
			p.app.Events.Dispatch(events.Event{Type: events.ConnectionStatusChanged})
		}()
	}

	// Test DB Connection
	if p.app.DB != nil {
		go func() {
			if err := p.app.DB.TestConnection(); err != nil {
				p.app.Events.Dispatch(events.Errorf("presenter", "DB connection test failed: %v", err))
			}
			p.app.Events.Dispatch(events.Event{Type: events.ConnectionStatusChanged})
		}()
	}

	p.app.Events.Dispatch(events.Infof("presenter", "Connection status refresh initiated."))
}
