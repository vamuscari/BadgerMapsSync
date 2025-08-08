IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='DataSets' AND xtype='U')
CREATE TABLE DataSets (
    Name NVARCHAR(255),
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
    UpdatedAt DATETIME2 DEFAULT GETDATE(),
    PRIMARY KEY (Name, ProfileId)
);