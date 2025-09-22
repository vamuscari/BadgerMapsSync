CREATE TABLE IF NOT EXISTS SyncHistory (
    HistoryId SERIAL PRIMARY KEY,
    CorrelationId VARCHAR(64) NOT NULL UNIQUE,
    RunType VARCHAR(40) NOT NULL,
    Direction VARCHAR(40) NOT NULL,
    Source VARCHAR(80),
    Trigger VARCHAR(40),
    Status VARCHAR(40) NOT NULL,
    ItemsProcessed INTEGER DEFAULT 0,
    ErrorCount INTEGER DEFAULT 0,
    StartedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CompletedAt TIMESTAMP,
    DurationSeconds INTEGER,
    Summary TEXT,
    Details TEXT
);
