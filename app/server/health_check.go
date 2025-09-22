package server

import (
	"badgermaps/api"
	"badgermaps/app/audit"
	"badgermaps/database"
	"badgermaps/events"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
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
		Version:    "1.0.0", // TODO: Get from build info
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
	// TODO: Implement account validation when database methods are available
	// This will require adding GetAccount method to database.DB interface
	// or using ExecuteQuery to fetch account data

	return nil
}

// ValidateDataConsistency checks for data consistency issues
func (dv *DataValidator) ValidateDataConsistency() error {
	// TODO: Implement data consistency checks
	// This will require adding methods to database.DB interface or using ExecuteQuery
	// to run custom queries for finding orphaned and duplicate records

	// Example queries that could be implemented:
	// - SELECT COUNT(*) FROM account_checkins WHERE account_id NOT IN (SELECT id FROM accounts)
	// - SELECT customer_id, COUNT(*) FROM accounts GROUP BY customer_id HAVING COUNT(*) > 1

	return nil
}

// Helper methods for schema validation
func (hc *HealthChecker) validateDatabaseSchema() error {
	// TODO: Implement schema validation
	return nil
}

func (dv *DataValidator) validateAccountsSchema() error {
	// TODO: Implement accounts schema validation
	return nil
}

func (dv *DataValidator) validateCheckinsSchema() error {
	// TODO: Implement checkins schema validation
	return nil
}

func (dv *DataValidator) validateRoutesSchema() error {
	// TODO: Implement routes schema validation
	return nil
}
