UPDATE SyncHistory
SET Status = ?,
    ItemsProcessed = ?,
    ErrorCount = ?,
    CompletedAt = CURRENT_TIMESTAMP,
    DurationSeconds = ?,
    Summary = ?,
    Details = ?
WHERE CorrelationId = ?;
