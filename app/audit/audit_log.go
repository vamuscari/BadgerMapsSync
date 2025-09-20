package audit

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogLevel string

const (
	LevelInfo     LogLevel = "INFO"
	LevelWarning  LogLevel = "WARNING"
	LevelError    LogLevel = "ERROR"
	LevelCritical LogLevel = "CRITICAL"
	LevelDebug    LogLevel = "DEBUG"
)

type OperationType string

const (
	OpAPICall        OperationType = "API_CALL"
	OpDatabaseChange OperationType = "DB_CHANGE"
	OpWebhookReceive OperationType = "WEBHOOK_IN"
	OpWebhookSend    OperationType = "WEBHOOK_OUT"
	OpSyncPull       OperationType = "SYNC_PULL"
	OpSyncPush       OperationType = "SYNC_PUSH"
	OpBackup         OperationType = "BACKUP"
	OpRestore        OperationType = "RESTORE"
	OpScheduledJob   OperationType = "SCHEDULED_JOB"
	OpAuthentication OperationType = "AUTH"
	OpConfiguration  OperationType = "CONFIG"
)

type AuditEntry struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	Level         LogLevel               `json:"level"`
	OperationType OperationType          `json:"operation_type"`
	User          string                 `json:"user,omitempty"`
	Source        string                 `json:"source"`
	Action        string                 `json:"action"`
	Resource      string                 `json:"resource,omitempty"`
	ResourceID    string                 `json:"resource_id,omitempty"`
	Success       bool                   `json:"success"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Duration      time.Duration          `json:"duration_ms,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type AuditLogger struct {
	mu            sync.RWMutex
	file          *os.File
	logger        *log.Logger
	rotationSize  int64
	retentionDays int
	currentSize   int64
	logPath       string
	enabled       bool
	asyncQueue    chan *AuditEntry
	wg            sync.WaitGroup
}

// NewAuditLogger creates a new audit logger instance
func NewAuditLogger(logPath string, enabled bool) (*AuditLogger, error) {
	if !enabled {
		return &AuditLogger{enabled: false}, nil
	}

	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat audit log file: %w", err)
	}

	al := &AuditLogger{
		file:          file,
		logger:        log.New(file, "", 0),
		rotationSize:  100 * 1024 * 1024, // 100MB default
		retentionDays: 90,                // 90 days default
		currentSize:   info.Size(),
		logPath:       logPath,
		enabled:       true,
		asyncQueue:    make(chan *AuditEntry, 1000),
	}

	// Start async writer
	al.wg.Add(1)
	go al.asyncWriter()

	// Start cleanup routine
	go al.cleanupOldLogs()

	return al, nil
}

// Log creates and writes an audit log entry
func (al *AuditLogger) Log(entry *AuditEntry) {
	if !al.enabled {
		return
	}

	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = generateAuditID()
	}

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Send to async queue
	select {
	case al.asyncQueue <- entry:
	default:
		// Queue full, write synchronously
		al.writeEntry(entry)
	}
}

// LogAPICall logs an API operation
func (al *AuditLogger) LogAPICall(method, endpoint string, success bool, duration time.Duration, err error) {
	entry := &AuditEntry{
		OperationType: OpAPICall,
		Source:        "API",
		Action:        method,
		Resource:      endpoint,
		Success:       success,
		Duration:      duration,
		Level:         LevelInfo,
	}

	if err != nil {
		entry.ErrorMessage = err.Error()
		entry.Level = LevelError
	}

	al.Log(entry)
}

// LogDatabaseChange logs a database operation
func (al *AuditLogger) LogDatabaseChange(operation, table string, recordID string, success bool, err error) {
	entry := &AuditEntry{
		OperationType: OpDatabaseChange,
		Source:        "Database",
		Action:        operation,
		Resource:      table,
		ResourceID:    recordID,
		Success:       success,
		Level:         LevelInfo,
	}

	if err != nil {
		entry.ErrorMessage = err.Error()
		entry.Level = LevelError
	}

	al.Log(entry)
}

// LogWebhook logs webhook operations
func (al *AuditLogger) LogWebhook(direction string, url string, payload interface{}, success bool, err error) {
	opType := OpWebhookReceive
	if direction == "outgoing" {
		opType = OpWebhookSend
	}

	entry := &AuditEntry{
		OperationType: opType,
		Source:        "Webhook",
		Action:        direction,
		Resource:      url,
		Success:       success,
		Level:         LevelInfo,
	}

	if payload != nil {
		entry.Metadata = map[string]interface{}{
			"payload_preview": truncatePayload(payload, 500),
		}
	}

	if err != nil {
		entry.ErrorMessage = err.Error()
		entry.Level = LevelError
	}

	al.Log(entry)
}

// LogSync logs sync operations
func (al *AuditLogger) LogSync(syncType string, recordCount int, success bool, duration time.Duration, err error) {
	opType := OpSyncPull
	if syncType == "push" {
		opType = OpSyncPush
	}

	entry := &AuditEntry{
		OperationType: opType,
		Source:        "Sync",
		Action:        syncType,
		Success:       success,
		Duration:      duration,
		Level:         LevelInfo,
		Metadata: map[string]interface{}{
			"record_count": recordCount,
		},
	}

	if err != nil {
		entry.ErrorMessage = err.Error()
		entry.Level = LevelError
	}

	al.Log(entry)
}

// asyncWriter handles asynchronous writing of audit entries
func (al *AuditLogger) asyncWriter() {
	defer al.wg.Done()

	for entry := range al.asyncQueue {
		al.writeEntry(entry)
	}
}

// writeEntry writes a single audit entry to the log file
func (al *AuditLogger) writeEntry(entry *AuditEntry) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Check if rotation is needed
	if al.currentSize >= al.rotationSize {
		al.rotateLog()
	}

	// Convert entry to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal audit entry: %v", err)
		return
	}

	// Write to file
	if _, err := al.file.Write(append(data, '\n')); err != nil {
		log.Printf("Failed to write audit entry: %v", err)
		return
	}

	al.currentSize += int64(len(data) + 1)
}

// rotateLog rotates the current log file
func (al *AuditLogger) rotateLog() {
	// Close current file
	al.file.Close()

	// Generate rotation filename
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", al.logPath, timestamp)

	// Rename current file
	if err := os.Rename(al.logPath, rotatedPath); err != nil {
		log.Printf("Failed to rotate audit log: %v", err)
	}

	// Open new file
	file, err := os.OpenFile(al.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open new audit log file: %v", err)
		return
	}

	al.file = file
	al.logger = log.New(file, "", 0)
	al.currentSize = 0
}

// cleanupOldLogs removes audit logs older than retention period
func (al *AuditLogger) cleanupOldLogs() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		al.performCleanup()
	}
}

// performCleanup performs the actual cleanup of old log files
func (al *AuditLogger) performCleanup() {
	dir := filepath.Dir(al.logPath)
	baseFileName := filepath.Base(al.logPath)
	cutoffTime := time.Now().AddDate(0, 0, -al.retentionDays)

	// Read directory
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Failed to read audit log directory: %v", err)
		return
	}

	for _, file := range files {
		// Check if file is a rotated audit log
		if !file.IsDir() && len(file.Name()) > len(baseFileName) &&
			file.Name()[:len(baseFileName)] == baseFileName {

			info, err := file.Info()
			if err != nil {
				continue
			}

			// Remove if older than retention period
			if info.ModTime().Before(cutoffTime) {
				filePath := filepath.Join(dir, file.Name())
				if err := os.Remove(filePath); err != nil {
					log.Printf("Failed to remove old audit log %s: %v", filePath, err)
				}
			}
		}
	}
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	if !al.enabled {
		return nil
	}

	// Close async queue
	close(al.asyncQueue)

	// Wait for pending writes
	al.wg.Wait()

	// Close file
	return al.file.Close()
}

// Query searches audit logs based on filters
func (al *AuditLogger) Query(filters AuditFilters) ([]*AuditEntry, error) {
	if !al.enabled {
		return nil, nil
	}

	al.mu.RLock()
	defer al.mu.RUnlock()

	var entries []*AuditEntry

	// Read from current file and rotated files
	files := []string{al.logPath}

	// Find rotated files
	dir := filepath.Dir(al.logPath)
	baseFileName := filepath.Base(al.logPath)

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range dirEntries {
		if !file.IsDir() && len(file.Name()) > len(baseFileName) &&
			file.Name()[:len(baseFileName)] == baseFileName {
			files = append(files, filepath.Join(dir, file.Name()))
		}
	}

	// Read and filter entries from each file
	for _, filePath := range files {
		fileEntries, err := al.readEntriesFromFile(filePath, filters)
		if err != nil {
			log.Printf("Failed to read audit file %s: %v", filePath, err)
			continue
		}
		entries = append(entries, fileEntries...)
	}

	return entries, nil
}

// readEntriesFromFile reads audit entries from a specific file
func (al *AuditLogger) readEntriesFromFile(filePath string, filters AuditFilters) ([]*AuditEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []*AuditEntry
	decoder := json.NewDecoder(file)

	for decoder.More() {
		var entry AuditEntry
		if err := decoder.Decode(&entry); err != nil {
			continue // Skip malformed entries
		}

		if filters.Matches(&entry) {
			entries = append(entries, &entry)
		}
	}

	return entries, nil
}

// AuditFilters defines filters for querying audit logs
type AuditFilters struct {
	StartTime     *time.Time
	EndTime       *time.Time
	OperationType *OperationType
	Level         *LogLevel
	Success       *bool
	User          *string
	Resource      *string
	Limit         int
}

// Matches checks if an audit entry matches the filters
func (f *AuditFilters) Matches(entry *AuditEntry) bool {
	if f.StartTime != nil && entry.Timestamp.Before(*f.StartTime) {
		return false
	}
	if f.EndTime != nil && entry.Timestamp.After(*f.EndTime) {
		return false
	}
	if f.OperationType != nil && entry.OperationType != *f.OperationType {
		return false
	}
	if f.Level != nil && entry.Level != *f.Level {
		return false
	}
	if f.Success != nil && entry.Success != *f.Success {
		return false
	}
	if f.User != nil && entry.User != *f.User {
		return false
	}
	if f.Resource != nil && entry.Resource != *f.Resource {
		return false
	}
	return true
}

// Helper functions

func generateAuditID() string {
	return fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), os.Getpid())
}

func truncatePayload(payload interface{}, maxLength int) string {
	data, _ := json.Marshal(payload)
	if len(data) > maxLength {
		return string(data[:maxLength]) + "..."
	}
	return string(data)
}
