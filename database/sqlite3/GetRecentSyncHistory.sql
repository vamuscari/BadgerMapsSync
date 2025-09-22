SELECT HistoryId,
       CorrelationId,
       RunType,
       Direction,
       Source,
       Initiator,
       Status,
       ItemsProcessed,
       ErrorCount,
       StartedAt,
       CompletedAt,
       DurationSeconds,
       Summary,
       Details
FROM SyncHistory
ORDER BY StartedAt DESC
LIMIT ?;
