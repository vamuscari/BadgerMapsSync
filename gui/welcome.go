package gui

import (
	"badgermaps/app"
	"badgermaps/app/action"
	"badgermaps/database"
	"badgermaps/events"
	"badgermaps/utils"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// WelcomeScreen creates the onboarding experience for first-time users
type WelcomeScreen struct {
	app        *app.App
	presenter  *GuiPresenter
	onComplete func()

	apiKey     string
	apiBaseURL string

	dbType string
	dbPath string
	dbHost string
	dbPort string
	dbUser string
	dbPass string
	dbName string

	serverHost       string
	serverPort       string
	serverTLSEnabled bool
}

// NewWelcomeScreen creates a new welcome screen
func NewWelcomeScreen(a *app.App, presenter *GuiPresenter, onComplete func()) *WelcomeScreen {
	w := &WelcomeScreen{
		app:        a,
		presenter:  presenter,
		onComplete: onComplete,
	}
	w.initializeWizardState()
	return w
}

func (w *WelcomeScreen) initializeWizardState() {
	if w.app == nil || w.app.Config == nil {
		return
	}

	w.apiKey = strings.TrimSpace(w.app.Config.API.APIKey)
	w.apiBaseURL = strings.TrimSpace(w.app.Config.API.BaseURL)
	if w.apiBaseURL == "" {
		w.apiBaseURL = "https://www.badgermapping.com/api"
	}
	if w.app.API != nil {
		if strings.TrimSpace(w.app.API.APIKey) != "" {
			w.apiKey = strings.TrimSpace(w.app.API.APIKey)
		}
		if strings.TrimSpace(w.app.API.BaseURL) != "" {
			w.apiBaseURL = strings.TrimSpace(w.app.API.BaseURL)
		}
	}

	dbConfig := w.app.Config.DB
	w.dbType = strings.TrimSpace(dbConfig.Type)
	if w.dbType == "" {
		w.dbType = "sqlite3"
	}
	w.dbPath = strings.TrimSpace(dbConfig.Path)
	w.dbHost = strings.TrimSpace(dbConfig.Host)
	if dbConfig.Port > 0 {
		w.dbPort = strconv.Itoa(dbConfig.Port)
	}
	w.dbUser = strings.TrimSpace(dbConfig.Username)
	w.dbPass = dbConfig.Password
	w.dbName = strings.TrimSpace(dbConfig.Database)

	if w.dbPath == "" {
		w.dbPath = "badgermaps.db"
	}
	if w.dbHost == "" {
		w.dbHost = "localhost"
	}
	if w.dbType == "postgres" && w.dbPort == "" {
		w.dbPort = "5432"
	}
	if w.dbType == "mssql" && w.dbPort == "" {
		w.dbPort = "1433"
	}
	if w.dbName == "" {
		w.dbName = "badgermaps"
	}

	serverConfig := w.app.Config.Server
	w.serverHost = strings.TrimSpace(serverConfig.Host)
	if w.serverHost == "" {
		w.serverHost = "0.0.0.0"
	}
	if serverConfig.Port > 0 {
		w.serverPort = strconv.Itoa(serverConfig.Port)
	} else {
		w.serverPort = "8080"
	}
	w.serverTLSEnabled = serverConfig.TLSEnabled
}

func (w *WelcomeScreen) applyWizardConfiguration() error {
	if w.app == nil || w.app.Config == nil {
		return fmt.Errorf("application state is unavailable")
	}

	trimmedAPIKey := strings.TrimSpace(w.apiKey)
	trimmedBaseURL := strings.TrimSpace(w.apiBaseURL)
	trimmedDBType := strings.TrimSpace(w.dbType)
	trimmedDBPath := strings.TrimSpace(w.dbPath)
	trimmedDBHost := strings.TrimSpace(w.dbHost)
	trimmedDBPort := strings.TrimSpace(w.dbPort)
	trimmedDBUser := strings.TrimSpace(w.dbUser)
	trimmedDBName := strings.TrimSpace(w.dbName)
	trimmedServerHost := strings.TrimSpace(w.serverHost)
	trimmedServerPort := strings.TrimSpace(w.serverPort)

	if err := validateAPIKey(trimmedAPIKey); err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}
	if trimmedBaseURL == "" {
		return fmt.Errorf("API base URL is required")
	}
	if _, err := url.ParseRequestURI(trimmedBaseURL); err != nil {
		return fmt.Errorf("invalid API base URL: %w", err)
	}

	if trimmedDBType == "" {
		return fmt.Errorf("database type is required")
	}
	switch trimmedDBType {
	case "sqlite3":
		if trimmedDBPath == "" {
			return fmt.Errorf("database file path is required")
		}
	case "postgres", "mssql":
		if err := validateDatabaseHost(trimmedDBHost); err != nil {
			return fmt.Errorf("invalid database host: %w", err)
		}
		if err := validateDatabasePort(trimmedDBPort); err != nil {
			return fmt.Errorf("invalid database port: %w", err)
		}
		if err := validateDatabaseName(trimmedDBName); err != nil {
			return fmt.Errorf("invalid database name: %w", err)
		}
	default:
		return fmt.Errorf("unsupported database type: %s", trimmedDBType)
	}

	if err := validateDatabaseHost(trimmedServerHost); err != nil {
		return fmt.Errorf("invalid server host: %w", err)
	}
	if err := validateDatabasePort(trimmedServerPort); err != nil {
		return fmt.Errorf("invalid server port: %w", err)
	}
	serverPort, _ := strconv.Atoi(trimmedServerPort)

	w.app.Config.API.APIKey = trimmedAPIKey
	w.app.Config.API.BaseURL = trimmedBaseURL

	w.app.Config.DB = appDefaultDBConfig(trimmedDBType, trimmedDBPath, trimmedDBHost, trimmedDBPort, trimmedDBUser, w.dbPass, trimmedDBName)

	w.app.Config.Server.Host = trimmedServerHost
	w.app.Config.Server.Port = serverPort
	w.app.Config.Server.TLSEnabled = w.serverTLSEnabled
	w.app.State.ServerHost = trimmedServerHost
	w.app.State.ServerPort = serverPort
	w.app.State.TLSEnabled = w.serverTLSEnabled

	if strings.TrimSpace(w.app.ConfigFile) == "" {
		if path, ok, err := w.app.GetConfigFilePath(); err == nil && ok && strings.TrimSpace(path) != "" {
			w.app.ConfigFile = path
		} else {
			w.app.ConfigFile = utils.GetConfigDirFile("config.yaml")
		}
	}

	if err := w.app.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save setup configuration: %w", err)
	}

	// Apply wizard settings to runtime objects without blocking UI on live connection attempts.
	if w.app.API != nil {
		w.app.API.APIKey = w.app.Config.API.APIKey
		w.app.API.BaseURL = w.app.Config.API.BaseURL
		w.app.API.SetConnected(false)
	}
	if w.app.DB != nil {
		w.app.DB.SetConnected(false)
	}
	w.app.ActionExecutor = action.NewExecutor(w.app.DB, w.app.API)

	w.app.Events.Dispatch(events.Event{Type: "connection.status.changed"})
	return nil
}

func appDefaultDBConfig(dbType, dbPath, dbHost, dbPort, dbUser, dbPass, dbName string) database.DBConfig {
	port, _ := strconv.Atoi(dbPort)
	cfg := database.DBConfig{
		Type: dbType,
	}
	switch dbType {
	case "sqlite3":
		cfg.Path = dbPath
	case "postgres":
		cfg.Host = dbHost
		cfg.Port = port
		cfg.Username = dbUser
		cfg.Password = dbPass
		cfg.Database = dbName
		cfg.SSLMode = "disable"
	case "mssql":
		cfg.Host = dbHost
		cfg.Port = port
		cfg.Username = dbUser
		cfg.Password = dbPass
		cfg.Database = dbName
	}
	return cfg
}

// CreateContent builds the welcome screen content
func (w *WelcomeScreen) CreateContent() fyne.CanvasObject {
	// Check if this is first-time setup
	needsSetup := w.checkNeedsSetup()

	if needsSetup {
		w.app.Events.Dispatch(events.Infof("welcome", "First-time setup required"))
		return w.createSetupWizard()
	}

	w.app.Events.Dispatch(events.Debugf("welcome", "Configuration exists, showing welcome overview"))
	return w.createWelcomeOverview()
}

func (w *WelcomeScreen) checkNeedsSetup() bool {
	// Check if basic configuration is missing
	apiConfigured := w.app.API != nil && w.app.API.APIKey != ""
	dbConfigured := w.app.DB != nil && w.app.DB.GetType() != ""

	return !apiConfigured || !dbConfigured
}

func (w *WelcomeScreen) createWelcomeOverview() fyne.CanvasObject {
	// Title with app icon
	title := widget.NewLabelWithStyle("Welcome to BadgerMaps Sync",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	subtitle := widget.NewLabel("Your data synchronization hub for BadgerMaps")
	subtitle.Alignment = fyne.TextAlignCenter

	// Quick status overview
	statusCards := w.createQuickStatusCards()

	// Quick action buttons
	quickActions := w.createQuickActions()

	// Recent activity
	recentActivity := w.createRecentActivity()

	// Continue button
	continueBtn := widget.NewButtonWithIcon("Continue to Dashboard",
		theme.NavigateNextIcon(), w.onComplete)
	continueBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		container.NewPadded(container.NewVBox(
			title,
			subtitle,
		)),
		widget.NewSeparator(),
		container.NewPadded(statusCards),
		widget.NewSeparator(),
		container.NewPadded(quickActions),
		widget.NewSeparator(),
		container.NewPadded(recentActivity),
		layout.NewSpacer(),
		container.NewPadded(container.NewCenter(continueBtn)),
	)

	return container.NewScroll(content)
}

func (w *WelcomeScreen) createSetupWizard() fyne.CanvasObject {
	currentStep := 0
	steps := []string{"Welcome", "API Configuration", "Database Setup", "Server Settings", "Complete"}

	// Progress indicator
	progressBar := widget.NewProgressBar()
	progressLabel := widget.NewLabel(steps[currentStep])
	progressLabel.Alignment = fyne.TextAlignCenter

	// Content container that will hold current step
	contentContainer := container.NewMax()

	// Navigation buttons
	var prevBtn, nextBtn *widget.Button

	updateStep := func(step int) {
		if step < 0 || step >= len(steps) {
			return
		}
		currentStep = step
		progressBar.SetValue(float64(step) / float64(len(steps)-1))
		progressLabel.SetText(steps[step])

		// Update button states
		if step == 0 {
			prevBtn.Disable()
		} else {
			prevBtn.Enable()
		}
		if step == len(steps)-1 {
			nextBtn.SetText("Finish")
			nextBtn.SetIcon(theme.ConfirmIcon())
		} else {
			nextBtn.SetText("Next")
			nextBtn.SetIcon(theme.NavigateNextIcon())
		}
		prevBtn.Refresh()
		nextBtn.Refresh()

		// Update content
		var content fyne.CanvasObject
		switch step {
		case 0:
			content = w.createWelcomeStep()
		case 1:
			content = w.createAPIStep()
		case 2:
			content = w.createDatabaseStep()
		case 3:
			content = w.createServerStep()
		case 4:
			content = w.createCompleteStep()
		}
		contentContainer.Objects = []fyne.CanvasObject{content}
		contentContainer.Refresh()
	}

	prevBtn = widget.NewButtonWithIcon("Previous", theme.NavigateBackIcon(), func() {
		updateStep(currentStep - 1)
	})

	nextBtn = widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() {
		if currentStep == len(steps)-1 {
			if err := w.applyWizardConfiguration(); err != nil {
				w.app.Events.Dispatch(events.Errorf("setup", "failed to finalize setup: %v", err))
				if w.presenter != nil && w.presenter.view != nil {
					w.presenter.view.ShowToast(fmt.Sprintf("Error: %v", err))
				}
				return
			}
			w.onComplete()
		} else {
			// Validate current step before proceeding
			if w.validateStep(currentStep) {
				updateStep(currentStep + 1)
			}
		}
	})
	nextBtn.Importance = widget.HighImportance

	skipBtn := widget.NewButtonWithIcon("Skip Setup", theme.MediaSkipNextIcon(), func() {
		w.onComplete()
	})

	// Initialize first step
	updateStep(0)

	// Main layout
	return container.NewBorder(
		container.NewVBox(
			progressBar,
			progressLabel,
			widget.NewSeparator(),
		),
		container.NewBorder(
			widget.NewSeparator(),
			nil, nil, nil,
			container.NewHBox(
				skipBtn,
				layout.NewSpacer(),
				prevBtn,
				nextBtn,
			),
		),
		nil, nil,
		contentContainer,
	)
}

func (w *WelcomeScreen) createWelcomeStep() fyne.CanvasObject {
	icon := canvas.NewImageFromResource(theme.HelpIcon())
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(64, 64))

	title := widget.NewLabelWithStyle("Welcome to BadgerMaps Sync Setup",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	description := widget.NewLabel("This wizard will help you configure BadgerMaps Sync for first-time use.\n\n" +
		"We'll set up:\n" +
		"• API connection to BadgerMaps\n" +
		"• Local database for data storage\n" +
		"• Server settings for webhooks\n\n" +
		"This should only take a few minutes.")
	description.Wrapping = fyne.TextWrapWord

	tips := widget.NewCard("Before You Begin", "", widget.NewLabel(
		"Make sure you have:\n"+
			"• Your BadgerMaps API key\n"+
			"• Database connection details (if not using SQLite)\n"+
			"• Network access for API calls"))

	return container.NewVBox(
		container.NewPadded(container.NewCenter(icon)),
		container.NewPadded(title),
		container.NewPadded(description),
		container.NewPadded(tips),
	)
}

func (w *WelcomeScreen) createAPIStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("API Configuration",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	description := widget.NewLabel("Connect to your BadgerMaps account using your API key.")
	description.Wrapping = fyne.TextWrapWord

	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("Enter your BadgerMaps API key")
	apiKeyEntry.SetText(w.apiKey)

	baseURLEntry := widget.NewEntry()
	baseURLEntry.SetPlaceHolder("https://www.badgermapping.com/api")
	if strings.TrimSpace(w.apiBaseURL) != "" {
		baseURLEntry.SetText(w.apiBaseURL)
	} else {
		baseURLEntry.SetText("https://www.badgermapping.com/api")
	}

	statusLabel := widget.NewLabel("")
	validationLabel := widget.NewLabel("")
	validationLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Add real-time validation
	apiKeyEntry.OnChanged = func(text string) {
		w.apiKey = strings.TrimSpace(text)
		if err := validateAPIKey(text); err != nil {
			validationLabel.SetText("Warning: " + err.Error())
		} else {
			validationLabel.SetText("Valid API key format")
		}
		validationLabel.Refresh()
	}

	baseURLEntry.OnChanged = func(text string) {
		w.apiBaseURL = strings.TrimSpace(text)
		if text != "" {
			if _, err := url.Parse(text); err != nil {
				validationLabel.SetText("Warning: Invalid URL format")
			} else if !strings.HasPrefix(text, "http://") && !strings.HasPrefix(text, "https://") {
				validationLabel.SetText("Warning: URL should start with http:// or https://")
			} else {
				validationLabel.SetText("Valid URL format")
			}
			validationLabel.Refresh()
		}
	}

	testButton := widget.NewButtonWithIcon("Test Connection", theme.ConfirmIcon(), func() {
		// Validate before testing
		if err := validateAPIKey(apiKeyEntry.Text); err != nil {
			statusLabel.SetText(fmt.Sprintf("Validation error: %v", err))
			statusLabel.Refresh()
			return
		}

		if baseURLEntry.Text != "" {
			if _, err := url.Parse(baseURLEntry.Text); err != nil {
				statusLabel.SetText(fmt.Sprintf("Invalid URL: %v", err))
				statusLabel.Refresh()
				return
			}
		}

		statusLabel.SetText("Testing connection...")
		statusLabel.Refresh()
		w.apiKey = strings.TrimSpace(apiKeyEntry.Text)
		w.apiBaseURL = strings.TrimSpace(baseURLEntry.Text)

		go func() {
			apiKey := strings.TrimSpace(apiKeyEntry.Text)
			baseURL := strings.TrimSpace(baseURLEntry.Text)

			// Log connection attempt without exposing credentials
			w.app.Events.Dispatch(events.Infof("setup", "Testing API connection to %s with key %s",
				baseURL, sanitizeCredentials(apiKey)))

			w.presenter.HandleTestAPIConnection(apiKey, baseURL)
			// UI updates must occur on the main thread
			fyne.Do(func() {
				statusLabel.SetText("Connection test complete.")
				statusLabel.Refresh()
			})
		}()
	})

	helpCard := widget.NewCard("How to get your API key", "", widget.NewLabel(
		"1. Log in to your BadgerMaps account\n"+
			"2. Go to Settings > API\n"+
			"3. Generate or copy your API key\n"+
			"4. Paste it above"))

	form := widget.NewForm(
		widget.NewFormItem("API Key", apiKeyEntry),
		widget.NewFormItem("Base URL", baseURLEntry),
	)

	return container.NewVBox(
		container.NewPadded(title),
		container.NewPadded(description),
		container.NewPadded(form),
		container.NewPadded(validationLabel),
		container.NewPadded(container.NewHBox(testButton, statusLabel)),
		container.NewPadded(helpCard),
	)
}

func (w *WelcomeScreen) createDatabaseStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Database Setup",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	description := widget.NewLabel("Choose how to store your synchronized data locally.")
	description.Wrapping = fyne.TextWrapWord

	// Database type selection with visual cards
	sqliteCard := w.createDBOptionCard("SQLite",
		"Simple file-based database. Perfect for single-user setups.",
		theme.DocumentIcon())

	postgresCard := w.createDBOptionCard("PostgreSQL",
		"Robust database server. Great for multi-user or production use.",
		theme.StorageIcon())

	mssqlCard := w.createDBOptionCard("MS SQL Server",
		"Microsoft SQL Server. Ideal for Windows enterprise environments.",
		theme.ComputerIcon())

	dbOptions := container.NewGridWithColumns(3, sqliteCard, postgresCard, mssqlCard)

	// Configuration form container
	configContainer := container.NewMax()

	updateConfig := func(selectedType string) {
		w.dbType = selectedType
		var content fyne.CanvasObject

		switch selectedType {
		case "sqlite3":
			content = w.createSQLiteConfig()
		case "postgres":
			content = w.createPostgresConfig()
		case "mssql":
			content = w.createMSSQLConfig()
		}

		configContainer.Objects = []fyne.CanvasObject{content}
		configContainer.Refresh()
	}

	// Set up card tap handlers
	sqliteCard.(*widget.Card).SetContent(
		container.NewVBox(
			sqliteCard.(*widget.Card).Content,
			widget.NewButtonWithIcon("Select", theme.ConfirmIcon(), func() {
				updateConfig("sqlite3")
			}),
		))

	postgresCard.(*widget.Card).SetContent(
		container.NewVBox(
			postgresCard.(*widget.Card).Content,
			widget.NewButtonWithIcon("Select", theme.ConfirmIcon(), func() {
				updateConfig("postgres")
			}),
		))

	mssqlCard.(*widget.Card).SetContent(
		container.NewVBox(
			mssqlCard.(*widget.Card).Content,
			widget.NewButtonWithIcon("Select", theme.ConfirmIcon(), func() {
				updateConfig("mssql")
			}),
		))

	// Initialize with currently selected type
	initialType := w.dbType
	switch initialType {
	case "sqlite3", "postgres", "mssql":
	default:
		initialType = "sqlite3"
	}
	updateConfig(initialType)

	return container.NewVBox(
		container.NewPadded(title),
		container.NewPadded(description),
		container.NewPadded(dbOptions),
		widget.NewSeparator(),
		container.NewPadded(configContainer),
	)
}

func (w *WelcomeScreen) createDBOptionCard(title, description string, icon fyne.Resource) fyne.CanvasObject {
	iconWidget := widget.NewIcon(icon)
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	descLabel := widget.NewLabel(description)
	descLabel.Wrapping = fyne.TextWrapWord

	return widget.NewCard("", "", container.NewVBox(
		container.NewCenter(iconWidget),
		titleLabel,
		descLabel,
	))
}

func (w *WelcomeScreen) createSQLiteConfig() fyne.CanvasObject {
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("badgermaps.db")
	if strings.TrimSpace(w.dbPath) != "" {
		pathEntry.SetText(w.dbPath)
	} else {
		pathEntry.SetText("badgermaps.db")
	}
	pathEntry.OnChanged = func(text string) {
		w.dbPath = strings.TrimSpace(text)
	}

	form := widget.NewForm(
		widget.NewFormItem("Database File", pathEntry),
	)

	info := widget.NewCard("", "", widget.NewLabel(
		"SQLite will create a local database file.\n"+
			"No additional setup required!"))

	return container.NewVBox(
		widget.NewLabelWithStyle("SQLite Configuration",
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		form,
		info,
	)
}

func (w *WelcomeScreen) createPostgresConfig() fyne.CanvasObject {
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("localhost")
	if strings.TrimSpace(w.dbHost) != "" {
		hostEntry.SetText(w.dbHost)
	} else {
		hostEntry.SetText("localhost")
	}

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("5432")
	if strings.TrimSpace(w.dbPort) != "" {
		portEntry.SetText(w.dbPort)
	} else {
		portEntry.SetText("5432")
	}

	userEntry := widget.NewEntry()
	userEntry.SetPlaceHolder("postgres")
	userEntry.SetText(w.dbUser)

	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("password")
	passEntry.SetText(w.dbPass)

	dbEntry := widget.NewEntry()
	dbEntry.SetPlaceHolder("badgermaps")
	if strings.TrimSpace(w.dbName) != "" {
		dbEntry.SetText(w.dbName)
	} else {
		dbEntry.SetText("badgermaps")
	}

	validationLabel := widget.NewLabel("")
	validationLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Add validation
	hostEntry.OnChanged = func(text string) {
		w.dbHost = strings.TrimSpace(text)
		if err := validateDatabaseHost(text); err != nil {
			validationLabel.SetText("Warning: " + err.Error())
		} else {
			validationLabel.SetText("")
		}
		validationLabel.Refresh()
	}

	portEntry.OnChanged = func(text string) {
		w.dbPort = strings.TrimSpace(text)
		if err := validateDatabasePort(text); err != nil {
			validationLabel.SetText("Warning: " + err.Error())
		} else {
			validationLabel.SetText("")
		}
		validationLabel.Refresh()
	}

	dbEntry.OnChanged = func(text string) {
		w.dbName = strings.TrimSpace(text)
		if err := validateDatabaseName(text); err != nil {
			validationLabel.SetText("Warning: " + err.Error())
		} else {
			validationLabel.SetText("")
		}
		validationLabel.Refresh()
	}
	userEntry.OnChanged = func(text string) {
		w.dbUser = strings.TrimSpace(text)
	}
	passEntry.OnChanged = func(text string) {
		w.dbPass = text
	}

	form := widget.NewForm(
		widget.NewFormItem("Host", hostEntry),
		widget.NewFormItem("Port", portEntry),
		widget.NewFormItem("Username", userEntry),
		widget.NewFormItem("Password", passEntry),
		widget.NewFormItem("Database", dbEntry),
	)

	return container.NewVBox(
		widget.NewLabelWithStyle("PostgreSQL Configuration",
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		form,
		validationLabel,
	)
}

func (w *WelcomeScreen) createMSSQLConfig() fyne.CanvasObject {
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("localhost")
	if strings.TrimSpace(w.dbHost) != "" {
		hostEntry.SetText(w.dbHost)
	} else {
		hostEntry.SetText("localhost")
	}

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("1433")
	if strings.TrimSpace(w.dbPort) != "" {
		portEntry.SetText(w.dbPort)
	} else {
		portEntry.SetText("1433")
	}

	userEntry := widget.NewEntry()
	userEntry.SetPlaceHolder("sa")
	userEntry.SetText(w.dbUser)

	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("password")
	passEntry.SetText(w.dbPass)

	dbEntry := widget.NewEntry()
	dbEntry.SetPlaceHolder("badgermaps")
	if strings.TrimSpace(w.dbName) != "" {
		dbEntry.SetText(w.dbName)
	} else {
		dbEntry.SetText("badgermaps")
	}

	validationLabel := widget.NewLabel("")
	validationLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Add validation
	hostEntry.OnChanged = func(text string) {
		w.dbHost = strings.TrimSpace(text)
		if err := validateDatabaseHost(text); err != nil {
			validationLabel.SetText("Warning: " + err.Error())
		} else {
			validationLabel.SetText("")
		}
		validationLabel.Refresh()
	}

	portEntry.OnChanged = func(text string) {
		w.dbPort = strings.TrimSpace(text)
		if err := validateDatabasePort(text); err != nil {
			validationLabel.SetText("Warning: " + err.Error())
		} else {
			validationLabel.SetText("")
		}
		validationLabel.Refresh()
	}

	dbEntry.OnChanged = func(text string) {
		w.dbName = strings.TrimSpace(text)
		if err := validateDatabaseName(text); err != nil {
			validationLabel.SetText("Warning: " + err.Error())
		} else {
			validationLabel.SetText("")
		}
		validationLabel.Refresh()
	}
	userEntry.OnChanged = func(text string) {
		w.dbUser = strings.TrimSpace(text)
	}
	passEntry.OnChanged = func(text string) {
		w.dbPass = text
	}

	form := widget.NewForm(
		widget.NewFormItem("Host", hostEntry),
		widget.NewFormItem("Port", portEntry),
		widget.NewFormItem("Username", userEntry),
		widget.NewFormItem("Password", passEntry),
		widget.NewFormItem("Database", dbEntry),
	)

	return container.NewVBox(
		widget.NewLabelWithStyle("MS SQL Server Configuration",
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		form,
		validationLabel,
	)
}

func (w *WelcomeScreen) createServerStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Server Settings",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	description := widget.NewLabel("Configure the webhook server for receiving real-time updates.")
	description.Wrapping = fyne.TextWrapWord

	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("0.0.0.0")
	if strings.TrimSpace(w.serverHost) != "" {
		hostEntry.SetText(w.serverHost)
	} else {
		hostEntry.SetText("0.0.0.0")
	}

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("8080")
	if strings.TrimSpace(w.serverPort) != "" {
		portEntry.SetText(w.serverPort)
	} else {
		portEntry.SetText("8080")
	}

	tlsCheck := widget.NewCheck("Enable TLS/HTTPS", func(checked bool) {
		w.serverTLSEnabled = checked
	})
	tlsCheck.SetChecked(w.serverTLSEnabled)
	hostEntry.OnChanged = func(text string) {
		w.serverHost = strings.TrimSpace(text)
	}
	portEntry.OnChanged = func(text string) {
		w.serverPort = strings.TrimSpace(text)
	}

	form := widget.NewForm(
		widget.NewFormItem("Host", hostEntry),
		widget.NewFormItem("Port", portEntry),
		widget.NewFormItem("Security", tlsCheck),
	)

	info := widget.NewCard("", "", widget.NewLabel(
		"The webhook server allows BadgerMaps to send real-time updates.\n"+
			"This is optional but recommended for keeping data in sync."))

	return container.NewVBox(
		container.NewPadded(title),
		container.NewPadded(description),
		container.NewPadded(form),
		container.NewPadded(info),
	)
}

func (w *WelcomeScreen) createCompleteStep() fyne.CanvasObject {
	icon := widget.NewIcon(theme.ConfirmIcon())

	title := widget.NewLabelWithStyle("Setup Complete!",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	message := widget.NewLabel("BadgerMaps Sync is now configured and ready to use.\n\n" +
		"You can now:\n" +
		"• Pull data from BadgerMaps\n" +
		"• Push changes back to BadgerMaps\n" +
		"• Set up automated workflows\n" +
		"• Explore your synchronized data")
	message.Wrapping = fyne.TextWrapWord
	message.Alignment = fyne.TextAlignCenter

	// Summary of configuration
	summary := w.createConfigSummary()

	return container.NewVBox(
		container.NewPadded(container.NewCenter(icon)),
		container.NewPadded(title),
		container.NewPadded(message),
		container.NewPadded(summary),
	)
}

func (w *WelcomeScreen) createConfigSummary() fyne.CanvasObject {
	var items []fyne.CanvasObject

	// API Status with nil checks
	apiStatus := "Not configured"
	apiColor := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	if w.app != nil && w.app.API != nil {
		if w.app.API.IsConnected() {
			apiStatus = "Connected"
			apiColor = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
		} else {
			apiStatus = "Configured but not connected"
			apiColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
		}
	}

	apiLabel := canvas.NewText(apiStatus, apiColor)
	items = append(items, container.NewHBox(
		widget.NewLabelWithStyle("API:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		apiLabel,
	))

	// Database Status with nil checks
	dbStatus := "Not configured"
	dbColor := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	if w.app != nil && w.app.DB != nil {
		if w.app.DB.IsConnected() {
			dbStatus = fmt.Sprintf("Connected (%s)", w.app.DB.GetType())
			dbColor = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
		} else {
			dbStatus = fmt.Sprintf("Configured (%s) but not connected", w.app.DB.GetType())
			dbColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
		}
	}

	dbLabel := canvas.NewText(dbStatus, dbColor)
	items = append(items, container.NewHBox(
		widget.NewLabelWithStyle("Database:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dbLabel,
	))

	// Server Status with nil checks
	serverStatus := "Not running"
	serverColor := color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	if w.app != nil && w.app.Server != nil {
		if pid, running := w.app.Server.GetServerStatus(); running {
			serverStatus = fmt.Sprintf("Running (PID: %d)", pid)
			serverColor = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
		}
	}

	serverLabel := canvas.NewText(serverStatus, serverColor)
	items = append(items, container.NewHBox(
		widget.NewLabelWithStyle("Server:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		serverLabel,
	))

	return widget.NewCard("Configuration Summary", "", container.NewVBox(items...))
}

func (w *WelcomeScreen) validateStep(step int) bool {
	// Add validation logic for each step
	switch step {
	case 1: // API Configuration
		trimmedAPIKey := strings.TrimSpace(w.apiKey)
		trimmedBaseURL := strings.TrimSpace(w.apiBaseURL)
		if err := validateAPIKey(trimmedAPIKey); err != nil {
			w.app.Events.Dispatch(events.Warningf("setup", "Invalid API key: %v", err))
			return false
		}
		if trimmedBaseURL == "" {
			w.app.Events.Dispatch(events.Warningf("setup", "API base URL is required"))
			return false
		}
		if _, err := url.ParseRequestURI(trimmedBaseURL); err != nil {
			w.app.Events.Dispatch(events.Warningf("setup", "Invalid API base URL: %v", err))
			return false
		}
	case 2: // Database Setup
		switch strings.TrimSpace(w.dbType) {
		case "sqlite3":
			if strings.TrimSpace(w.dbPath) == "" {
				w.app.Events.Dispatch(events.Warningf("setup", "Database file path is required"))
				return false
			}
		case "postgres", "mssql":
			if err := validateDatabaseHost(strings.TrimSpace(w.dbHost)); err != nil {
				w.app.Events.Dispatch(events.Warningf("setup", "Invalid database host: %v", err))
				return false
			}
			if err := validateDatabasePort(strings.TrimSpace(w.dbPort)); err != nil {
				w.app.Events.Dispatch(events.Warningf("setup", "Invalid database port: %v", err))
				return false
			}
			if err := validateDatabaseName(strings.TrimSpace(w.dbName)); err != nil {
				w.app.Events.Dispatch(events.Warningf("setup", "Invalid database name: %v", err))
				return false
			}
		default:
			w.app.Events.Dispatch(events.Warningf("setup", "Database configuration is required"))
			return false
		}
	case 3: // Server Settings
		if err := validateDatabaseHost(strings.TrimSpace(w.serverHost)); err != nil {
			w.app.Events.Dispatch(events.Warningf("setup", "Invalid server host: %v", err))
			return false
		}
		if err := validateDatabasePort(strings.TrimSpace(w.serverPort)); err != nil {
			w.app.Events.Dispatch(events.Warningf("setup", "Invalid server port: %v", err))
			return false
		}
	}
	return true
}

// validateAPIKey validates the format of an API key
func validateAPIKey(key string) error {
	if key == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	if len(key) < 10 {
		return fmt.Errorf("API key appears too short")
	}
	if len(key) > 1000 {
		return fmt.Errorf("API key appears too long")
	}
	// Check for suspicious characters that might indicate SQL injection or XSS
	if strings.ContainsAny(key, "';\"<>&") {
		return fmt.Errorf("API key contains invalid characters")
	}
	// Check for only printable ASCII characters
	for _, r := range key {
		if r < 32 || r > 126 {
			return fmt.Errorf("API key contains non-printable characters")
		}
	}
	return nil
}

// sanitizeCredentials removes sensitive data from logs
func sanitizeCredentials(text string) string {
	if len(text) <= 8 {
		return "***"
	}
	return text[:4] + "***" + text[len(text)-4:]
}

// validateDatabaseHost validates database host input
func validateDatabaseHost(host string) error {
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	// Basic validation for SQL injection attempts
	if strings.ContainsAny(host, "';\"") {
		return fmt.Errorf("host contains invalid characters")
	}
	// Check for valid hostname or IP pattern
	hostnameRegex := regexp.MustCompile(`^([a-zA-Z0-9\-\.]+|localhost|\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})$`)
	if !hostnameRegex.MatchString(host) {
		return fmt.Errorf("invalid hostname or IP address")
	}
	return nil
}

// validateDatabasePort validates port input
func validateDatabasePort(portStr string) error {
	if portStr == "" {
		return fmt.Errorf("port cannot be empty")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("port must be a number")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

// validateDatabaseName validates database name input
func validateDatabaseName(name string) error {
	if name == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	// Check for SQL injection attempts
	if strings.ContainsAny(name, "';\"") {
		return fmt.Errorf("database name contains invalid characters")
	}
	// Check for valid database name pattern
	dbNameRegex := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	if !dbNameRegex.MatchString(name) {
		return fmt.Errorf("invalid database name format")
	}
	return nil
}

func (w *WelcomeScreen) createQuickStatusCards() fyne.CanvasObject {
	// Create status cards in a grid
	apiCard := w.createStatusCard("API", w.app.API != nil && w.app.API.IsConnected())
	dbCard := w.createStatusCard("Database", w.app.DB != nil && w.app.DB.IsConnected())
	serverCard := w.createStatusCard("Server", func() bool {
		_, running := w.app.Server.GetServerStatus()
		return running
	}())

	return container.NewGridWithColumns(3, apiCard, dbCard, serverCard)
}

func (w *WelcomeScreen) createStatusCard(title string, isConnected bool) fyne.CanvasObject {
	var icon fyne.Resource
	var statusText string
	var statusColor color.Color

	if isConnected {
		icon = theme.ConfirmIcon()
		statusText = "Connected"
		statusColor = color.NRGBA{R: 0, G: 200, B: 0, A: 255}
	} else {
		icon = theme.ErrorIcon()
		statusText = "Not Connected"
		statusColor = color.NRGBA{R: 200, G: 0, B: 0, A: 255}
	}

	iconWidget := widget.NewIcon(icon)
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	statusLabel := canvas.NewText(statusText, statusColor)
	statusLabel.Alignment = fyne.TextAlignCenter

	return widget.NewCard("", "", container.NewVBox(
		container.NewCenter(iconWidget),
		titleLabel,
		container.NewCenter(statusLabel),
	))
}

func (w *WelcomeScreen) createQuickActions() fyne.CanvasObject {
	pullBtn := widget.NewButtonWithIcon("Quick Pull", theme.DownloadIcon(), func() {
		w.presenter.HandlePullGroup()
	})

	pushBtn := widget.NewButtonWithIcon("Quick Push", theme.UploadIcon(), func() {
		w.presenter.HandlePushAll()
	})

	configBtn := widget.NewButtonWithIcon("Configure", theme.SettingsIcon(), func() {
		// Navigate to config tab
		w.onComplete()
	})

	return widget.NewCard("Quick Actions", "",
		container.NewGridWithColumns(3, pullBtn, pushBtn, configBtn))
}

func (w *WelcomeScreen) createRecentActivity() fyne.CanvasObject {
	// This would normally pull from actual activity logs
	activities := []string{
		"System initialized",
		"Ready for first sync",
		"No recent activity",
	}

	list := widget.NewList(
		func() int { return len(activities) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.InfoIcon()),
				widget.NewLabel("Activity"),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			container := o.(*fyne.Container)
			label := container.Objects[1].(*widget.Label)
			label.SetText(activities[i])
		},
	)

	return widget.NewCard("Recent Activity", "",
		container.NewMax(list))
}
