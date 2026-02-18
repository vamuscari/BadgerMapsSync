CREATE TABLE IF NOT EXISTS AccountCheckinsPendingChanges (
    ChangeId INTEGER PRIMARY KEY AUTOINCREMENT,
    CheckinId INTEGER NOT NULL,
    AccountId INTEGER NOT NULL,
    CrmId TEXT,
    LogDatetime TEXT,
    Type TEXT,
    Comments TEXT,
    ExtraFields TEXT,
    EndpointType TEXT NOT NULL DEFAULT 'standard' CHECK(EndpointType IN ('standard', 'custom')),
    CreatedBy TEXT,
    ChangeType TEXT NOT NULL CHECK(ChangeType IN ('CREATE', 'UPDATE', 'DELETE')),
    Status TEXT NOT NULL DEFAULT 'pending' CHECK(Status IN ('pending', 'processing', 'completed', 'failed')),
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    ProcessedAt DATETIME
);
