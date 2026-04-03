UPDATE "JobLog"
SET "Status" = $1,
    "ItemsProcessed" = $2,
    "ErrorCount" = $3,
    "CompletedAt" = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    "CompletedAtTimezone" = $4,
    "DurationSeconds" = $5,
    "Summary" = $6,
    "Details" = $7
WHERE "CorrelationId" = $8;
