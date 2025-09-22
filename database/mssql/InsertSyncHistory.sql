INSERT INTO SyncHistory (
    CorrelationId,
    RunType,
    Direction,
    Source,
    Initiator,
    Status,
    ItemsProcessed,
    ErrorCount,
    Summary,
    Details
) OUTPUT INSERTED.HistoryId
VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10);
