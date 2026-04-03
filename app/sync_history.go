package app

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"badgermaps/database"
	"badgermaps/events"
)

type jobLogRun struct {
	correlationID string
	startedAt     time.Time
	direction     string
	source        string
	expectedItems int
	errorCount    int
}

type jobLogMetricTarget struct {
	correlationID string
	fallback      bool
	expectedItems int
	errorCount    int
}

type jobLogJobMeta struct {
	CorrelationID       string
	ParentCorrelationID string
	RootCorrelationID   string
	RunType             string
	Direction           string
	Source              string
	Initiator           string
	JobKind             string
	Mode                string
	StepID              string
	StepIndex           int
	TotalSteps          int
	ActionType          string
	CommandText         string
}

type jobLogMetricSnapshot struct {
	ItemsProcessed int
	ErrorCount     int
	Summary        string
	Details        string
}

type jobLogPendingCompletion struct {
	Status      string
	ErrorCount  int
	ErrorDetail string
}

type jobLogMetricTerminal struct {
	Status     string
	ErrorCount int
	Details    string
}

func (a *App) ensureJobLogTracking() {
	if a.Events == nil || a.DB == nil || !a.DB.IsConnected() {
		return
	}

	a.jobLogMu.Lock()
	if a.jobLogRuns == nil {
		a.jobLogRuns = make(map[string]*jobLogRun)
	}
	if a.jobLogTargets == nil {
		a.jobLogTargets = make(map[string]*jobLogMetricTarget)
	}
	if a.jobLogJobMeta == nil {
		a.jobLogJobMeta = make(map[string]*jobLogJobMeta)
	}
	if a.jobLogMetrics == nil {
		a.jobLogMetrics = make(map[string]*jobLogMetricSnapshot)
	}
	if a.jobLogPending == nil {
		a.jobLogPending = make(map[string]*jobLogPendingCompletion)
	}
	if a.jobLogDone == nil {
		a.jobLogDone = make(map[string]*jobLogMetricTerminal)
	}
	if a.jobLogOnce {
		a.jobLogMu.Unlock()
		return
	}
	a.jobLogOnce = true
	a.jobLogMu.Unlock()

	a.Events.Subscribe("sync.job.*", a.recordSyncJobEvent)
	a.Events.Subscribe("pull.*", a.recordJobLogMetricEvent)
	a.Events.Subscribe("push.*", a.recordJobLogMetricEvent)
}

func (a *App) recordSyncJobEvent(e events.Event) {
	if a.DB == nil || !a.DB.IsConnected() {
		return
	}

	payload, ok := e.Payload.(events.GenericPayload)
	if !ok || payload.Data == nil {
		return
	}

	meta := jobMetaFromPayload(payload.Data, e.Source)
	if strings.TrimSpace(meta.CorrelationID) == "" {
		return
	}

	errorCount := intFromMap(payload.Data, "error_count")
	errorDetails := stringFromMap(payload.Data, "error")
	status := strings.TrimSpace(stringFromMap(payload.Data, "status"))
	if status == "" {
		switch e.Type {
		case "sync.job.queued":
			status = "queued"
		case "sync.job.start":
			status = "running"
		case "sync.job.error":
			status = "failed"
		case "sync.job.complete":
			status = "completed"
		default:
			status = "running"
		}
	}

	a.jobLogMu.Lock()
	a.jobLogJobMeta[meta.CorrelationID] = &meta
	snapshot := a.jobLogMetrics[meta.CorrelationID]
	if snapshot == nil {
		snapshot = &jobLogMetricSnapshot{}
		a.jobLogMetrics[meta.CorrelationID] = snapshot
	}
	if errorCount > snapshot.ErrorCount {
		snapshot.ErrorCount = errorCount
	}
	a.jobLogMu.Unlock()

	switch e.Type {
	case "sync.job.queued", "sync.job.start":
		a.ensureJobLogEntry(meta, status)
	case "sync.job.complete", "sync.job.error":
		if a.deferQueuedCompletion(meta, status, errorCount, errorDetails) {
			return
		}
		a.completeQueuedJobLog(meta, status, errorCount, errorDetails)
	}
}

func (a *App) recordJobLogMetricEvent(e events.Event) {
	if a.DB == nil || !a.DB.IsConnected() {
		return
	}

	source := strings.TrimSpace(e.Source)
	if source == "" {
		source = "general"
	}

	direction, tracked := metricEventDirection(e.Type)
	if !tracked {
		return
	}
	key := jobLogMetricKey(direction, source)
	correlationID := strings.TrimSpace(metricEventCorrelationID(e.Payload))
	if correlationID != "" {
		a.bindMetricTargetToCorrelation(key, correlationID, direction, source)
	}

	switch e.Type {
	case "pull.start", "pull.group.start", "push.scan.start":
		a.startOrAttachMetricTarget(key, direction, source, correlationID)
	case "pull.ids_fetched":
		payload, ok := e.Payload.(events.ResourceIDsFetchedPayload)
		if !ok {
			return
		}
		summary := fmt.Sprintf("Queued %d %s for pull", payload.Count, friendlyResourceLabel(source))
		a.updateMetricTarget(key, payload.Count, summary, "")
	case "push.scan.complete":
		payload, ok := e.Payload.(events.PushScanCompletePayload)
		if !ok {
			return
		}
		total := countChanges(payload.Changes)
		summary := fmt.Sprintf("Queued %d %s for push", total, friendlyResourceLabel(source))
		a.updateMetricTarget(key, total, summary, "")
	case "push.item.error":
		a.incrementMetricTargetErrors(key)
	case "pull.group.complete":
		payload, ok := e.Payload.(events.CompletionPayload)
		if !ok {
			return
		}
		errorCount := 0
		if target, exists := a.getMetricTarget(key); exists {
			expected := target.expectedItems
			if expected > payload.Count {
				errorCount = expected - payload.Count
			}
		}
		status := "completed"
		if !payload.Success || errorCount > 0 {
			status = "completed_with_errors"
		}
		summary := fmt.Sprintf("Pulled %d %s", payload.Count, friendlyResourceLabel(source))
		a.finalizeMetricTarget(key, status, payload.Count, errorCount, summary, "")
	case "pull.complete":
		payload, ok := e.Payload.(events.CompletionPayload)
		if !ok {
			return
		}
		count := payload.Count
		if count <= 0 && payload.Success {
			count = 1
		}
		errorCount := 0
		status := "completed"
		details := ""
		if payload.Error != nil {
			details = payload.Error.Error()
		}
		summary := fmt.Sprintf("Pulled %s", friendlyResourceLabel(source))
		if count > 0 {
			summary = fmt.Sprintf("Pulled %d %s", count, friendlyResourceLabel(source))
		}
		if !payload.Success {
			status = "failed"
			errorCount = 1
			if details != "" {
				summary = fmt.Sprintf("Pull failed for %s", friendlyResourceLabel(source))
			} else {
				summary = fmt.Sprintf("Pull finished for %s with errors", friendlyResourceLabel(source))
			}
		}
		a.finalizeMetricTarget(key, status, count, errorCount, summary, details)
	case "pull.group.error", "pull.error":
		details := ""
		if payload, ok := e.Payload.(events.ErrorPayload); ok && payload.Error != nil {
			details = payload.Error.Error()
		}
		summary := fmt.Sprintf("Pull failed for %s", friendlyResourceLabel(source))
		a.finalizeMetricTarget(key, "failed", -1, -1, summary, details)
	case "push.complete":
		payload, ok := e.Payload.(events.PushCompletePayload)
		if !ok {
			return
		}
		expected := 0
		if target, exists := a.getMetricTarget(key); exists {
			expected = target.expectedItems
		}
		if expected == 0 {
			expected = payload.ErrorCount
		}
		processed := expected
		if processed < payload.ErrorCount {
			processed = payload.ErrorCount
		}
		processed -= payload.ErrorCount
		if processed < 0 {
			processed = 0
		}
		status := "completed"
		summary := fmt.Sprintf("Push completed for %s", friendlyResourceLabel(source))
		if payload.ErrorCount > 0 {
			status = "completed_with_errors"
			summary = fmt.Sprintf("Push finished for %s with %d errors", friendlyResourceLabel(source), payload.ErrorCount)
		}
		a.finalizeMetricTarget(key, status, processed, payload.ErrorCount, summary, "")
	case "push.error":
		details := ""
		if payload, ok := e.Payload.(events.ErrorPayload); ok && payload.Error != nil {
			details = payload.Error.Error()
		}
		summary := fmt.Sprintf("Push failed for %s", friendlyResourceLabel(source))
		a.finalizeMetricTarget(key, "failed", -1, -1, summary, details)
	}
}

func (a *App) ensureJobLogEntry(meta jobLogJobMeta, status string) {
	entry := &database.JobLogEntry{
		CorrelationID:       meta.CorrelationID,
		ParentCorrelationID: meta.ParentCorrelationID,
		RootCorrelationID:   meta.RootCorrelationID,
		RunType:             meta.RunType,
		Direction:           meta.Direction,
		Source:              meta.Source,
		Initiator:           meta.Initiator,
		JobKind:             meta.JobKind,
		Mode:                meta.Mode,
		StepID:              meta.StepID,
		StepIndex:           meta.StepIndex,
		TotalSteps:          meta.TotalSteps,
		ActionType:          meta.ActionType,
		CommandText:         meta.CommandText,
		Status:              status,
		ItemsProcessed:      0,
		ErrorCount:          0,
		StartedAtTimezone:   a.ServerTimezoneLocation().String(),
		Summary:             jobSummaryForStatus(meta, status, ""),
	}

	if _, err := database.InsertJobLog(a.DB, entry); err != nil && !isDuplicateCorrelationError(err) {
		a.Events.Dispatch(events.Errorf("job_log", "Failed to create job log entry: %v", err))
	}
}

func (a *App) completeQueuedJobLog(meta jobLogJobMeta, status string, errorCount int, errorDetails string) {
	itemsProcessed := 0
	summary := ""
	details := strings.TrimSpace(errorDetails)

	a.jobLogMu.Lock()
	snapshot := a.jobLogMetrics[meta.CorrelationID]
	if snapshot == nil {
		snapshot = &jobLogMetricSnapshot{}
	}
	if meta.ParentCorrelationID == "" {
		aggItems, aggErrors := a.aggregateRootMetricsLocked(meta.CorrelationID)
		if aggItems > snapshot.ItemsProcessed {
			itemsProcessed = aggItems
		} else {
			itemsProcessed = snapshot.ItemsProcessed
		}
		if aggErrors > errorCount {
			errorCount = aggErrors
		}
	} else {
		itemsProcessed = snapshot.ItemsProcessed
	}
	if snapshot.ErrorCount > errorCount {
		errorCount = snapshot.ErrorCount
	}
	if strings.TrimSpace(snapshot.Summary) != "" {
		summary = snapshot.Summary
	}
	if strings.TrimSpace(snapshot.Details) != "" {
		if details == "" {
			details = strings.TrimSpace(snapshot.Details)
		} else {
			details = strings.TrimSpace(snapshot.Details) + " | " + details
		}
	}

	a.clearMetricTargetsLocked(meta.CorrelationID)
	delete(a.jobLogPending, meta.CorrelationID)
	delete(a.jobLogDone, meta.CorrelationID)
	if meta.ParentCorrelationID == "" {
		a.cleanupRootTrackingLocked(meta.CorrelationID)
	} else {
		delete(a.jobLogTargets, jobLogMetricKey(meta.Direction, meta.Source))
	}
	a.jobLogMu.Unlock()

	if summary == "" {
		summary = jobSummaryForStatus(meta, status, details)
	}

	if err := database.CompleteJobLog(
		a.DB,
		meta.CorrelationID,
		status,
		itemsProcessed,
		errorCount,
		a.ServerTimezoneLocation().String(),
		0,
		summary,
		details,
	); err != nil {
		if a.shouldSuppressJobLogFinalizeError(err) {
			return
		}
		a.Events.Dispatch(events.Errorf("job_log", "Failed to finalize queued job log: %v", err))
		return
	}

	a.Events.Dispatch(events.Event{Type: "sync.history.updated", Source: status})
}

func (a *App) startOrAttachMetricTarget(key, direction, source, correlationID string) {
	if a.bindMetricTargetToCorrelation(key, correlationID, direction, source) {
		return
	}

	a.jobLogMu.Lock()
	if _, exists := a.jobLogTargets[key]; exists {
		a.jobLogMu.Unlock()
		return
	}

	now := time.Now().UTC()
	fallbackCorrelationID := uuid.NewString()
	run := &jobLogRun{
		correlationID: fallbackCorrelationID,
		startedAt:     now,
		direction:     direction,
		source:        source,
	}
	a.jobLogRuns[key] = run
	a.jobLogTargets[key] = &jobLogMetricTarget{correlationID: fallbackCorrelationID, fallback: true}
	a.jobLogMetrics[fallbackCorrelationID] = &jobLogMetricSnapshot{}
	a.jobLogMu.Unlock()

	entry := &database.JobLogEntry{
		CorrelationID:     fallbackCorrelationID,
		RootCorrelationID: fallbackCorrelationID,
		RunType:           direction,
		Direction:         direction,
		Source:            source,
		Initiator:         "manual",
		JobKind:           "sync",
		Mode:              direction,
		Status:            "running",
		ItemsProcessed:    0,
		ErrorCount:        0,
		StartedAtTimezone: a.ServerTimezoneLocation().String(),
		Summary:           fmt.Sprintf("%s started for %s", strings.Title(direction), friendlyResourceLabel(source)),
	}
	if _, err := database.InsertJobLog(a.DB, entry); err != nil {
		a.Events.Dispatch(events.Errorf("job_log", "Failed to create standalone job log entry: %v", err))
	}
}

func (a *App) updateMetricTarget(key string, items int, summary, details string) {
	a.jobLogMu.Lock()
	target, exists := a.jobLogTargets[key]
	if !exists || target == nil {
		a.jobLogMu.Unlock()
		return
	}
	target.expectedItems = items
	targetSummary := strings.TrimSpace(summary)
	targetDetails := strings.TrimSpace(details)
	a.updateJobLogMetricSnapshotLocked(target.correlationID, items, target.errorCount, targetSummary, targetDetails)
	correlationID := target.correlationID
	errorCount := target.errorCount
	a.jobLogMu.Unlock()

	if err := database.UpdateJobLogMetrics(a.DB, correlationID, items, errorCount, targetSummary, targetDetails); err != nil {
		a.Events.Dispatch(events.Errorf("job_log", "Failed to update job log metrics: %v", err))
	}
}

func (a *App) incrementMetricTargetErrors(key string) {
	a.jobLogMu.Lock()
	target, exists := a.jobLogTargets[key]
	if !exists || target == nil {
		a.jobLogMu.Unlock()
		return
	}
	target.errorCount++
	items := target.expectedItems
	errorCount := target.errorCount
	correlationID := target.correlationID
	a.updateJobLogMetricSnapshotLocked(correlationID, items, errorCount, "", "")
	a.jobLogMu.Unlock()

	if err := database.UpdateJobLogMetrics(a.DB, correlationID, items, errorCount, "", ""); err != nil {
		a.Events.Dispatch(events.Errorf("job_log", "Failed to increment job log error count: %v", err))
	}
}

func (a *App) finalizeMetricTarget(key, status string, itemsProcessed, errorCount int, summary, details string) {
	a.jobLogMu.Lock()
	target, exists := a.jobLogTargets[key]
	if !exists || target == nil {
		a.jobLogMu.Unlock()
		return
	}

	if itemsProcessed < 0 {
		itemsProcessed = target.expectedItems
	}
	if itemsProcessed < 0 {
		itemsProcessed = 0
	}
	if errorCount < 0 {
		errorCount = target.errorCount
	}
	if errorCount < 0 {
		errorCount = 0
	}

	correlationID := target.correlationID
	fallback := target.fallback
	a.updateJobLogMetricSnapshotLocked(correlationID, itemsProcessed, errorCount, summary, details)
	a.jobLogDone[correlationID] = &jobLogMetricTerminal{
		Status:     strings.TrimSpace(status),
		ErrorCount: errorCount,
		Details:    strings.TrimSpace(details),
	}
	delete(a.jobLogTargets, key)
	run := a.jobLogRuns[key]
	if fallback {
		delete(a.jobLogRuns, key)
	}
	a.jobLogMu.Unlock()

	if !fallback {
		if err := database.UpdateJobLogMetrics(a.DB, correlationID, itemsProcessed, errorCount, summary, details); err != nil {
			a.Events.Dispatch(events.Errorf("job_log", "Failed to update queued job metrics: %v", err))
		}
		a.finalizePendingQueuedCompletion(correlationID)
		return
	}

	var durationSeconds int64
	if run != nil && !run.startedAt.IsZero() {
		delta := time.Since(run.startedAt)
		if delta < 0 {
			delta = 0
		}
		durationSeconds = int64(delta.Seconds())
	}

	if err := database.CompleteJobLog(
		a.DB,
		correlationID,
		status,
		itemsProcessed,
		errorCount,
		a.ServerTimezoneLocation().String(),
		durationSeconds,
		summary,
		details,
	); err != nil {
		if a.shouldSuppressJobLogFinalizeError(err) {
			return
		}
		a.Events.Dispatch(events.Errorf("job_log", "Failed to finalize standalone job log: %v", err))
		return
	}

	a.jobLogMu.Lock()
	delete(a.jobLogDone, correlationID)
	delete(a.jobLogPending, correlationID)
	a.jobLogMu.Unlock()

	a.Events.Dispatch(events.Event{Type: "sync.history.updated", Source: status})
}

func (a *App) bindMetricTargetToCorrelation(key, correlationID, direction, source string) bool {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return false
	}

	a.jobLogMu.Lock()
	target := a.jobLogTargets[key]
	if target == nil {
		target = &jobLogMetricTarget{}
		a.jobLogTargets[key] = target
	}
	if target.correlationID != correlationID {
		target.expectedItems = 0
		target.errorCount = 0
	}
	target.correlationID = correlationID
	target.fallback = false
	if _, exists := a.jobLogMetrics[correlationID]; !exists {
		a.jobLogMetrics[correlationID] = &jobLogMetricSnapshot{}
	}
	meta := a.jobLogJobMeta[correlationID]
	if meta == nil {
		meta = &jobLogJobMeta{
			CorrelationID:     correlationID,
			RootCorrelationID: correlationID,
			RunType:           "sync_job",
			Direction:         direction,
			Source:            source,
			Initiator:         "system",
			JobKind:           "sync",
			Mode:              direction,
		}
		a.jobLogJobMeta[correlationID] = meta
	} else {
		if strings.TrimSpace(meta.Direction) == "" {
			meta.Direction = direction
		}
		if strings.TrimSpace(meta.Source) == "" {
			meta.Source = source
		}
		if strings.TrimSpace(meta.RootCorrelationID) == "" {
			meta.RootCorrelationID = meta.CorrelationID
		}
	}
	a.jobLogMu.Unlock()
	return true
}

func (a *App) deferQueuedCompletion(meta jobLogJobMeta, status string, errorCount int, errorDetails string) bool {
	if !shouldWaitForMetricCompletion(meta) {
		return false
	}

	correlationID := strings.TrimSpace(meta.CorrelationID)
	if correlationID == "" {
		return false
	}

	a.jobLogMu.Lock()
	if a.jobLogPending == nil {
		a.jobLogPending = make(map[string]*jobLogPendingCompletion)
	}
	a.jobLogPending[correlationID] = &jobLogPendingCompletion{
		Status:      status,
		ErrorCount:  errorCount,
		ErrorDetail: strings.TrimSpace(errorDetails),
	}
	metricDone := a.jobLogDone[correlationID]
	a.jobLogMu.Unlock()

	if metricDone != nil {
		a.finalizePendingQueuedCompletion(correlationID)
	}
	return true
}

func shouldWaitForMetricCompletion(meta jobLogJobMeta) bool {
	if !strings.EqualFold(strings.TrimSpace(meta.JobKind), "sync") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(meta.Direction)) {
	case "pull", "push":
		return true
	default:
		return false
	}
}

func (a *App) finalizePendingQueuedCompletion(correlationID string) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return
	}

	a.jobLogMu.Lock()
	pending := a.jobLogPending[correlationID]
	meta := a.jobLogJobMeta[correlationID]
	metricDone := a.jobLogDone[correlationID]
	if pending == nil || meta == nil {
		a.jobLogMu.Unlock()
		return
	}
	pendingCopy := *pending
	metaCopy := *meta
	delete(a.jobLogPending, correlationID)
	a.jobLogMu.Unlock()

	finalStatus := strings.TrimSpace(pendingCopy.Status)
	finalErrorCount := pendingCopy.ErrorCount
	finalDetails := strings.TrimSpace(pendingCopy.ErrorDetail)
	if metricDone != nil {
		if status := strings.TrimSpace(metricDone.Status); status != "" {
			finalStatus = status
		}
		if metricDone.ErrorCount > finalErrorCount {
			finalErrorCount = metricDone.ErrorCount
		}
		metricDetails := strings.TrimSpace(metricDone.Details)
		if metricDetails != "" {
			if finalDetails == "" {
				finalDetails = metricDetails
			} else {
				finalDetails = metricDetails + " | " + finalDetails
			}
		}
	}
	a.completeQueuedJobLog(metaCopy, finalStatus, finalErrorCount, finalDetails)
}

func (a *App) getMetricTarget(key string) (*jobLogMetricTarget, bool) {
	a.jobLogMu.Lock()
	defer a.jobLogMu.Unlock()
	target, exists := a.jobLogTargets[key]
	if !exists || target == nil {
		return nil, false
	}
	copy := *target
	return &copy, true
}

func (a *App) updateJobLogMetricSnapshotLocked(correlationID string, items, errors int, summary, details string) {
	if strings.TrimSpace(correlationID) == "" {
		return
	}
	snapshot := a.jobLogMetrics[correlationID]
	if snapshot == nil {
		snapshot = &jobLogMetricSnapshot{}
		a.jobLogMetrics[correlationID] = snapshot
	}
	if items >= 0 {
		snapshot.ItemsProcessed = items
	}
	if errors >= 0 {
		snapshot.ErrorCount = errors
	}
	if strings.TrimSpace(summary) != "" {
		snapshot.Summary = strings.TrimSpace(summary)
	}
	if strings.TrimSpace(details) != "" {
		snapshot.Details = strings.TrimSpace(details)
	}
}

func (a *App) clearMetricTargetsLocked(correlationID string) {
	for key, target := range a.jobLogTargets {
		if target != nil && target.correlationID == correlationID {
			delete(a.jobLogTargets, key)
		}
	}
}

func (a *App) cleanupRootTrackingLocked(rootCorrelationID string) {
	for correlationID, meta := range a.jobLogJobMeta {
		if meta == nil {
			continue
		}
		if correlationID == rootCorrelationID || meta.RootCorrelationID == rootCorrelationID {
			delete(a.jobLogJobMeta, correlationID)
			delete(a.jobLogMetrics, correlationID)
			delete(a.jobLogPending, correlationID)
			delete(a.jobLogDone, correlationID)
		}
	}
}

func (a *App) aggregateRootMetricsLocked(rootCorrelationID string) (int, int) {
	totalItems := 0
	totalErrors := 0
	for correlationID, meta := range a.jobLogJobMeta {
		if meta == nil || strings.TrimSpace(meta.ParentCorrelationID) == "" {
			continue
		}
		if meta.RootCorrelationID != rootCorrelationID {
			continue
		}
		snapshot := a.jobLogMetrics[correlationID]
		if snapshot == nil {
			continue
		}
		totalItems += snapshot.ItemsProcessed
		totalErrors += snapshot.ErrorCount
	}
	return totalItems, totalErrors
}

func metricEventDirection(eventType events.EventType) (string, bool) {
	s := string(eventType)
	switch {
	case strings.HasPrefix(s, "pull."):
		return "pull", true
	case strings.HasPrefix(s, "push."):
		return "push", true
	default:
		return "", false
	}
}

func jobLogMetricKey(direction, source string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(direction), strings.TrimSpace(source))
}

func metricEventCorrelationID(payload events.Payload) string {
	switch typed := payload.(type) {
	case events.PullStartPayload:
		return typed.JobID
	case events.ResourceIDsFetchedPayload:
		return typed.JobID
	case events.CompletionPayload:
		return typed.JobID
	case events.ErrorPayload:
		return typed.JobID
	case events.PushScanStartPayload:
		return typed.JobID
	case events.PushScanCompletePayload:
		return typed.JobID
	case events.PushCompletePayload:
		return typed.JobID
	default:
		return ""
	}
}

func jobMetaFromPayload(data map[string]interface{}, source string) jobLogJobMeta {
	meta := jobLogJobMeta{}
	meta.CorrelationID = strings.TrimSpace(stringFromMap(data, "job_id"))
	meta.ParentCorrelationID = strings.TrimSpace(stringFromMap(data, "parent_job_id"))
	meta.RootCorrelationID = strings.TrimSpace(stringFromMap(data, "root_job_id"))
	if meta.RootCorrelationID == "" {
		meta.RootCorrelationID = meta.CorrelationID
	}
	meta.RunType = "sync_job"
	meta.Mode = strings.TrimSpace(stringFromMap(data, "mode"))
	meta.Direction = directionFromMode(meta.Mode)
	meta.Source = strings.TrimSpace(source)
	if meta.Source == "" {
		meta.Source = strings.TrimSpace(stringFromMap(data, "source"))
	}
	if meta.Source == "" {
		meta.Source = "manual"
	}
	meta.Initiator = "system"
	meta.JobKind = strings.TrimSpace(stringFromMap(data, "job_kind"))
	meta.StepID = strings.TrimSpace(stringFromMap(data, "step_id"))
	meta.StepIndex = intFromMap(data, "step_index")
	meta.TotalSteps = intFromMap(data, "total_steps")
	meta.ActionType = strings.TrimSpace(stringFromMap(data, "action_type"))
	meta.CommandText = strings.TrimSpace(stringFromMap(data, "command_text"))
	return meta
}

func jobSummaryForStatus(meta jobLogJobMeta, status string, details string) string {
	resource := strings.TrimSpace(meta.Source)
	if resource == "" {
		resource = "sync"
	}
	kind := strings.TrimSpace(meta.JobKind)
	if kind == "" {
		kind = "job"
	}
	stepName := strings.TrimSpace(meta.StepID)
	if stepName != "" {
		return fmt.Sprintf("%s %s %s", strings.Title(kind), stepName, status)
	}
	if strings.TrimSpace(details) != "" && strings.EqualFold(status, "failed") {
		return fmt.Sprintf("%s failed for %s", strings.Title(kind), resource)
	}
	return fmt.Sprintf("%s %s for %s", strings.Title(kind), status, resource)
}

func directionFromMode(mode string) string {
	trimmed := strings.ToLower(strings.TrimSpace(mode))
	switch {
	case strings.HasPrefix(trimmed, "pull"):
		return "pull"
	case strings.HasPrefix(trimmed, "push"):
		return "push"
	default:
		return "sync"
	}
}

func isDuplicateCorrelationError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}

func stringFromMap(data map[string]interface{}, key string) string {
	if data == nil {
		return ""
	}
	value, exists := data[key]
	if !exists || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case []byte:
		return string(typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func intFromMap(data map[string]interface{}, key string) int {
	if data == nil {
		return 0
	}
	value, exists := data[key]
	if !exists || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint8:
		return int(typed)
	case uint16:
		return int(typed)
	case uint32:
		return int(typed)
	case uint64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func countChanges(data any) int {
	if data == nil {
		return 0
	}
	switch v := data.(type) {
	case []database.AccountPendingChange:
		return len(v)
	case []database.CheckinPendingChange:
		return len(v)
	case []interface{}:
		return len(v)
	}

	rv := reflect.ValueOf(data)
	if !rv.IsValid() {
		return 0
	}
	if rv.Kind() == reflect.Slice {
		return rv.Len()
	}

	return 0
}

func friendlyResourceLabel(source string) string {
	if source == "" {
		return "items"
	}
	replacer := strings.NewReplacer("_", " ", "-", " ")
	label := replacer.Replace(source)
	return strings.Title(label)
}

func (a *App) shouldSuppressJobLogFinalizeError(err error) bool {
	if err == nil || !a.IsShuttingDown() {
		return false
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "database is closed") {
		return true
	}

	return strings.Contains(message, "database connection is not initialized")
}

// shouldSuppressSyncHistoryFinalizeError remains for compatibility with existing tests/callers.
func (a *App) shouldSuppressSyncHistoryFinalizeError(err error) bool {
	return a.shouldSuppressJobLogFinalizeError(err)
}
