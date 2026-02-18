IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='AccountCheckinsPendingChanges' AND xtype='U')
CREATE TABLE AccountCheckinsPendingChanges (
    ChangeId INT IDENTITY(1,1) PRIMARY KEY,
    CheckinId INT NOT NULL,
    AccountId INT NOT NULL,
    CrmId NVARCHAR(255),
    LogDatetime DATETIME2,
    Type NVARCHAR(100),
    Comments NVARCHAR(MAX),
    ExtraFields NVARCHAR(MAX),
    EndpointType NVARCHAR(20) NOT NULL DEFAULT 'standard' CHECK(EndpointType IN ('standard', 'custom')),
    CreatedBy NVARCHAR(255),
    ChangeType NVARCHAR(10) NOT NULL CHECK(ChangeType IN ('CREATE', 'UPDATE', 'DELETE')),
    Status NVARCHAR(10) NOT NULL DEFAULT 'pending' CHECK(Status IN ('pending', 'processing', 'completed', 'failed')),
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    ProcessedAt DATETIME2
);
