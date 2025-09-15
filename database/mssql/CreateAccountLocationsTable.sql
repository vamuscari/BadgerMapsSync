IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='AccountLocations' AND xtype='U')
CREATE TABLE AccountLocations (
    LocationId INT IDENTITY(1,1) PRIMARY KEY,
    AccountId INT,
    City NVARCHAR(255),
    Name NVARCHAR(255),
    Zipcode NVARCHAR(20),
    Longitude FLOAT,
    State NVARCHAR(100),
    Latitude FLOAT,
    AddressLine1 NVARCHAR(MAX),
    Location NVARCHAR(MAX),
    IsApproximate BIT DEFAULT 0,
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    UpdatedAt DATETIME2 DEFAULT GETDATE(),
    FOREIGN KEY (AccountId) REFERENCES Accounts(AccountId),
    UNIQUE (AccountId)
); 