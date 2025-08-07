CREATE TABLE IF NOT EXISTS Accounts (
    Id INTEGER PRIMARY KEY,
    FirstName TEXT,
    LastName TEXT,
    PhoneNumber TEXT,
    Email TEXT,
    CustomerId TEXT,
    Notes TEXT,
    OriginalAddress TEXT,
    CrmId TEXT,
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP
);