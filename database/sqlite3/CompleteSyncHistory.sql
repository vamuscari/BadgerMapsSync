UPDATE SyncHistory
SET Status = ?,
    ItemsProcessed = ?,
    ErrorCount = ?,
    CompletedAt = COALESCE(CompletedAt, CURRENT_TIMESTAMP),
    DurationSeconds = ?,
    Summary = ?,
    Details = ?
WHERE CorrelationId = ?;
