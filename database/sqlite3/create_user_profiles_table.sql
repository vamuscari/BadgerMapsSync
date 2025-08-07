CREATE TABLE IF NOT EXISTS UserProfiles (
    Id INTEGER PRIMARY KEY,
    Email TEXT,
    FirstName TEXT,
    LastName TEXT,
    IsManager BOOLEAN,
    Manager TEXT,
    CompanyId INTEGER,
    CompanyName TEXT,
    CompanyShortName TEXT,
    Completed BOOLEAN,
    TrialDaysLeft INTEGER,
    HasData BOOLEAN,
    DefaultApptLength INTEGER,
    CrmBaseUrl TEXT,
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP
);