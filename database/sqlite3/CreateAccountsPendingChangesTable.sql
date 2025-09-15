CREATE TABLE IF NOT EXISTS AccountsPendingChanges (
    ChangeId INTEGER PRIMARY KEY AUTOINCREMENT,
    AccountId INTEGER NOT NULL,
    ChangeType TEXT NOT NULL CHECK(ChangeType IN ('CREATE', 'UPDATE', 'DELETE')),
    Changes TEXT, -- JSON object with field changes, e.g., {"PhoneNumber": "123-456-7890", "Notes": "New notes"}
    Status TEXT NOT NULL DEFAULT 'pending' CHECK(Status IN ('pending', 'processing', 'completed', 'failed')),
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    ProcessedAt DATETIME
);
