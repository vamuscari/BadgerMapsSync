IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='WebhookLog' AND xtype='U')
CREATE TABLE WebhookLog (
    Id INT PRIMARY KEY IDENTITY,
    ReceivedAt DATETIME NOT NULL,
    Method NVARCHAR(10) NOT NULL,
    Uri NVARCHAR(255) NOT NULL,
    Headers NVARCHAR(MAX),
    Body NVARCHAR(MAX)
);
