IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='Routes' AND xtype='U')
CREATE TABLE Routes (
    RouteId INT IDENTITY(1,1) PRIMARY KEY,
    Name NVARCHAR(255),
    RouteDate DATE,
    Duration INT,
    StartAddress NTEXT,
    DestinationAddress NTEXT,
    StartTime TIME,
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    UpdatedAt DATETIME2 DEFAULT GETDATE()
); 