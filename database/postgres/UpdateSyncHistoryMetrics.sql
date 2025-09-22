UPDATE "SyncHistory"
SET "ItemsProcessed" = $1,
    "Summary" = $2
WHERE "CorrelationId" = $3;
