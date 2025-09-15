IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='CommandLog' and xtype='U')
CREATE TABLE CommandLog (
    LogId INT IDENTITY(1,1) PRIMARY KEY,
    Command NVARCHAR(255) NOT NULL,
    Args NVARCHAR(MAX),
    Timestamp DATETIME DEFAULT GETDATE(),
    Success BIT NOT NULL,
    ErrorMessage NVARCHAR(MAX)
);
