IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='DataSetValues' AND xtype='U')
CREATE TABLE DataSetValues (
    DataSetValueId INT IDENTITY(1,1) PRIMARY KEY,
    ProfileId INT,
    DataSetName NVARCHAR(255),
    DataSetPosition INT,
    Text NVARCHAR(MAX),
    Value NVARCHAR(MAX),
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    UpdatedAt DATETIME2 DEFAULT GETDATE(),
    FOREIGN KEY (ProfileId) REFERENCES UserProfiles(ProfileId),
    FOREIGN KEY (DataSetName, ProfileId) REFERENCES DataSets(Name, ProfileId)
);