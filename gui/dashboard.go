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
	"image/color"
	"strings"
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
	refreshBtn.Importance = widget.HighImportance
	refreshBtnWrapper := container.New(
		layout.NewGridWrapLayout(fyne.NewSize(220, refreshBtn.MinSize().Height)),
		refreshBtn,
	)
	footer := container.NewVBox(
		widget.NewSeparator(),
		container.NewCenter(refreshBtnWrapper),
	)

	// Show recent activity in the details panel on load (if UI is fully initialized)
	if d.ui.rightPaneContent != nil {
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

	return container.NewVBox(greetingLabel)
}

func (d *SmartDashboard) createStatusCards() fyne.CanvasObject {
	// API Status Card
	apiConnected := d.ui.app.API != nil && d.ui.app.API.IsConnected()
	apiHeadline := "Not Connected"
	if apiConnected {
		apiHeadline = "Connected"
	}
	apiCard := d.createConnectionCard(
		"API Connection",
		apiHeadline,
		apiConnected,
		nil, // No additional details
	)

	// Database Status Card - show schema status in the card
	dbConnected := d.ui.app.DB != nil && d.ui.app.DB.IsConnected()
	var dbStatusDetail string
	if dbConnected {
		// Check schema status
		if err := d.ui.app.DB.ValidateSchema(d.ui.app.State); err == nil {
			dbStatusDetail = "Connected (Schema Valid)"
		} else {
			dbStatusDetail = "Connected (Schema Invalid)"
			d.ui.app.Events.Dispatch(events.Debugf("dashboard", "Schema validation failed: %v", err))
		}
	} else {
		dbStatusDetail = "Not Connected"
	}
	dbHeadline := "Not Connected"
	if dbConnected {
		dbHeadline = "Connected"
	}

	dbCard := d.createConnectionCard(
		"Database",
		dbHeadline,
		dbConnected,
		[]fyne.CanvasObject{widget.NewLabel(dbStatusDetail)},
	)

	// Server Status Card
	pid, serverRunning := d.ui.app.Server.GetServerStatus()
	serverHeadline := "Stopped"
	var serverStatusDetail string
	if serverRunning {
		serverHeadline = "Running"
		serverStatusDetail = fmt.Sprintf("Running (PID: %d)", pid)
	} else {
		serverStatusDetail = "Stopped"
	}

	serverCard := d.createConnectionCard(
		"Webhook Server",
		serverHeadline,
		serverRunning,
		[]fyne.CanvasObject{widget.NewLabel(serverStatusDetail)},
	)

	return container.NewGridWithColumns(3, apiCard, dbCard, serverCard)
}

func (d *SmartDashboard) createConnectionCard(title, statusText string, isHealthy bool, details []fyne.CanvasObject) fyne.CanvasObject {
	statusColorName := StatusNegativeColorName
	if isHealthy {
		statusColorName = StatusPositiveColorName
	}
	statusColor := d.themeColor(statusColorName)

	statusLabel := canvas.NewText(statusText, statusColor)
	statusLabel.Alignment = fyne.TextAlignTrailing
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	statusLabel.TextSize = theme.TextSize()

	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	header := container.NewHBox(
		titleLabel,
		layout.NewSpacer(),
		statusLabel,
	)

	content := container.NewVBox(header)

	// Add additional details if provided
	if len(details) > 0 {
		content.Add(widget.NewSeparator())
		for _, detail := range details {
			content.Add(detail)
		}
	}

	background := canvas.NewRectangle(d.themeColor(StatusCardBackgroundColorName))
	background.CornerRadius = theme.Padding()
	background.StrokeWidth = 1
	background.StrokeColor = d.themeColor(StatusCardBorderColorName)

	// Slight vertical spacing inside the card for visual balance
	padded := container.NewPadded(content)

	return container.NewStack(background, padded)
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
	if d.ui.rightPaneContent != nil {
		d.ui.ShowDetails(detailsContainer)
	}
}

func (d *SmartDashboard) themeColor(name fyne.ThemeColorName) color.Color {
	return d.ui.themeColor(name)
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
		statWidgets = append(statWidgets, d.createSystemStatCard(stat))
	}

	if len(statWidgets) > 4 {
		statWidgets = statWidgets[:4]
	}

	var grid fyne.CanvasObject
	if len(statWidgets) == 0 {
		empty := widget.NewLabel("No statistics available")
		empty.Alignment = fyne.TextAlignCenter
		grid = container.NewCenter(empty)
	} else {
		grid = container.NewGridWithColumns(2, statWidgets...)
	}

	return container.NewVBox(
		title,
		grid,
	)
}

func (d *SmartDashboard) createSystemStatCard(stat SystemStat) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(stat.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	value := canvas.NewText(stat.Value, theme.ForegroundColor())
	value.Alignment = fyne.TextAlignTrailing
	value.TextStyle = fyne.TextStyle{Bold: true}
	value.TextSize = theme.TextSize() + 2

	description := widget.NewLabel(stat.Description)
	description.Alignment = fyne.TextAlignLeading
	description.Wrapping = fyne.TextWrapWord

	header := container.NewHBox(
		title,
		layout.NewSpacer(),
		value,
	)

	content := container.NewVBox(header)
	if stat.Description != "" {
		content.Add(widget.NewSeparator())
		content.Add(description)
	}

	background := canvas.NewRectangle(d.themeColor(StatusCardBackgroundColorName))
	background.CornerRadius = theme.Padding()
	background.StrokeColor = d.themeColor(StatusCardBorderColorName)
	background.StrokeWidth = 1

	return container.NewStack(
		background,
		container.NewPadded(content),
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
	if d.ui.app.DB == nil || !d.ui.app.DB.IsConnected() {
		return LastSyncInfo{
			Time:   "Unavailable",
			Status: "Connect the database to track sync runs",
		}
	}

	entries, err := database.GetRecentSyncHistory(d.ui.app.DB, 10)
	if err != nil {
		d.ui.app.Events.Dispatch(events.Errorf("dashboard", "Failed to load sync history: %v", err))
		return LastSyncInfo{
			Time:   "Unknown",
			Status: "Unable to load sync history",
		}
	}

	var inProgress *database.SyncHistoryEntry
	for i := range entries {
		entry := entries[i]
		switch strings.ToLower(entry.Status) {
		case "running":
			if inProgress == nil {
				inProgress = &entry
			}
			continue
		case "completed", "completed_with_errors", "failed":
			return d.syncInfoFromEntry(entry)
		default:
			if entry.CompletedAt != nil {
				return d.syncInfoFromEntry(entry)
			}
			if inProgress == nil {
				inProgress = &entry
			}
		}
	}

	if inProgress != nil {
		status := d.describeSyncStatus(*inProgress)
		when := inProgress.StartedAt
		if inProgress.CompletedAt != nil {
			when = *inProgress.CompletedAt
		}
		return LastSyncInfo{
			Time:   fmt.Sprintf("Started %s", formatRelativeTime(when)),
			Status: status,
		}
	}

	return LastSyncInfo{
		Time:   "Never",
		Status: "No sync history recorded yet",
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

func (d *SmartDashboard) syncInfoFromEntry(entry database.SyncHistoryEntry) LastSyncInfo {
	when := entry.StartedAt
	if entry.CompletedAt != nil {
		when = *entry.CompletedAt
	}

	return LastSyncInfo{
		Time:   formatRelativeTime(when),
		Status: d.describeSyncStatus(entry),
	}
}

func (d *SmartDashboard) describeSyncStatus(entry database.SyncHistoryEntry) string {
	if entry.Summary != "" {
		return entry.Summary
	}

	status := humanizeToken(entry.Status)
	source := humanizeToken(entry.Source)
	if source != "" && strings.ToLower(source) != "general" {
		status = fmt.Sprintf("%s (%s)", status, strings.ToLower(source))
	}
	if entry.ErrorCount > 0 && !strings.Contains(strings.ToLower(status), "error") {
		status = fmt.Sprintf("%s with %d error(s)", status, entry.ErrorCount)
	}
	return status
}

func humanizeToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	parts := strings.Split(token, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		rl := strings.ToLower(part)
		parts[i] = strings.ToUpper(rl[:1]) + rl[1:]
	}
	return strings.Join(parts, " ")
}

func formatRelativeTime(ts time.Time) string {
	if ts.IsZero() {
		return "Unknown"
	}

	whenUTC := ts.UTC()
	now := time.Now().UTC()
	if whenUTC.After(now) {
		delta := whenUTC.Sub(now)
		return fmt.Sprintf("in %s", coarseDuration(delta))
	}

	return fmt.Sprintf("%s ago", coarseDuration(now.Sub(whenUTC)))
}

func coarseDuration(d time.Duration) string {
	if d < time.Minute {
		return "moments"
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	if d < 7*24*time.Hour {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	weeks := int(d.Hours() / (24 * 7))
	if weeks == 1 {
		return "1 week"
	}
	return fmt.Sprintf("%d weeks", weeks)
}

// RefreshDashboard updates all dashboard components with latest data
func (d *SmartDashboard) RefreshDashboard() {
	// This would refresh all components with latest data
	d.ui.app.Events.Dispatch(events.Debugf("dashboard", "Refreshing dashboard"))
}
