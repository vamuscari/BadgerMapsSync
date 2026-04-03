UPDATE JobLog
SET ItemsProcessed = ?,
    ErrorCount = ?,
    Summary = ?,
    Details = ?
WHERE CorrelationId = ?;
