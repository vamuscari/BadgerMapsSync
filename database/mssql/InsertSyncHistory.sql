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
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
