package database

import (
	"database/sql"
	"fmt"
	"time"
)

type AccountPendingChange struct {
	ChangeId    int
	AccountId   int
	ChangeType  string
	Changes     string
	Status      string
	CreatedAt   time.Time
	ProcessedAt sql.NullTime
}

type CheckinPendingChange struct {
	ChangeId    int
	CheckinId   int
	ChangeType  string
	Changes     string
	Status      string
	CreatedAt   time.Time
	ProcessedAt sql.NullTime
}

func GetPendingAccountChanges(db DB) ([]AccountPendingChange, error) {
	sqlText := sqlCommandLoader(db.GetType(), "get_pending_account_changes")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: get_pending_account_changes")
	}

	sqlDB, err := sql.Open(db.GetType(), db.DatabaseConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	rows, err := sqlDB.Query(sqlText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []AccountPendingChange
	for rows.Next() {
		var change AccountPendingChange
		if err := rows.Scan(&change.ChangeId, &change.AccountId, &change.ChangeType, &change.Changes, &change.Status, &change.CreatedAt, &change.ProcessedAt); err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func GetPendingCheckinChanges(db DB) ([]CheckinPendingChange, error) {
	sqlText := sqlCommandLoader(db.GetType(), "get_pending_checkin_changes")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: get_pending_checkin_changes")
	}

	sqlDB, err := sql.Open(db.GetType(), db.DatabaseConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	rows, err := sqlDB.Query(sqlText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []CheckinPendingChange
	for rows.Next() {
		var change CheckinPendingChange
		if err := rows.Scan(&change.ChangeId, &change.CheckinId, &change.ChangeType, &change.Changes, &change.Status, &change.CreatedAt, &change.ProcessedAt); err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func UpdatePendingChangeStatus(db DB, table string, changeId int, status string) error {
	sqlText := fmt.Sprintf(sqlCommandLoader(db.GetType(), "update_pending_change_status"), table)
	if sqlText == "" {
		return fmt.Errorf("unknown or unavailable SQL command: update_pending_change_status")
	}

	sqlDB, err := sql.Open(db.GetType(), db.DatabaseConnection())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	_, err = sqlDB.Exec(sqlText, status, changeId)
	return err
}
