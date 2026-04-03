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
       StartedAtTimezone,
       CompletedAt,
       CompletedAtTimezone,
       DurationSeconds,
       Summary,
       Details
FROM SyncHistory
ORDER BY StartedAt DESC
OFFSET 0 ROWS FETCH NEXT {{LIMIT}} ROWS ONLY;
