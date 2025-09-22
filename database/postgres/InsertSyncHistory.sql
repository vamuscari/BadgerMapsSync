INSERT INTO "SyncHistory" (
    "CorrelationId",
    "RunType",
    "Direction",
    "Source",
    "Initiator",
    "Status",
    "ItemsProcessed",
    "ErrorCount",
    "Summary",
    "Details"
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING "HistoryId";
