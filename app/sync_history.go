package app

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"

	"badgermaps/database"
	"badgermaps/events"
)

type syncHistoryRun struct {
	correlationID string
	startedAt     time.Time
	runType       string
	direction     string
	source        string
	expectedItems int
	errorCount    int
}

func (a *App) ensureSyncHistoryTracking() {
	if a.Events == nil || a.DB == nil || !a.DB.IsConnected() {
		return
	}

	a.syncHistoryMu.Lock()
	if a.syncHistoryRuns == nil {
		a.syncHistoryRuns = make(map[string]*syncHistoryRun)
	}
	if a.syncHistoryOnce {
		a.syncHistoryMu.Unlock()
		return
	}
	a.syncHistoryOnce = true
	a.syncHistoryMu.Unlock()

	a.Events.Subscribe("pull.*", a.recordSyncHistoryEvent)
	a.Events.Subscribe("push.*", a.recordSyncHistoryEvent)
}

func (a *App) recordSyncHistoryEvent(e events.Event) {
	if a.DB == nil || !a.DB.IsConnected() {
		return
	}

	source := e.Source
	if source == "" {
		source = "general"
	}

	switch e.Type {
	case "pull.group.start":
		a.startSyncHistoryRun(syncHistoryKey("pull", source), "pull", "pull", source, fmt.Sprintf("Pull started for %s", friendlyResourceLabel(source)))
	case "pull.ids_fetched":
		if payload, ok := e.Payload.(events.ResourceIDsFetchedPayload); ok {
			key := syncHistoryKey("pull", source)
			summary := fmt.Sprintf("Queued %d %s for pull", payload.Count, friendlyResourceLabel(source))
			a.updateSyncHistoryMetrics(key, payload.Count, summary)
		}
	case "pull.group.complete":
		if payload, ok := e.Payload.(events.CompletionPayload); ok {
			key := syncHistoryKey("pull", source)
			errorCount := 0
			expected := a.syncHistoryExpected(key)
			if expected > payload.Count {
				errorCount = expected - payload.Count
			}
			status := "completed"
			if !payload.Success || errorCount > 0 {
				status = "completed_with_errors"
			}
			summary := fmt.Sprintf("Pulled %d %s", payload.Count, friendlyResourceLabel(source))
			a.completeSyncHistoryRun(key, status, payload.Count, errorCount, summary, "")
		}
	case "pull.group.error":
		key := syncHistoryKey("pull", source)
		details := ""
		if payload, ok := e.Payload.(events.ErrorPayload); ok && payload.Error != nil {
			details = payload.Error.Error()
		}
		summary := fmt.Sprintf("Pull failed for %s", friendlyResourceLabel(source))
		a.completeSyncHistoryRun(key, "failed", -1, -1, summary, details)
	case "push.scan.start":
		a.startSyncHistoryRun(syncHistoryKey("push", source), "push", "push", source, fmt.Sprintf("Scanning %s pending changes", friendlyResourceLabel(source)))
	case "push.scan.complete":
		if payload, ok := e.Payload.(events.PushScanCompletePayload); ok {
			total := countChanges(payload.Changes)
			key := syncHistoryKey("push", source)
			summary := fmt.Sprintf("Queued %d %s for push", total, friendlyResourceLabel(source))
			a.updateSyncHistoryMetrics(key, total, summary)
		}
	case "push.complete":
		if payload, ok := e.Payload.(events.PushCompletePayload); ok {
			key := syncHistoryKey("push", source)
			expected := a.syncHistoryExpected(key)
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
			a.completeSyncHistoryRun(key, status, processed, payload.ErrorCount, summary, "")
		}
	case "push.error":
		key := syncHistoryKey("push", source)
		details := ""
		if payload, ok := e.Payload.(events.ErrorPayload); ok && payload.Error != nil {
			details = payload.Error.Error()
		}
		summary := fmt.Sprintf("Push failed for %s", friendlyResourceLabel(source))
		a.completeSyncHistoryRun(key, "failed", -1, -1, summary, details)
	case "push.item.error":
		key := syncHistoryKey("push", source)
		a.incrementSyncHistoryErrors(key)
	}
}

func (a *App) startSyncHistoryRun(key, runType, direction, source, summary string) {
	if key == "" {
		return
	}

	a.syncHistoryMu.Lock()
	if existing := a.syncHistoryRuns[key]; existing != nil {
		// Prevent overlapping runs with the same key
		a.syncHistoryMu.Unlock()
		return
	}
	a.syncHistoryMu.Unlock()

	run := &syncHistoryRun{
		correlationID: uuid.NewString(),
		startedAt:     time.Now().UTC(),
		runType:       runType,
		direction:     direction,
		source:        source,
	}

	entry := &database.SyncHistoryEntry{
		CorrelationID:  run.correlationID,
		RunType:        runType,
		Direction:      direction,
		Source:         source,
		Initiator:      "manual",
		Status:         "running",
		ItemsProcessed: 0,
		ErrorCount:     0,
		Summary:        summary,
	}

	if _, err := database.InsertSyncHistory(a.DB, entry); err != nil {
		a.Events.Dispatch(events.Errorf("sync_history", "Failed to record sync start: %v", err))
		return
	}

	a.syncHistoryMu.Lock()
	a.syncHistoryRuns[key] = run
	a.syncHistoryMu.Unlock()
}

func (a *App) updateSyncHistoryMetrics(key string, items int, summary string) {
	if key == "" {
		return
	}

	var correlationID string
	a.syncHistoryMu.Lock()
	run, ok := a.syncHistoryRuns[key]
	if ok {
		run.expectedItems = items
		runSummary := summary
		if runSummary == "" {
			runSummary = fmt.Sprintf("Processed %d items", items)
		}
		summary = runSummary
		correlationID = run.correlationID
	}
	a.syncHistoryMu.Unlock()

	if !ok {
		return
	}

	if err := database.UpdateSyncHistoryMetrics(a.DB, correlationID, items, summary); err != nil {
		a.Events.Dispatch(events.Errorf("sync_history", "Failed to update sync metrics: %v", err))
	}
}

func (a *App) completeSyncHistoryRun(key, status string, itemsProcessed, errorCount int, summary, details string) {
	if key == "" {
		return
	}

	var (
		correlationID string
		startedAt     time.Time
		expected      int
		savedErrors   int
	)

	a.syncHistoryMu.Lock()
	run, ok := a.syncHistoryRuns[key]
	if ok {
		correlationID = run.correlationID
		startedAt = run.startedAt
		expected = run.expectedItems
		savedErrors = run.errorCount
		delete(a.syncHistoryRuns, key)
	}
	a.syncHistoryMu.Unlock()

	if !ok {
		return
	}

	if itemsProcessed < 0 {
		itemsProcessed = expected
	}
	if itemsProcessed < 0 {
		itemsProcessed = 0
	}
	if errorCount < 0 {
		errorCount = savedErrors
	}
	if errorCount < 0 {
		errorCount = 0
	}

	var durationSeconds int64
	if !startedAt.IsZero() {
		delta := time.Since(startedAt)
		if delta < 0 {
			delta = 0
		}
		durationSeconds = int64(delta.Seconds())
	}

	if summary == "" {
		summary = fmt.Sprintf("Sync %s", status)
	}

	if err := database.CompleteSyncHistory(a.DB, correlationID, status, itemsProcessed, errorCount, durationSeconds, summary, details); err != nil {
		a.Events.Dispatch(events.Errorf("sync_history", "Failed to finalize sync history: %v", err))
		return
	}

	a.Events.Dispatch(events.Event{Type: "sync.history.updated", Source: status})
}

func (a *App) incrementSyncHistoryErrors(key string) {
	a.syncHistoryMu.Lock()
	if run, ok := a.syncHistoryRuns[key]; ok {
		run.errorCount++
	}
	a.syncHistoryMu.Unlock()
}

func (a *App) syncHistoryExpected(key string) int {
	a.syncHistoryMu.Lock()
	defer a.syncHistoryMu.Unlock()
	if run, ok := a.syncHistoryRuns[key]; ok {
		return run.expectedItems
	}
	return 0
}

func syncHistoryKey(direction, source string) string {
	return fmt.Sprintf("%s:%s", direction, source)
}

func friendlyResourceLabel(source string) string {
	if source == "" {
		return "items"
	}
	replacer := strings.NewReplacer("_", " ", "-", " ")
	label := replacer.Replace(source)
	return strings.Title(label)
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
