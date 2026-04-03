UPDATE JobLog
SET Status = ?,
    ItemsProcessed = ?,
    ErrorCount = ?,
    CompletedAt = SYSUTCDATETIME(),
    CompletedAtTimezone = ?,
    DurationSeconds = ?,
    Summary = ?,
    Details = ?
WHERE CorrelationId = ?;
