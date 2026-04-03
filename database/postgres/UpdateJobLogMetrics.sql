UPDATE "JobLog"
SET "ItemsProcessed" = $1,
    "ErrorCount" = $2,
    "Summary" = $3,
    "Details" = $4
WHERE "CorrelationId" = $5;
