INSERT INTO SyncHistory (
    CorrelationId,
    RunType,
    Direction,
    Source,
    Initiator,
    Status,
    ItemsProcessed,
    ErrorCount,
    StartedAtTimezone,
    Summary,
    Details
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING HistoryId;
