UPDATE SyncHistory
SET ItemsProcessed = @p1,
    Summary = @p2
WHERE CorrelationId = @p3;
