CREATE TABLE IF NOT EXISTS AccountCheckins (
    Id INTEGER PRIMARY KEY,
    CrmId TEXT,
    Customer INTEGER,
    LogDatetime TEXT,
    Type TEXT,
    Comments TEXT,
    ExtraFields TEXT,
    CreatedBy TEXT,
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP
);