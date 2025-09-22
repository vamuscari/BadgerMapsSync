CREATE TABLE IF NOT EXISTS SyncHistory (
    HistoryId INTEGER PRIMARY KEY AUTOINCREMENT,
    CorrelationId TEXT NOT NULL UNIQUE,
    RunType TEXT NOT NULL,
    Direction TEXT NOT NULL,
    Source TEXT,
    Initiator TEXT,
    Status TEXT NOT NULL,
    ItemsProcessed INTEGER DEFAULT 0,
    ErrorCount INTEGER DEFAULT 0,
    StartedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    CompletedAt DATETIME,
    DurationSeconds INTEGER,
    Summary TEXT,
    Details TEXT
);
