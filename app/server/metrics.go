package server

import (
	"badgermaps/app/audit"
	"badgermaps/events"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// MetricType represents different types of metrics
type MetricType string

const (
	MetricCounter   MetricType = "counter"
	MetricGauge     MetricType = "gauge"
	MetricHistogram MetricType = "histogram"
	MetricSummary   MetricType = "summary"
)

// Metric represents a single metric
type Metric struct {
	Name        string                 `json:"name"`
	Type        MetricType             `json:"type"`
	Value       float64                `json:"value"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description,omitempty"`
	Unit        string                 `json:"unit,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MetricSnapshot represents a point-in-time view of all metrics
type MetricSnapshot struct {
	Timestamp   time.Time           `json:"timestamp"`
	Metrics     map[string]*Metric  `json:"metrics"`
	System      *SystemMetrics      `json:"system"`
	Performance *PerformanceMetrics `json:"performance"`
}

// SystemMetrics contains system-level metrics
type SystemMetrics struct {
	CPUUsage       float64       `json:"cpu_usage_percent"`
	MemoryUsed     uint64        `json:"memory_used_bytes"`
	MemoryTotal    uint64        `json:"memory_total_bytes"`
	MemoryPercent  float64       `json:"memory_usage_percent"`
	GoroutineCount int           `json:"goroutine_count"`
	GCPauseNS      uint64        `json:"gc_pause_ns"`
	GCRuns         uint32        `json:"gc_runs"`
	Uptime         time.Duration `json:"uptime_seconds"`
}

// PerformanceMetrics contains application performance metrics
type PerformanceMetrics struct {
	APICallsTotal     int64         `json:"api_calls_total"`
	APICallsSuccess   int64         `json:"api_calls_success"`
	APICallsFailed    int64         `json:"api_calls_failed"`
	APILatencyP50     time.Duration `json:"api_latency_p50_ms"`
	APILatencyP95     time.Duration `json:"api_latency_p95_ms"`
	APILatencyP99     time.Duration `json:"api_latency_p99_ms"`
	DBQueriesTotal    int64         `json:"db_queries_total"`
	DBQueriesSuccess  int64         `json:"db_queries_success"`
	DBQueriesFailed   int64         `json:"db_queries_failed"`
	DBLatencyP50      time.Duration `json:"db_latency_p50_ms"`
	DBLatencyP95      time.Duration `json:"db_latency_p95_ms"`
	DBLatencyP99      time.Duration `json:"db_latency_p99_ms"`
	SyncOpsTotal      int64         `json:"sync_ops_total"`
	SyncOpsSuccess    int64         `json:"sync_ops_success"`
	SyncOpsFailed     int64         `json:"sync_ops_failed"`
	WebhooksReceived  int64         `json:"webhooks_received"`
	WebhooksProcessed int64         `json:"webhooks_processed"`
	JobsScheduled     int64         `json:"jobs_scheduled"`
	JobsExecuted      int64         `json:"jobs_executed"`
	JobsSuccess       int64         `json:"jobs_success"`
	JobsFailed        int64         `json:"jobs_failed"`
}

// MetricsCollector collects and manages metrics
type MetricsCollector struct {
	mu          sync.RWMutex
	metrics     map[string]*Metric
	startTime   time.Time
	events      *events.EventDispatcher
	auditLogger *audit.AuditLogger

	// Performance counters (atomic for thread safety)
	apiCallsTotal     atomic.Int64
	apiCallsSuccess   atomic.Int64
	apiCallsFailed    atomic.Int64
	dbQueriesTotal    atomic.Int64
	dbQueriesSuccess  atomic.Int64
	dbQueriesFailed   atomic.Int64
	syncOpsTotal      atomic.Int64
	syncOpsSuccess    atomic.Int64
	syncOpsFailed     atomic.Int64
	webhooksReceived  atomic.Int64
	webhooksProcessed atomic.Int64
	jobsScheduled     atomic.Int64
	jobsExecuted      atomic.Int64
	jobsSuccess       atomic.Int64
	jobsFailed        atomic.Int64

	// Latency tracking
	apiLatencies []time.Duration
	dbLatencies  []time.Duration
	latencyMu    sync.Mutex

	// Collection settings
	retentionPeriod time.Duration
	collectionRate  time.Duration
	stopChan        chan struct{}
	running         bool
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(events *events.EventDispatcher, auditLogger *audit.AuditLogger) *MetricsCollector {
	return &MetricsCollector{
		metrics:         make(map[string]*Metric),
		startTime:       time.Now(),
		events:          events,
		auditLogger:     auditLogger,
		apiLatencies:    make([]time.Duration, 0, 1000),
		dbLatencies:     make([]time.Duration, 0, 1000),
		retentionPeriod: 24 * time.Hour,
		collectionRate:  10 * time.Second,
		stopChan:        make(chan struct{}),
	}
}

// Start begins metrics collection
func (mc *MetricsCollector) Start() {
	mc.mu.Lock()
	if mc.running {
		mc.mu.Unlock()
		return
	}
	mc.running = true
	mc.mu.Unlock()

	go mc.collectPeriodic()

	mc.events.Dispatch(events.Infof("metrics", "Metrics collector started"))
}

// Stop halts metrics collection
func (mc *MetricsCollector) Stop() {
	mc.mu.Lock()
	if !mc.running {
		mc.mu.Unlock()
		return
	}
	mc.running = false
	close(mc.stopChan)
	mc.mu.Unlock()

	mc.events.Dispatch(events.Infof("metrics", "Metrics collector stopped"))
}

// collectPeriodic runs periodic metric collection
func (mc *MetricsCollector) collectPeriodic() {
	ticker := time.NewTicker(mc.collectionRate)
	defer ticker.Stop()

	for {
		select {
		case <-mc.stopChan:
			return
		case <-ticker.C:
			mc.collectSystemMetrics()
			mc.pruneOldMetrics()
		}
	}
}

// collectSystemMetrics collects current system metrics
func (mc *MetricsCollector) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mc.RecordGauge("system.memory.alloc", float64(m.Alloc), "bytes", "Allocated memory")
	mc.RecordGauge("system.memory.total", float64(m.TotalAlloc), "bytes", "Total allocated memory")
	mc.RecordGauge("system.memory.sys", float64(m.Sys), "bytes", "System memory")
	mc.RecordGauge("system.gc.runs", float64(m.NumGC), "count", "Number of GC runs")
	mc.RecordGauge("system.gc.pause", float64(m.PauseNs[(m.NumGC+255)%256]), "nanoseconds", "Last GC pause duration")
	mc.RecordGauge("system.goroutines", float64(runtime.NumGoroutine()), "count", "Number of goroutines")
	mc.RecordGauge("system.cpu.count", float64(runtime.NumCPU()), "count", "Number of CPUs")
}

// RecordCounter increments a counter metric
func (mc *MetricsCollector) RecordCounter(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metric, exists := mc.metrics[name]
	if !exists {
		metric = &Metric{
			Name:   name,
			Type:   MetricCounter,
			Value:  0,
			Labels: labels,
		}
		mc.metrics[name] = metric
	}

	metric.Value += value
	metric.Timestamp = time.Now()
}

// RecordGauge sets a gauge metric
func (mc *MetricsCollector) RecordGauge(name string, value float64, unit string, description string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics[name] = &Metric{
		Name:        name,
		Type:        MetricGauge,
		Value:       value,
		Unit:        unit,
		Description: description,
		Timestamp:   time.Now(),
	}
}

// RecordAPICall records an API call metric
func (mc *MetricsCollector) RecordAPICall(endpoint string, success bool, latency time.Duration) {
	mc.apiCallsTotal.Add(1)

	if success {
		mc.apiCallsSuccess.Add(1)
	} else {
		mc.apiCallsFailed.Add(1)
	}

	mc.latencyMu.Lock()
	mc.apiLatencies = append(mc.apiLatencies, latency)
	// Keep only last 1000 samples
	if len(mc.apiLatencies) > 1000 {
		mc.apiLatencies = mc.apiLatencies[len(mc.apiLatencies)-1000:]
	}
	mc.latencyMu.Unlock()

	// Record individual endpoint metrics
	labels := map[string]string{"endpoint": endpoint, "status": fmt.Sprintf("%v", success)}
	mc.RecordCounter("api.calls", 1, labels)
}

// RecordDBQuery records a database query metric
func (mc *MetricsCollector) RecordDBQuery(table string, operation string, success bool, latency time.Duration) {
	mc.dbQueriesTotal.Add(1)

	if success {
		mc.dbQueriesSuccess.Add(1)
	} else {
		mc.dbQueriesFailed.Add(1)
	}

	mc.latencyMu.Lock()
	mc.dbLatencies = append(mc.dbLatencies, latency)
	// Keep only last 1000 samples
	if len(mc.dbLatencies) > 1000 {
		mc.dbLatencies = mc.dbLatencies[len(mc.dbLatencies)-1000:]
	}
	mc.latencyMu.Unlock()

	// Record individual query metrics
	labels := map[string]string{"table": table, "operation": operation, "status": fmt.Sprintf("%v", success)}
	mc.RecordCounter("db.queries", 1, labels)
}

// RecordSyncOperation records a sync operation metric
func (mc *MetricsCollector) RecordSyncOperation(syncType string, success bool, recordCount int) {
	mc.syncOpsTotal.Add(1)

	if success {
		mc.syncOpsSuccess.Add(1)
	} else {
		mc.syncOpsFailed.Add(1)
	}

	labels := map[string]string{"type": syncType, "status": fmt.Sprintf("%v", success)}
	mc.RecordCounter("sync.operations", 1, labels)

	if success && recordCount > 0 {
		mc.RecordCounter("sync.records", float64(recordCount), map[string]string{"type": syncType})
	}
}

// RecordWebhook records webhook metrics
func (mc *MetricsCollector) RecordWebhook(received bool) {
	if received {
		mc.webhooksReceived.Add(1)
	} else {
		mc.webhooksProcessed.Add(1)
	}
}

// RecordJob records job execution metrics
func (mc *MetricsCollector) RecordJob(scheduled bool, success bool) {
	if scheduled {
		mc.jobsScheduled.Add(1)
	} else {
		mc.jobsExecuted.Add(1)
		if success {
			mc.jobsSuccess.Add(1)
		} else {
			mc.jobsFailed.Add(1)
		}
	}
}

// GetSnapshot returns a snapshot of all current metrics
func (mc *MetricsCollector) GetSnapshot() *MetricSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Copy metrics
	metrics := make(map[string]*Metric)
	for k, v := range mc.metrics {
		metricCopy := *v
		metrics[k] = &metricCopy
	}

	// Calculate system metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	system := &SystemMetrics{
		MemoryUsed:     m.Alloc,
		MemoryTotal:    m.Sys,
		MemoryPercent:  float64(m.Alloc) / float64(m.Sys) * 100,
		GoroutineCount: runtime.NumGoroutine(),
		GCPauseNS:      m.PauseNs[(m.NumGC+255)%256],
		GCRuns:         m.NumGC,
		Uptime:         time.Since(mc.startTime),
	}

	// Calculate performance metrics
	performance := &PerformanceMetrics{
		APICallsTotal:     mc.apiCallsTotal.Load(),
		APICallsSuccess:   mc.apiCallsSuccess.Load(),
		APICallsFailed:    mc.apiCallsFailed.Load(),
		DBQueriesTotal:    mc.dbQueriesTotal.Load(),
		DBQueriesSuccess:  mc.dbQueriesSuccess.Load(),
		DBQueriesFailed:   mc.dbQueriesFailed.Load(),
		SyncOpsTotal:      mc.syncOpsTotal.Load(),
		SyncOpsSuccess:    mc.syncOpsSuccess.Load(),
		SyncOpsFailed:     mc.syncOpsFailed.Load(),
		WebhooksReceived:  mc.webhooksReceived.Load(),
		WebhooksProcessed: mc.webhooksProcessed.Load(),
		JobsScheduled:     mc.jobsScheduled.Load(),
		JobsExecuted:      mc.jobsExecuted.Load(),
		JobsSuccess:       mc.jobsSuccess.Load(),
		JobsFailed:        mc.jobsFailed.Load(),
	}

	// Calculate latency percentiles
	mc.latencyMu.Lock()
	if len(mc.apiLatencies) > 0 {
		performance.APILatencyP50 = calculatePercentile(mc.apiLatencies, 50)
		performance.APILatencyP95 = calculatePercentile(mc.apiLatencies, 95)
		performance.APILatencyP99 = calculatePercentile(mc.apiLatencies, 99)
	}
	if len(mc.dbLatencies) > 0 {
		performance.DBLatencyP50 = calculatePercentile(mc.dbLatencies, 50)
		performance.DBLatencyP95 = calculatePercentile(mc.dbLatencies, 95)
		performance.DBLatencyP99 = calculatePercentile(mc.dbLatencies, 99)
	}
	mc.latencyMu.Unlock()

	return &MetricSnapshot{
		Timestamp:   time.Now(),
		Metrics:     metrics,
		System:      system,
		Performance: performance,
	}
}

// HTTPHandler returns an HTTP handler for metrics endpoint
func (mc *MetricsCollector) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot := mc.GetSnapshot()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(snapshot)
	}
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics = make(map[string]*Metric)
	mc.apiCallsTotal.Store(0)
	mc.apiCallsSuccess.Store(0)
	mc.apiCallsFailed.Store(0)
	mc.dbQueriesTotal.Store(0)
	mc.dbQueriesSuccess.Store(0)
	mc.dbQueriesFailed.Store(0)
	mc.syncOpsTotal.Store(0)
	mc.syncOpsSuccess.Store(0)
	mc.syncOpsFailed.Store(0)
	mc.webhooksReceived.Store(0)
	mc.webhooksProcessed.Store(0)
	mc.jobsScheduled.Store(0)
	mc.jobsExecuted.Store(0)
	mc.jobsSuccess.Store(0)
	mc.jobsFailed.Store(0)

	mc.latencyMu.Lock()
	mc.apiLatencies = make([]time.Duration, 0, 1000)
	mc.dbLatencies = make([]time.Duration, 0, 1000)
	mc.latencyMu.Unlock()

	mc.events.Dispatch(events.Infof("metrics", "Metrics reset"))
}

// pruneOldMetrics removes metrics older than retention period
func (mc *MetricsCollector) pruneOldMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-mc.retentionPeriod)

	for name, metric := range mc.metrics {
		if metric.Timestamp.Before(cutoff) {
			delete(mc.metrics, name)
		}
	}
}

// Helper function to calculate percentiles
func calculatePercentile(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 || percentile < 0 || percentile > 100 {
		return 0
	}

	// Create a copy to avoid modifying original slice
	sorted := make([]time.Duration, len(values))
	copy(sorted, values)

	// Sort the values
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate index for the percentile
	index := int(math.Ceil(float64(len(sorted))*percentile/100)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}
