IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='AccountsPendingChanges' AND xtype='U')
CREATE TABLE AccountsPendingChanges (
    ChangeId INT IDENTITY(1,1) PRIMARY KEY,
    AccountId INT NOT NULL,
    ChangeType NVARCHAR(10) NOT NULL CHECK(ChangeType IN ('CREATE', 'UPDATE', 'DELETE')),
    Changes NVARCHAR(MAX),
    Status NVARCHAR(10) NOT NULL DEFAULT 'pending' CHECK(Status IN ('pending', 'processing', 'completed', 'failed')),
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    ProcessedAt DATETIME2
);
