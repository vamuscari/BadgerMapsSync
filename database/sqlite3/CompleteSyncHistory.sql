UPDATE SyncHistory
SET Status = ?,
    ItemsProcessed = ?,
    ErrorCount = ?,
    CompletedAt = CURRENT_TIMESTAMP,
    CompletedAtTimezone = ?,
    DurationSeconds = ?,
    Summary = ?,
    Details = ?
WHERE CorrelationId = ?;
