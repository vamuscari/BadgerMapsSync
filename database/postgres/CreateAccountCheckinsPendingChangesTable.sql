CREATE TABLE IF NOT EXISTS AccountCheckinsPendingChanges (
    ChangeId SERIAL PRIMARY KEY,
    CheckinId INTEGER NOT NULL,
    AccountId INTEGER NOT NULL,
    CrmId VARCHAR(255),
    LogDatetime TIMESTAMP,
    Type VARCHAR(100),
    Comments TEXT,
    ExtraFields TEXT,
    EndpointType VARCHAR(20) NOT NULL DEFAULT 'standard' CHECK(EndpointType IN ('standard', 'custom')),
    CreatedBy VARCHAR(255),
    ChangeType VARCHAR(10) NOT NULL CHECK(ChangeType IN ('CREATE', 'UPDATE', 'DELETE')),
    Status VARCHAR(10) NOT NULL DEFAULT 'pending' CHECK(Status IN ('pending', 'processing', 'completed', 'failed')),
    CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ProcessedAt TIMESTAMP
);
