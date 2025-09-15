CREATE TABLE IF NOT EXISTS AccountLocations (
    LocationId SERIAL PRIMARY KEY,
    AccountId INTEGER,
    City VARCHAR(255),
    Name VARCHAR(255),
    Zipcode VARCHAR(20),
    Longitude DECIMAL(11, 8),
    State VARCHAR(100),
    Latitude DECIMAL(10, 8),
    AddressLine1 TEXT,
    Location TEXT,
    IsApproximate BOOLEAN DEFAULT FALSE,
    CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (AccountId) REFERENCES Accounts(AccountId),
    UNIQUE (AccountId)
);