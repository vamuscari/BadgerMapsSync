package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// JobLogEntry represents a row in the JobLog table.
type JobLogEntry struct {
	HistoryID           int64
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
	Status              string
	ItemsProcessed      int
	ErrorCount          int
	StartedAt           time.Time
	StartedAtTimezone   string
	CompletedAt         *time.Time
	CompletedAtTimezone string
	DurationSeconds     int
	Summary             string
	Details             string
}

func InsertJobLog(db DB, entry *JobLogEntry) (int64, error) {
	if db == nil || db.GetDB() == nil {
		return 0, fmt.Errorf("database connection is not initialized")
	}
	if entry == nil {
		return 0, fmt.Errorf("job log entry is nil")
	}
	entry.CorrelationID = strings.TrimSpace(entry.CorrelationID)
	if entry.CorrelationID == "" {
		return 0, fmt.Errorf("job log entry requires correlation id")
	}
	entry.ParentCorrelationID = strings.TrimSpace(entry.ParentCorrelationID)
	entry.RootCorrelationID = strings.TrimSpace(entry.RootCorrelationID)
	entry.RunType = strings.TrimSpace(entry.RunType)
	entry.Direction = strings.TrimSpace(entry.Direction)
	entry.Source = strings.TrimSpace(entry.Source)
	entry.Initiator = strings.TrimSpace(entry.Initiator)
	entry.JobKind = strings.TrimSpace(entry.JobKind)
	entry.Mode = strings.TrimSpace(entry.Mode)
	entry.StepID = strings.TrimSpace(entry.StepID)
	entry.ActionType = strings.TrimSpace(entry.ActionType)
	entry.CommandText = strings.TrimSpace(entry.CommandText)
	entry.Status = strings.TrimSpace(entry.Status)
	entry.Summary = strings.TrimSpace(entry.Summary)
	entry.Details = strings.TrimSpace(entry.Details)
	if entry.RootCorrelationID == "" {
		entry.RootCorrelationID = entry.CorrelationID
	}
	if entry.RunType == "" {
		entry.RunType = "unknown"
	}
	if entry.Direction == "" {
		entry.Direction = entry.RunType
	}
	if entry.Status == "" {
		entry.Status = "running"
	}
	entry.StartedAtTimezone = normalizeSyncHistoryTimezone(entry.StartedAtTimezone)

	sqlText := db.GetSQL("InsertJobLog")
	if sqlText == "" {
		return 0, fmt.Errorf("unknown or unavailable SQL command: InsertJobLog")
	}

	sqlDB := db.GetDB()
	args := []any{
		entry.CorrelationID,
		nullableTrimmed(entry.ParentCorrelationID),
		nullableTrimmed(entry.RootCorrelationID),
		entry.RunType,
		entry.Direction,
		nullableTrimmed(entry.Source),
		nullableTrimmed(entry.Initiator),
		nullableTrimmed(entry.JobKind),
		nullableTrimmed(entry.Mode),
		nullableTrimmed(entry.StepID),
		entry.StepIndex,
		entry.TotalSteps,
		nullableTrimmed(entry.ActionType),
		nullableTrimmed(entry.CommandText),
		entry.Status,
		entry.ItemsProcessed,
		entry.ErrorCount,
		entry.StartedAtTimezone,
		nullableTrimmed(entry.Summary),
		nullableTrimmed(entry.Details),
	}

	var (
		id  int64
		err error
	)

	switch db.GetType() {
	case "postgres", "mssql":
		err = sqlDB.QueryRow(sqlText, args...).Scan(&id)
	default:
		result, execErr := sqlDB.Exec(sqlText, args...)
		if execErr != nil {
			return 0, execErr
		}
		id, err = result.LastInsertId()
	}

	if err != nil {
		return 0, err
	}

	entry.HistoryID = id
	return id, nil
}

func UpdateJobLogMetrics(db DB, correlationID string, itemsProcessed, errorCount int, summary, details string) error {
	if db == nil || db.GetDB() == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	sqlText := db.GetSQL("UpdateJobLogMetrics")
	if sqlText == "" {
		return fmt.Errorf("unknown or unavailable SQL command: UpdateJobLogMetrics")
	}

	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return fmt.Errorf("correlation id is required")
	}

	_, err := db.GetDB().Exec(sqlText, itemsProcessed, errorCount, summary, details, correlationID)
	return err
}

func CompleteJobLog(db DB, correlationID, status string, itemsProcessed, errorCount int, completedAtTZ string, durationSeconds int64, summary, details string) error {
	if db == nil || db.GetDB() == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	sqlText := db.GetSQL("CompleteJobLog")
	if sqlText == "" {
		return fmt.Errorf("unknown or unavailable SQL command: CompleteJobLog")
	}

	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return fmt.Errorf("correlation id is required")
	}

	if status == "" {
		status = "completed"
	}
	completedAtTZ = normalizeSyncHistoryTimezone(completedAtTZ)

	_, err := db.GetDB().Exec(
		sqlText,
		status,
		itemsProcessed,
		errorCount,
		completedAtTZ,
		durationSeconds,
		summary,
		details,
		correlationID,
	)
	return err
}

func GetRecentJobLog(db DB, limit int) ([]JobLogEntry, error) {
	if db == nil || db.GetDB() == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}
	if limit <= 0 {
		limit = 20
	}

	sqlText := db.GetSQL("GetRecentJobLog")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: GetRecentJobLog")
	}
	sqlText = strings.Replace(sqlText, "{{LIMIT}}", strconv.Itoa(limit), 1)

	rows, err := db.GetDB().Query(sqlText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]JobLogEntry, 0)
	for rows.Next() {
		var (
			entry               JobLogEntry
			parentCorrelationID sql.NullString
			rootCorrelationID   sql.NullString
			source              sql.NullString
			initiator           sql.NullString
			jobKind             sql.NullString
			mode                sql.NullString
			stepID              sql.NullString
			actionType          sql.NullString
			commandText         sql.NullString
			started             any
			completed           any
			startedTZ           sql.NullString
			endedTZ             sql.NullString
			duration            sql.NullInt64
			summary             sql.NullString
			details             sql.NullString
		)

		if err := rows.Scan(
			&entry.HistoryID,
			&entry.CorrelationID,
			&parentCorrelationID,
			&rootCorrelationID,
			&entry.RunType,
			&entry.Direction,
			&source,
			&initiator,
			&jobKind,
			&mode,
			&stepID,
			&entry.StepIndex,
			&entry.TotalSteps,
			&actionType,
			&commandText,
			&entry.Status,
			&entry.ItemsProcessed,
			&entry.ErrorCount,
			&started,
			&startedTZ,
			&completed,
			&endedTZ,
			&duration,
			&summary,
			&details,
		); err != nil {
			return nil, err
		}

		entry.ParentCorrelationID = strings.TrimSpace(parentCorrelationID.String)
		entry.RootCorrelationID = strings.TrimSpace(rootCorrelationID.String)
		entry.Source = strings.TrimSpace(source.String)
		entry.Initiator = strings.TrimSpace(initiator.String)
		entry.JobKind = strings.TrimSpace(jobKind.String)
		entry.Mode = strings.TrimSpace(mode.String)
		entry.StepID = strings.TrimSpace(stepID.String)
		entry.ActionType = strings.TrimSpace(actionType.String)
		entry.CommandText = strings.TrimSpace(commandText.String)
		entry.StartedAt = normaliseToTime(started)
		entry.StartedAtTimezone = normalizeSyncHistoryTimezone(startedTZ.String)
		if completedTime := normaliseToNullableTime(completed); completedTime != nil {
			entry.CompletedAt = completedTime
		}
		entry.CompletedAtTimezone = strings.TrimSpace(endedTZ.String)
		if entry.CompletedAt != nil && entry.CompletedAtTimezone == "" {
			entry.CompletedAtTimezone = "UTC"
		}
		if duration.Valid {
			entry.DurationSeconds = int(duration.Int64)
		}
		entry.Summary = strings.TrimSpace(summary.String)
		entry.Details = strings.TrimSpace(details.String)
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func BackfillJobLogFromSyncHistory(db DB) (int64, error) {
	if db == nil || db.GetDB() == nil {
		return 0, fmt.Errorf("database connection is not initialized")
	}
	exists, err := db.TableExists("SyncHistory")
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, nil
	}

	sqlText := db.GetSQL("BackfillJobLogFromSyncHistory")
	if sqlText == "" {
		return 0, fmt.Errorf("unknown or unavailable SQL command: BackfillJobLogFromSyncHistory")
	}

	result, err := db.GetDB().Exec(sqlText)
	if err != nil {
		return 0, err
	}
	rowsAffected, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return 0, nil
	}
	return rowsAffected, nil
}

func EnsureJobLogSetup(db DB) error {
	if db == nil || db.GetDB() == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	if err := RunCommand(db, "CreateJobLogTable"); err != nil {
		return fmt.Errorf("failed to ensure JobLog table: %w", err)
	}
	if err := RunCommand(db, "CreateJobLogIndexes"); err != nil {
		return fmt.Errorf("failed to ensure JobLog indexes: %w", err)
	}
	if _, err := BackfillJobLogFromSyncHistory(db); err != nil {
		return fmt.Errorf("failed to backfill JobLog: %w", err)
	}
	return nil
}

func nullableTrimmed(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
