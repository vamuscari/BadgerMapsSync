UPDATE SyncHistory
SET Status = ?,
    ItemsProcessed = ?,
    ErrorCount = ?,
    CompletedAt = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    CompletedAtTimezone = ?,
    DurationSeconds = ?,
    Summary = ?,
    Details = ?
WHERE CorrelationId = ?;
