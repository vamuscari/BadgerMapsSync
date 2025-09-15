CREATE TABLE IF NOT EXISTS AccountCheckinsPendingChanges (
    ChangeId INTEGER PRIMARY KEY AUTOINCREMENT,
    CheckinId INTEGER NOT NULL,
    AccountId INTEGER,
    ChangeType TEXT NOT NULL CHECK(ChangeType IN ('CREATE', 'UPDATE', 'DELETE')),
    Changes TEXT,
    Status TEXT NOT NULL DEFAULT 'pending' CHECK(Status IN ('pending', 'processing', 'completed', 'failed')),
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    ProcessedAt DATETIME
);
