IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='SyncHistory' AND xtype='U')
CREATE TABLE SyncHistory (
    HistoryId INT IDENTITY(1,1) PRIMARY KEY,
    CorrelationId NVARCHAR(64) NOT NULL UNIQUE,
    RunType NVARCHAR(40) NOT NULL,
    Direction NVARCHAR(40) NOT NULL,
    Source NVARCHAR(80),
    Trigger NVARCHAR(40),
    Status NVARCHAR(40) NOT NULL,
    ItemsProcessed INT DEFAULT 0,
    ErrorCount INT DEFAULT 0,
    StartedAt DATETIME2 DEFAULT SYSUTCDATETIME(),
    CompletedAt DATETIME2,
    DurationSeconds INT,
    Summary NVARCHAR(MAX),
    Details NVARCHAR(MAX)
);
