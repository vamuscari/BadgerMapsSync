INSERT INTO "JobLog" (
    "CorrelationId",
    "ParentCorrelationId",
    "RootCorrelationId",
    "RunType",
    "Direction",
    "Source",
    "Initiator",
    "JobKind",
    "Mode",
    "StepId",
    "StepIndex",
    "TotalSteps",
    "ActionType",
    "CommandText",
    "Status",
    "ItemsProcessed",
    "ErrorCount",
    "StartedAtTimezone",
    "Summary",
    "Details"
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
RETURNING "HistoryId";
