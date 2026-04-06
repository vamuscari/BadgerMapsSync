package server

import (
	"badgermaps/api"
	"badgermaps/app/audit"
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/events"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

type ComponentHealth struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	LastChecked time.Time              `json:"last_checked"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Latency     time.Duration          `json:"latency_ms"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type HealthCheck struct {
	Status     HealthStatus                `json:"status"`
	Timestamp  time.Time                   `json:"timestamp"`
	Components map[string]*ComponentHealth `json:"components"`
	Version    string                      `json:"version"`
	Uptime     time.Duration               `json:"uptime_seconds"`
}

type HealthChecker struct {
	db            database.DB
	api           *api.APIClient
	events        *events.EventDispatcher
	auditLogger   *audit.AuditLogger
	version       string
	startTime     time.Time
	mu            sync.RWMutex
	lastCheck     *HealthCheck
	checkInterval time.Duration
	stopChan      chan struct{}
	running       bool
}

func NewHealthChecker(
	db database.DB,
	api *api.APIClient,
	events *events.EventDispatcher,
	auditLogger *audit.AuditLogger,
) *HealthChecker {
	return &HealthChecker{
		db:            db,
		api:           api,
		events:        events,
		auditLogger:   auditLogger,
		version:       buildVersionFromBuildInfo(debug.ReadBuildInfo()),
		startTime:     time.Now(),
		checkInterval: 30 * time.Second,
		stopChan:      make(chan struct{}),
	}
}

// Start begins periodic health checks
func (hc *HealthChecker) Start() {
	hc.mu.Lock()
	if hc.running {
		hc.mu.Unlock()
		return
	}
	hc.running = true
	hc.mu.Unlock()

	// Run initial check
	hc.performHealthCheck()

	// Start periodic checks
	go hc.runPeriodic()
}

// Stop halts periodic health checks
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	if !hc.running {
		hc.mu.Unlock()
		return
	}
	hc.running = false
	close(hc.stopChan)
	hc.mu.Unlock()
}

// runPeriodic performs health checks at regular intervals
func (hc *HealthChecker) runPeriodic() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.stopChan:
			return
		case <-ticker.C:
			hc.performHealthCheck()
		}
	}
}

// performHealthCheck runs all health checks
func (hc *HealthChecker) performHealthCheck() {
	check := &HealthCheck{
		Timestamp:  time.Now(),
		Components: make(map[string]*ComponentHealth),
		Version:    hc.version,
		Uptime:     time.Since(hc.startTime),
	}

	// Check all components in parallel
	var wg sync.WaitGroup
	components := []struct {
		name  string
		check func() *ComponentHealth
	}{
		{"database", hc.checkDatabase},
		{"api", hc.checkAPI},
		{"disk", hc.checkDiskSpace},
		{"memory", hc.checkMemory},
	}

	for _, comp := range components {
		wg.Add(1)
		go func(name string, checkFunc func() *ComponentHealth) {
			defer wg.Done()
			health := checkFunc()
			hc.mu.Lock()
			check.Components[name] = health
			hc.mu.Unlock()
		}(comp.name, comp.check)
	}

	wg.Wait()

	// Determine overall status
	check.Status = hc.calculateOverallStatus(check.Components)

	// Store result
	hc.mu.Lock()
	hc.lastCheck = check
	hc.mu.Unlock()

	// Log if unhealthy
	if check.Status != StatusHealthy {
		hc.events.Dispatch(events.Warningf("health", "System health degraded: %s", check.Status))

		if hc.auditLogger != nil {
			hc.auditLogger.Log(&audit.AuditEntry{
				OperationType: audit.OpConfiguration,
				Source:        "HealthCheck",
				Action:        "CHECK",
				Success:       false,
				Level:         audit.LevelWarning,
				Metadata: map[string]interface{}{
					"status": string(check.Status),
				},
			})
		}
	}
}

// checkDatabase validates database connectivity and schema
func (hc *HealthChecker) checkDatabase() *ComponentHealth {
	health := &ComponentHealth{
		Name:        "database",
		LastChecked: time.Now(),
		Status:      StatusHealthy,
	}

	start := time.Now()

	// Test database connection
	if err := hc.db.TestConnection(); err != nil {
		health.Status = StatusUnhealthy
		health.Error = err.Error()
		health.Message = "Database connection failed"
		return health
	}

	// Validate schema
	if err := hc.validateDatabaseSchema(); err != nil {
		health.Status = StatusDegraded
		health.Error = err.Error()
		health.Message = "Database schema validation failed"
		return health
	}

	health.Latency = time.Since(start)
	health.Message = "Database is healthy"

	// Get database stats (if using sql.DB directly)
	sqlDB := hc.db.GetDB()
	if sqlDB != nil {
		stats := sqlDB.Stats()
		health.Metadata = map[string]interface{}{
			"connections": stats.OpenConnections,
			"in_use":      stats.InUse,
			"idle":        stats.Idle,
		}
	}

	return health
}

// checkAPI validates API connectivity and authentication
func (hc *HealthChecker) checkAPI() *ComponentHealth {
	health := &ComponentHealth{
		Name:        "api",
		LastChecked: time.Now(),
		Status:      StatusHealthy,
	}

	if hc.api == nil {
		health.Status = StatusUnhealthy
		health.Message = "API client not initialized"
		return health
	}

	start := time.Now()

	// Test API connection
	if err := hc.api.TestAPIConnection(); err != nil {
		health.Status = StatusUnhealthy
		health.Error = err.Error()
		health.Message = "API connection failed"
		return health
	}

	health.Latency = time.Since(start)
	health.Message = "API is healthy"
	health.Metadata = map[string]interface{}{
		"connected": hc.api.IsConnected(),
		"base_url":  hc.api.BaseURL,
	}

	return health
}

// checkDiskSpace validates available disk space
func (hc *HealthChecker) checkDiskSpace() *ComponentHealth {
	health := &ComponentHealth{
		Name:        "disk",
		LastChecked: time.Now(),
		Status:      StatusHealthy,
	}

	start := time.Now()

	available, total, err := diskUsage(".")
	if err != nil {
		health.Status = StatusUnhealthy
		health.Error = err.Error()
		health.Message = "Failed to get disk statistics"
		return health
	}

	if total == 0 {
		health.Status = StatusUnhealthy
		health.Message = "Disk statistics unavailable"
		health.Metadata = map[string]interface{}{"available_bytes": available, "total_bytes": total}
		return health
	}

	used := total - available
	usedPercent := float64(used) / float64(total) * 100

	// Check thresholds
	if usedPercent > 95 {
		health.Status = StatusUnhealthy
		health.Message = fmt.Sprintf("Critical: Disk usage very high: %.1f%%", usedPercent)
	} else if usedPercent > 85 {
		health.Status = StatusDegraded
		health.Message = fmt.Sprintf("Warning: Disk usage high: %.1f%%", usedPercent)
	} else {
		health.Message = fmt.Sprintf("Disk usage normal: %.1f%%", usedPercent)
	}

	health.Latency = time.Since(start)
	health.Metadata = map[string]interface{}{
		"available_bytes": available,
		"total_bytes":     total,
		"used_bytes":      used,
		"used_percent":    usedPercent,
	}

	return health
}

// checkMemory validates memory usage
func (hc *HealthChecker) checkMemory() *ComponentHealth {
	health := &ComponentHealth{
		Name:        "memory",
		LastChecked: time.Now(),
		Status:      StatusHealthy,
	}

	start := time.Now()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calculate memory usage
	allocatedMB := float64(m.Alloc) / 1024 / 1024
	totalAllocatedMB := float64(m.TotalAlloc) / 1024 / 1024
	systemMB := float64(m.Sys) / 1024 / 1024
	numGC := m.NumGC

	// Check thresholds (in MB)
	if allocatedMB > 1000 { // More than 1GB allocated
		health.Status = StatusDegraded
		health.Message = fmt.Sprintf("Warning: High memory usage: %.2f MB", allocatedMB)
	} else if allocatedMB > 2000 { // More than 2GB allocated
		health.Status = StatusUnhealthy
		health.Message = fmt.Sprintf("Critical: Very high memory usage: %.2f MB", allocatedMB)
	} else {
		health.Message = fmt.Sprintf("Memory usage normal: %.2f MB", allocatedMB)
	}

	health.Latency = time.Since(start)
	health.Metadata = map[string]interface{}{
		"allocated_mb":       allocatedMB,
		"total_allocated_mb": totalAllocatedMB,
		"system_mb":          systemMB,
		"gc_runs":            numGC,
		"goroutines":         runtime.NumGoroutine(),
	}

	return health
}

// calculateOverallStatus determines the overall health status
func (hc *HealthChecker) calculateOverallStatus(components map[string]*ComponentHealth) HealthStatus {
	hasUnhealthy := false
	hasDegraded := false

	for _, comp := range components {
		switch comp.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}
	return StatusHealthy
}

// GetHealth returns the current health status
func (hc *HealthChecker) GetHealth() *HealthCheck {
	hc.mu.RLock()
	if hc.lastCheck != nil {
		defer hc.mu.RUnlock()
		return hc.lastCheck
	}
	hc.mu.RUnlock()

	// Perform check without holding read lock
	hc.performHealthCheck()

	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.lastCheck
}

// HTTPHandler returns an HTTP handler for health checks
func (hc *HealthChecker) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := hc.GetHealth()

		// Set appropriate status code
		statusCode := http.StatusOK
		switch health.Status {
		case StatusDegraded:
			statusCode = http.StatusOK // Still operational but degraded
		case StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(health)
	}
}

// DataValidator provides data validation functionality
type DataValidator struct {
	db     database.DB
	api    *api.APIClient
	events *events.EventDispatcher
}

func buildVersionFromBuildInfo(info *debug.BuildInfo, ok bool) string {
	const defaultVersion = "development"

	if !ok || info == nil {
		return defaultVersion
	}

	if version := strings.TrimSpace(info.Main.Version); version != "" && version != "(devel)" {
		return version
	}

	revision := ""
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = strings.TrimSpace(setting.Value)
		case "vcs.modified":
			modified = strings.EqualFold(strings.TrimSpace(setting.Value), "true")
		}
	}

	if revision == "" {
		return defaultVersion
	}

	if len(revision) > 12 {
		revision = revision[:12]
	}
	if modified {
		return revision + "-dirty"
	}
	return revision
}

func NewDataValidator(db database.DB, api *api.APIClient, events *events.EventDispatcher) *DataValidator {
	return &DataValidator{
		db:     db,
		api:    api,
		events: events,
	}
}

// ValidateBeforeSync performs validation checks before sync operations
func (dv *DataValidator) ValidateBeforeSync(syncType SyncType) error {
	// Check API connectivity
	if err := dv.api.TestAPIConnection(); err != nil {
		return fmt.Errorf("API validation failed: %w", err)
	}

	// Check database connectivity
	if err := dv.db.TestConnection(); err != nil {
		return fmt.Errorf("database validation failed: %w", err)
	}

	// Validate schema based on sync type
	switch syncType {
	case SyncTypeAccounts:
		if err := dv.validateAccountsSchema(); err != nil {
			return fmt.Errorf("accounts schema validation failed: %w", err)
		}
	case SyncTypeCheckins:
		if err := dv.validateCheckinsSchema(); err != nil {
			return fmt.Errorf("checkins schema validation failed: %w", err)
		}
	case SyncTypeRoutes:
		if err := dv.validateRoutesSchema(); err != nil {
			return fmt.Errorf("routes schema validation failed: %w", err)
		}
	}

	dv.events.Dispatch(events.Infof("validator", "Pre-sync validation passed for %s", syncType))
	return nil
}

// ValidateAccountData validates account data integrity
func (dv *DataValidator) ValidateAccountData(accountID int) error {
	if accountID <= 0 {
		return fmt.Errorf("account id must be greater than 0")
	}

	account, err := database.GetAccountByID(dv.db, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("account %d does not exist", accountID)
		}
		return fmt.Errorf("failed to validate account %d: %w", accountID, err)
	}
	if account == nil {
		return fmt.Errorf("account %d does not exist", accountID)
	}
	if !account.AccountId.Valid {
		return fmt.Errorf("account %d has invalid account id value", accountID)
	}
	if int(account.AccountId.Int64) != accountID {
		return fmt.Errorf("account id mismatch: expected %d, got %d", accountID, account.AccountId.Int64)
	}

	return nil
}

// ValidateDataConsistency checks for data consistency issues
func (dv *DataValidator) ValidateDataConsistency() error {
	checks := []struct {
		command string
		label   string
	}{
		{command: "CountOrphanedCheckins", label: "orphaned_checkins"},
		{command: "CountOrphanedAccountLocations", label: "orphaned_account_locations"},
		{command: "CountOrphanedRouteWaypoints", label: "orphaned_route_waypoints"},
	}

	var violations []string
	for _, check := range checks {
		count, err := dv.queryCount(check.command)
		if err != nil {
			return fmt.Errorf("failed consistency check %s: %w", check.label, err)
		}
		if count > 0 {
			violations = append(violations, fmt.Sprintf("%s=%d", check.label, count))
		}
	}

	if len(violations) > 0 {
		return fmt.Errorf("data consistency violations detected: %s", strings.Join(violations, ", "))
	}

	return nil
}

// Helper methods for schema validation
func (hc *HealthChecker) validateDatabaseSchema() error {
	if hc.db == nil {
		return fmt.Errorf("database is not configured")
	}

	s := state.NewState()
	s.Quiet = true
	return hc.db.ValidateSchema(s)
}

func (dv *DataValidator) validateAccountsSchema() error {
	return dv.validateSchemaTables("Accounts", "AccountLocations", "AccountsPendingChanges")
}

func (dv *DataValidator) validateCheckinsSchema() error {
	return dv.validateSchemaTables("AccountCheckins", "AccountCheckinsPendingChanges")
}

func (dv *DataValidator) validateRoutesSchema() error {
	return dv.validateSchemaTables("Routes", "RouteWaypoints")
}

func (dv *DataValidator) validateSchemaTables(tableNames ...string) error {
	expectedSchema := database.GetExpectedSchema()
	for _, tableName := range tableNames {
		expectedColumns, ok := expectedSchema[tableName]
		if !ok {
			return fmt.Errorf("no expected schema registered for table %s", tableName)
		}
		if err := dv.validateTableSchema(tableName, expectedColumns); err != nil {
			return err
		}
	}
	return nil
}

func (dv *DataValidator) validateTableSchema(tableName string, expectedColumns []string) error {
	exists, err := dv.db.TableExists(tableName)
	if err != nil {
		return fmt.Errorf("failed to inspect table %s: %w", tableName, err)
	}
	if !exists {
		return fmt.Errorf("required table %s does not exist", tableName)
	}

	columns, err := dv.db.GetTableColumns(tableName)
	if err != nil {
		return fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
	}

	columnSet := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		columnSet[strings.ToLower(strings.TrimSpace(column))] = struct{}{}
	}

	for _, expected := range expectedColumns {
		if _, ok := columnSet[strings.ToLower(strings.TrimSpace(expected))]; !ok {
			return fmt.Errorf("missing column %s in table %s", expected, tableName)
		}
	}

	return nil
}

func (dv *DataValidator) queryCount(command string) (int, error) {
	sqlText := strings.TrimSpace(dv.db.GetSQL(command))
	if sqlText == "" {
		return 0, fmt.Errorf("SQL command %s is unavailable for database type %s", command, dv.db.GetType())
	}

	sqlDB := dv.db.GetDB()
	if sqlDB == nil {
		return 0, fmt.Errorf("database connection is not initialized")
	}

	var count int
	if err := sqlDB.QueryRow(sqlText).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
