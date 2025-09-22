package database

import (
	"database/sql"
	"fmt"
	"time"
)

// SyncHistoryEntry represents a row in the SyncHistory table.
type SyncHistoryEntry struct {
	HistoryID       int64
	CorrelationID   string
	RunType         string
	Direction       string
	Source          string
	Trigger         string
	Status          string
	ItemsProcessed  int
	ErrorCount      int
	StartedAt       time.Time
	CompletedAt     *time.Time
	DurationSeconds int
	Summary         string
	Details         string
}

// InsertSyncHistory creates a new sync history record and returns the new history ID.
func InsertSyncHistory(db DB, entry *SyncHistoryEntry) (int64, error) {
	if db == nil || db.GetDB() == nil {
		return 0, fmt.Errorf("database connection is not initialized")
	}
	if entry == nil {
		return 0, fmt.Errorf("sync history entry is nil")
	}
	if entry.CorrelationID == "" {
		return 0, fmt.Errorf("sync history entry requires correlation id")
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

	sqlDB := db.GetDB()
	args := []any{
		entry.CorrelationID,
		entry.RunType,
		entry.Direction,
		entry.Source,
		entry.Trigger,
		entry.Status,
		entry.ItemsProcessed,
		entry.ErrorCount,
		entry.Summary,
		entry.Details,
	}

	var (
		id  int64
		err error
	)

	switch db.GetType() {
	case "postgres":
		query := `INSERT INTO "SyncHistory" ("CorrelationId", "RunType", "Direction", "Source", "Trigger", "Status", "ItemsProcessed", "ErrorCount", "Summary", "Details")
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING "HistoryId"`
		err = sqlDB.QueryRow(query, args...).Scan(&id)
	case "mssql":
		query := `INSERT INTO SyncHistory (CorrelationId, RunType, Direction, Source, Trigger, Status, ItemsProcessed, ErrorCount, Summary, Details)
			OUTPUT INSERTED.HistoryId VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10)`
		err = sqlDB.QueryRow(query, args...).Scan(&id)
	default:
		query := `INSERT INTO SyncHistory (CorrelationId, RunType, Direction, Source, Trigger, Status, ItemsProcessed, ErrorCount, Summary, Details)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		result, execErr := sqlDB.Exec(query, args...)
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

// UpdateSyncHistoryMetrics updates the progress-related fields for a sync run.
func UpdateSyncHistoryMetrics(db DB, correlationID string, itemsProcessed int, summary string) error {
	if db == nil || db.GetDB() == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	sqlDB := db.GetDB()

	switch db.GetType() {
	case "postgres":
		query := `UPDATE "SyncHistory" SET "ItemsProcessed" = $1, "Summary" = $2 WHERE "CorrelationId" = $3`
		_, err := sqlDB.Exec(query, itemsProcessed, summary, correlationID)
		return err
	case "mssql":
		query := `UPDATE SyncHistory SET ItemsProcessed = @p1, Summary = @p2 WHERE CorrelationId = @p3`
		_, err := sqlDB.Exec(query, itemsProcessed, summary, correlationID)
		return err
	default:
		query := `UPDATE SyncHistory SET ItemsProcessed = ?, Summary = ? WHERE CorrelationId = ?`
		_, err := sqlDB.Exec(query, itemsProcessed, summary, correlationID)
		return err
	}
}

// CompleteSyncHistory finalizes a sync history record with completion details.
func CompleteSyncHistory(db DB, correlationID, status string, itemsProcessed, errorCount int, durationSeconds int64, summary, details string) error {
	if db == nil || db.GetDB() == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	sqlDB := db.GetDB()

	switch db.GetType() {
	case "postgres":
		query := `UPDATE "SyncHistory"
			SET "Status" = $1,
			    "ItemsProcessed" = $2,
			    "ErrorCount" = $3,
			    "CompletedAt" = COALESCE("CompletedAt", CURRENT_TIMESTAMP),
			    "DurationSeconds" = $4,
			    "Summary" = $5,
			    "Details" = $6
			WHERE "CorrelationId" = $7`
		_, err := sqlDB.Exec(query, status, itemsProcessed, errorCount, durationSeconds, summary, details, correlationID)
		return err
	case "mssql":
		query := `UPDATE SyncHistory
			SET Status = @p1,
			    ItemsProcessed = @p2,
			    ErrorCount = @p3,
			    CompletedAt = ISNULL(CompletedAt, SYSDATETIME()),
			    DurationSeconds = @p4,
			    Summary = @p5,
			    Details = @p6
			WHERE CorrelationId = @p7`
		_, err := sqlDB.Exec(query, status, itemsProcessed, errorCount, durationSeconds, summary, details, correlationID)
		return err
	default:
		query := `UPDATE SyncHistory
			SET Status = ?,
			    ItemsProcessed = ?,
			    ErrorCount = ?,
			    CompletedAt = COALESCE(CompletedAt, CURRENT_TIMESTAMP),
			    DurationSeconds = ?,
			    Summary = ?,
			    Details = ?
			WHERE CorrelationId = ?`
		_, err := sqlDB.Exec(query, status, itemsProcessed, errorCount, durationSeconds, summary, details, correlationID)
		return err
	}
}

// GetRecentSyncHistory returns the most recent sync history entries up to the provided limit.
func GetRecentSyncHistory(db DB, limit int) ([]SyncHistoryEntry, error) {
	if db == nil || db.GetDB() == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}
	if limit <= 0 {
		limit = 20
	}

	sqlDB := db.GetDB()

	var query string
	switch db.GetType() {
	case "postgres":
		query = fmt.Sprintf(`SELECT HistoryId, CorrelationId, RunType, Direction, Source, Trigger, Status, ItemsProcessed, ErrorCount, StartedAt, CompletedAt, DurationSeconds, Summary, Details
			FROM SyncHistory ORDER BY StartedAt DESC LIMIT %d`, limit)
	case "mssql":
		query = fmt.Sprintf(`SELECT TOP (%d) HistoryId, CorrelationId, RunType, Direction, Source, Trigger, Status, ItemsProcessed, ErrorCount, StartedAt, CompletedAt, DurationSeconds, Summary, Details
			FROM SyncHistory ORDER BY StartedAt DESC`, limit)
	default:
		query = fmt.Sprintf(`SELECT HistoryId, CorrelationId, RunType, Direction, Source, Trigger, Status, ItemsProcessed, ErrorCount, StartedAt, CompletedAt, DurationSeconds, Summary, Details
			FROM SyncHistory ORDER BY StartedAt DESC LIMIT %d`, limit)
	}

	rows, err := sqlDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []SyncHistoryEntry
	for rows.Next() {
		var (
			entry     SyncHistoryEntry
			started   any
			completed any
			duration  sql.NullInt64
		)

		if err := rows.Scan(
			&entry.HistoryID,
			&entry.CorrelationID,
			&entry.RunType,
			&entry.Direction,
			&entry.Source,
			&entry.Trigger,
			&entry.Status,
			&entry.ItemsProcessed,
			&entry.ErrorCount,
			&started,
			&completed,
			&duration,
			&entry.Summary,
			&entry.Details,
		); err != nil {
			return nil, err
		}

		entry.StartedAt = normaliseToTime(started)
		if completedTime := normaliseToNullableTime(completed); completedTime != nil {
			entry.CompletedAt = completedTime
		}
		if duration.Valid {
			entry.DurationSeconds = int(duration.Int64)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func normaliseToTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case *time.Time:
		if v != nil {
			return *v
		}
	case string:
		if t, err := parseTimeString(v); err == nil {
			return t
		}
	case []byte:
		if t, err := parseTimeString(string(v)); err == nil {
			return t
		}
	case sql.NullTime:
		if v.Valid {
			return v.Time
		}
	case sql.NullString:
		if v.Valid {
			if t, err := parseTimeString(v.String); err == nil {
				return t
			}
		}
	case int64:
		return time.Unix(v, 0).UTC()
	case float64:
		return time.Unix(int64(v), 0).UTC()
	}
	return time.Time{}
}

func normaliseToNullableTime(value any) *time.Time {
	t := normaliseToTime(value)
	if t.IsZero() {
		return nil
	}
	return &t
}

func parseTimeString(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", value)
}
