CREATE TABLE IF NOT EXISTS RouteWaypoints (
    Id SERIAL PRIMARY KEY,
    RouteId INTEGER,
    Name VARCHAR(255),
    Address TEXT,
    Latitude DECIMAL(10, 8),
    Longitude DECIMAL(11, 8),
    LayoverMinutes INTEGER,
    Position INTEGER,
    LocationId INTEGER,
    CustomerId INTEGER,
    Type INTEGER,
    CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);