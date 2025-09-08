CREATE TABLE IF NOT EXISTS AccountCheckinsPendingChanges (
    ChangeId SERIAL PRIMARY KEY,
    CheckinId INTEGER NOT NULL,
    ChangeType VARCHAR(10) NOT NULL CHECK(ChangeType IN ('CREATE', 'UPDATE', 'DELETE')),
    Changes TEXT,
    Status VARCHAR(10) NOT NULL DEFAULT 'pending' CHECK(Status IN ('pending', 'processing', 'completed', 'failed')),
    CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ProcessedAt TIMESTAMP
);
