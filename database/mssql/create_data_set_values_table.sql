IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='DataSetValues' AND xtype='U')
CREATE TABLE DataSetValues (
    Id INT IDENTITY(1,1) PRIMARY KEY,
    ProfileId INT,
    DataSetId INT,
    DataSetName NVARCHAR(255),
    DataSetPosition INT,
    Text NVARCHAR(MAX),
    Value NVARCHAR(MAX),
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    UpdatedAt DATETIME2 DEFAULT GETDATE()
);