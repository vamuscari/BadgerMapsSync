IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='RouteWaypoints' AND xtype='U')
CREATE TABLE RouteWaypoints (
    WaypointId INT IDENTITY(1,1) PRIMARY KEY,
    RouteId INT,
    Name NVARCHAR(255),
    Address NTEXT,
    Suite NTEXT,
    City NVARCHAR(255),
    State NVARCHAR(255),
    Zipcode NVARCHAR(255),
    Location NTEXT,
    Latitude FLOAT,
    Longitude FLOAT,
    LayoverMinutes INT,
    Position INT,
    CompleteAddress NTEXT,
    LocationId INT,
    CustomerId INT,
    ApptTime DATETIME2,
    Type INT,
    PlaceId NVARCHAR(255),
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    UpdatedAt DATETIME2 DEFAULT GETDATE()
); 