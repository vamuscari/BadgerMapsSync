CREATE TABLE IF NOT EXISTS AccountLocations (
    LocationId INTEGER PRIMARY KEY AUTOINCREMENT,
    AccountId INTEGER,
    City TEXT,
    Name TEXT,
    Zipcode TEXT,
    Longitude REAL,
    State TEXT,
    Latitude REAL,
    AddressLine1 TEXT,
    Location TEXT,
    IsApproximate BOOLEAN DEFAULT 0,
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (AccountId) REFERENCES Accounts(AccountId),
    UNIQUE (AccountId)
);