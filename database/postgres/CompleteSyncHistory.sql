UPDATE "SyncHistory"
SET "Status" = $1,
    "ItemsProcessed" = $2,
    "ErrorCount" = $3,
    "CompletedAt" = COALESCE("CompletedAt", CURRENT_TIMESTAMP),
    "DurationSeconds" = $4,
    "Summary" = $5,
    "Details" = $6
WHERE "CorrelationId" = $7;
