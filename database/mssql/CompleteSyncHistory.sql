UPDATE SyncHistory
SET Status = @p1,
    ItemsProcessed = @p2,
    ErrorCount = @p3,
    CompletedAt = ISNULL(CompletedAt, SYSDATETIME()),
    DurationSeconds = @p4,
    Summary = @p5,
    Details = @p6
WHERE CorrelationId = @p7;
