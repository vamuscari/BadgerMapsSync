CREATE TABLE IF NOT EXISTS RouteWaypoints (
    Id INTEGER PRIMARY KEY,
    RouteId INTEGER,
    Name TEXT,
    Address TEXT,
    Latitude REAL,
    Longitude REAL,
    LayoverMinutes INTEGER,
    Position INTEGER,
    LocationId INTEGER,
    CustomerId INTEGER,
    Type INTEGER,
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP
);