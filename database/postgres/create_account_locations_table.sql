CREATE TABLE IF NOT EXISTS AccountLocations (
    Id SERIAL PRIMARY KEY,
    AccountId INTEGER,
    City VARCHAR(255),
    Name VARCHAR(255),
    Zipcode VARCHAR(20),
    Longitude DECIMAL(11, 8),
    State VARCHAR(100),
    Latitude DECIMAL(10, 8),
    AddressLine1 TEXT,
    Location TEXT,
    CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);