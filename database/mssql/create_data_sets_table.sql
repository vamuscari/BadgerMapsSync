IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='DataSets' AND xtype='U')
CREATE TABLE DataSets (
    Id INT IDENTITY(1,1) PRIMARY KEY,
    Name NVARCHAR(255) UNIQUE,
    ProfileId INT,
    Filterable BIT,
    Label NVARCHAR(MAX),
    Position INT,
    Type NVARCHAR(255),
    HasData BIT,
    IsUserCanAddNewTextValues BIT,
    RawMin FLOAT,
    Min FLOAT,
    Max FLOAT,
    RawMax FLOAT,
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    UpdatedAt DATETIME2 DEFAULT GETDATE()
);