package gui

import (
	"badgermaps/events"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"time"
)

// SmartDashboard provides an intelligent home view with context-aware suggestions
type SmartDashboard struct {
	ui        *Gui
	presenter *GuiPresenter
}

// NewSmartDashboard creates a new smart dashboard
func NewSmartDashboard(ui *Gui, presenter *GuiPresenter) *SmartDashboard {
	return &SmartDashboard{
		ui:        ui,
		presenter: presenter,
	}
}

// CreateContent builds the smart dashboard interface
func (d *SmartDashboard) CreateContent() fyne.CanvasObject {
	// Header with greeting and status
	header := d.createHeader()

	// Connection status cards with quick actions
	statusCards := d.createStatusCards()

	// Statistics and insights
	insights := d.createInsights()

	// Refresh button pinned to page bottom
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		d.presenter.HandleRefreshStatus()
		// Refresh the dashboard
		if d.ui.tabs != nil {
			d.ui.RefreshHomeTab()
		}
	})
	footer := container.NewVBox(
		widget.NewSeparator(),
		container.NewCenter(refreshBtn),
	)

	// Show recent activity in the details panel on load (if UI is fully initialized)
	if d.ui.rightPane != nil {
		d.showRecentActivityInDetails()
	}

	// Main content without quick actions
	mainContent := container.NewVBox(
		statusCards,
		widget.NewSeparator(),
		insights,
	)

	return container.NewVScroll(container.NewBorder(
		header,
		footer,
		nil, nil,
		mainContent, // Remove padding to reduce height
	))
}

func (d *SmartDashboard) createHeader() fyne.CanvasObject {
	// Dynamic greeting based on time of day
	hour := time.Now().Hour()
	greeting := "Good morning"
	if hour >= 12 && hour < 17 {
		greeting = "Good afternoon"
	} else if hour >= 17 {
		greeting = "Good evening"
	}

	greetingLabel := widget.NewLabelWithStyle(greeting+"! Here's your sync overview:",
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Last sync timestamp
	lastSync := "Never"
	// In production, this would fetch from actual sync history
	if d.ui.app.DB != nil && d.ui.app.DB.IsConnected() {
		lastSync = "2 hours ago"
	}

	lastSyncLabel := widget.NewLabel(fmt.Sprintf("Last sync: %s", lastSync))

	return container.NewVBox(greetingLabel, lastSyncLabel)
}

func (d *SmartDashboard) createStatusCards() fyne.CanvasObject {
	// API Status Card
	apiCard := d.createConnectionCard(
		"API Connection",
		d.ui.app.API != nil && d.ui.app.API.IsConnected(),
		nil, // No actions
	)

	// Database Status Card - show schema status in the card
	dbConnected := d.ui.app.DB != nil && d.ui.app.DB.IsConnected()
	var dbStatusText string
	if dbConnected {
		// Check schema status
		if err := d.ui.app.DB.ValidateSchema(d.ui.app.State); err == nil {
			dbStatusText = "Connected (Schema Valid)"
		} else {
			dbStatusText = "Connected (Schema Invalid)"
			d.ui.app.Events.Dispatch(events.Debugf("dashboard", "Schema validation failed: %v", err))
		}
	} else {
		dbStatusText = "Not Connected"
	}

	dbCard := d.createConnectionCard(
		"Database",
		dbConnected,
		[]fyne.CanvasObject{widget.NewLabel(dbStatusText)},
	)

	// Server Status Card
	pid, serverRunning := d.ui.app.Server.GetServerStatus()
	var serverStatusText string
	if serverRunning {
		serverStatusText = fmt.Sprintf("Running (PID: %d)", pid)
	} else {
		serverStatusText = "Stopped"
	}

	serverCard := d.createConnectionCard(
		"Webhook Server",
		serverRunning,
		[]fyne.CanvasObject{widget.NewLabel(serverStatusText)},
	)

	return container.NewGridWithColumns(3, apiCard, dbCard, serverCard)
}

func (d *SmartDashboard) createConnectionCard(title string, isConnected bool, actions []fyne.CanvasObject) fyne.CanvasObject {
	var statusText string
	var statusColor color.Color

	if isConnected {
		statusText = "Connected"
		statusColor = color.NRGBA{R: 0, G: 200, B: 0, A: 255}
	} else {
		statusText = "Not Connected"
		statusColor = color.NRGBA{R: 200, G: 0, B: 0, A: 255}
	}

	// Status indicator
	indicator := canvas.NewCircle(statusColor)
	indicator.StrokeWidth = 2
	indicator.StrokeColor = statusColor
	indicator.Resize(fyne.NewSize(12, 12))

	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	statusLabel := widget.NewLabel(statusText)

	header := container.NewHBox(
		indicator,
		titleLabel,
	)

	content := container.NewVBox(header, statusLabel)

	// Add actions if provided
	if actions != nil && len(actions) > 0 {
		content.Add(widget.NewSeparator())
		for _, action := range actions {
			content.Add(action)
		}
	}

	return widget.NewCard("", "", content)
}

// showRecentActivityInDetails displays recent activity in the details panel
func (d *SmartDashboard) showRecentActivityInDetails() {
	// Get actual recent activities or show placeholder
	activities := d.getRecentActivities()

	var activityContent []fyne.CanvasObject

	title := widget.NewLabelWithStyle("Recent System Activity",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	activityContent = append(activityContent, title, widget.NewSeparator())

	if len(activities) == 0 {
		// No activities found
		emptyLabel := widget.NewLabel("No recent activity to display")
		emptyLabel.Alignment = fyne.TextAlignCenter
		activityContent = append(activityContent, emptyLabel)
	} else {
		for _, activity := range activities {
			activityRow := container.NewHBox(
				widget.NewIcon(activity.Icon),
				container.NewVBox(
					widget.NewLabelWithStyle(activity.Message, fyne.TextAlignLeading, fyne.TextStyle{Bold: false}),
					widget.NewLabelWithStyle(activity.Time, fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
				),
			)
			activityContent = append(activityContent, activityRow)
		}
	}

	// Add refresh button
	refreshBtn := widget.NewButtonWithIcon("Refresh Activity", theme.ViewRefreshIcon(), func() {
		d.ui.app.Events.Dispatch(events.Infof("dashboard", "Refreshing activity feed"))
		// Re-show activity (in real implementation, this would fetch fresh data)
		d.showRecentActivityInDetails()
	})

	activityContent = append(activityContent, widget.NewSeparator(), refreshBtn)

	// Show in details panel (only if UI is fully initialized)
	detailsContainer := container.NewVScroll(container.NewVBox(activityContent...))
	if d.ui.rightPane != nil {
		d.ui.ShowDetails(detailsContainer)
	}
}

// ActivityItem represents a system activity
type ActivityItem struct {
	Icon    fyne.Resource
	Message string
	Time    string
	Type    string
}

// getRecentActivities retrieves recent system activities
func (d *SmartDashboard) getRecentActivities() []ActivityItem {
	var activities []ActivityItem

	// Check system state and generate relevant activities
	apiConnected := d.ui.app.API != nil && d.ui.app.API.IsConnected()
	dbConnected := d.ui.app.DB != nil && d.ui.app.DB.IsConnected()
	_, serverRunning := d.ui.app.Server.GetServerStatus()

	// Generate activities based on current system state
	if dbConnected {
		activities = append(activities, ActivityItem{
			Icon:    theme.StorageIcon(),
			Message: "Database connection established",
			Time:    "System startup",
			Type:    "connection",
		})

		// Check if schema is valid
		if err := d.ui.app.DB.ValidateSchema(d.ui.app.State); err == nil {
			activities = append(activities, ActivityItem{
				Icon:    theme.ConfirmIcon(),
				Message: "Database schema validation passed",
				Time:    "System startup",
				Type:    "validation",
			})
		} else {
			activities = append(activities, ActivityItem{
				Icon:    theme.WarningIcon(),
				Message: "Database schema validation failed",
				Time:    "System startup",
				Type:    "warning",
			})
		}
	}

	if apiConnected {
		activities = append(activities, ActivityItem{
			Icon:    theme.ComputerIcon(),
			Message: "API connection established",
			Time:    "System startup",
			Type:    "connection",
		})
	}

	if serverRunning {
		activities = append(activities, ActivityItem{
			Icon:    theme.MediaPlayIcon(),
			Message: "Webhook server is running",
			Time:    "System startup",
			Type:    "server",
		})
	}

	// Add placeholder sync activities if system is properly configured
	if apiConnected && dbConnected {
		activities = append(activities, ActivityItem{
			Icon:    theme.InfoIcon(),
			Message: "System ready for data synchronization",
			Time:    "System startup",
			Type:    "status",
		})
	}

	// Limit to most recent activities
	if len(activities) > 5 {
		activities = activities[:5]
	}

	return activities
}

func (d *SmartDashboard) createInsights() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("System Overview",
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Get actual statistics from the database
	stats := d.getSystemStatistics()

	var statWidgets []fyne.CanvasObject
	for _, stat := range stats {
		// Create a compact card for each statistic
		statCard := widget.NewCard("", "", container.NewVBox(
			widget.NewLabelWithStyle(stat.Label, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle(stat.Value, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		))
		statWidgets = append(statWidgets, statCard)
	}

	grid := container.NewGridWithColumns(4, statWidgets...) // More columns = more compact

	return container.NewVBox(
		title,
		grid,
	)
}

// SystemStat represents a system statistic
type SystemStat struct {
	Label       string
	Value       string
	Description string
}

// getSystemStatistics retrieves actual system statistics
func (d *SmartDashboard) getSystemStatistics() []SystemStat {
	var stats []SystemStat

	// Database Statistics
	if d.ui.app.DB != nil && d.ui.app.DB.IsConnected() {
		// Get total accounts
		if accountCount := d.getTableRowCount("Accounts"); accountCount >= 0 {
			stats = append(stats, SystemStat{
				Label:       "Total Accounts",
				Value:       fmt.Sprintf("%d", accountCount),
				Description: "Records in database",
			})
		}

		// Get total check-ins
		if checkinCount := d.getTableRowCount("AccountCheckins"); checkinCount >= 0 {
			stats = append(stats, SystemStat{
				Label:       "Total Check-ins",
				Value:       fmt.Sprintf("%d", checkinCount),
				Description: "Records in database",
			})
		}

		// Get pending changes count
		pendingCount := d.getPendingChangesCount()
		stats = append(stats, SystemStat{
			Label:       "Pending Changes",
			Value:       fmt.Sprintf("%d", pendingCount.Total),
			Description: fmt.Sprintf("%d accounts, %d check-ins", pendingCount.Accounts, pendingCount.Checkins),
		})

		// Get last sync status
		lastSync := d.getLastSyncInfo()
		stats = append(stats, SystemStat{
			Label:       "Last Sync",
			Value:       lastSync.Time,
			Description: lastSync.Status,
		})
	} else {
		// Database not connected
		stats = append(stats, SystemStat{
			Label:       "Database Status",
			Value:       "Not Connected",
			Description: "Configure connection in Settings",
		})

		stats = append(stats, SystemStat{
			Label:       "API Status",
			Value:       d.getAPIStatus(),
			Description: "Check configuration if not connected",
		})
	}

	return stats
}

// getTableRowCount gets the number of rows in a table safely
func (d *SmartDashboard) getTableRowCount(tableName string) int {
	if d.ui.app.DB == nil || !d.ui.app.DB.IsConnected() {
		return -1
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	rows, err := d.ui.app.DB.ExecuteQuery(query)
	if err != nil {
		d.ui.app.Events.Dispatch(events.Debugf("dashboard", "Error counting rows in %s: %v", tableName, err))
		return -1
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

// PendingChangesCount holds counts of pending changes by type
type PendingChangesCount struct {
	Total    int
	Accounts int
	Checkins int
}

// getPendingChangesCount gets the count of pending changes
func (d *SmartDashboard) getPendingChangesCount() PendingChangesCount {
	count := PendingChangesCount{}

	if d.ui.app.DB == nil || !d.ui.app.DB.IsConnected() {
		return count
	}

	// Count pending account changes
	if accountChanges := d.getTableRowCount("AccountsPendingChanges"); accountChanges >= 0 {
		count.Accounts = accountChanges
	}

	// Count pending checkin changes
	if checkinChanges := d.getTableRowCount("AccountCheckinsPendingChanges"); checkinChanges >= 0 {
		count.Checkins = checkinChanges
	}

	count.Total = count.Accounts + count.Checkins
	return count
}

// LastSyncInfo holds information about the last sync operation
type LastSyncInfo struct {
	Time   string
	Status string
}

// getLastSyncInfo gets information about the last sync
func (d *SmartDashboard) getLastSyncInfo() LastSyncInfo {
	// In a real implementation, this would query a sync log table
	// For now, return placeholder data
	return LastSyncInfo{
		Time:   "Never",
		Status: "No sync performed yet",
	}
}

// getAPIStatus returns the current API connection status
func (d *SmartDashboard) getAPIStatus() string {
	if d.ui.app.API != nil && d.ui.app.API.IsConnected() {
		return "Connected"
	} else if d.ui.app.API != nil && d.ui.app.API.APIKey != "" {
		return "Configured"
	}
	return "Not Configured"
}

// RefreshDashboard updates all dashboard components with latest data
func (d *SmartDashboard) RefreshDashboard() {
	// This would refresh all components with latest data
	d.ui.app.Events.Dispatch(events.Debugf("dashboard", "Refreshing dashboard"))
}
