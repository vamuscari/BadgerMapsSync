CREATE TABLE IF NOT EXISTS AccountLocations (
    Id INTEGER PRIMARY KEY,
    AccountId INTEGER,
    City TEXT,
    Name TEXT,
    Zipcode TEXT,
    Longitude REAL,
    State TEXT,
    Latitude REAL,
    AddressLine1 TEXT,
    Location TEXT,
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP
);