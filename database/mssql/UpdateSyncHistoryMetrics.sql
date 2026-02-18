UPDATE SyncHistory
SET ItemsProcessed = ?,
    Summary = ?
WHERE CorrelationId = ?;
