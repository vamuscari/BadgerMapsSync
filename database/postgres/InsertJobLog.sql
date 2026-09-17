INSERT INTO JobLog (
    CorrelationId,
    ParentCorrelationId,
    RootCorrelationId,
    RunType,
    Direction,
    Source,
    Initiator,
    JobKind,
    Mode,
    StepId,
    StepIndex,
    TotalSteps,
    ActionType,
    CommandText,
    Status,
    ItemsProcessed,
    ErrorCount,
    StartedAtTimezone,
    Summary,
    Details
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING HistoryId;
