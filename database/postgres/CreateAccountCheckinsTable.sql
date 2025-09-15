CREATE TABLE IF NOT EXISTS AccountCheckins (
    CheckinId SERIAL PRIMARY KEY,
    CrmId VARCHAR(255),
    AccountId INTEGER,
    LogDatetime TIMESTAMP,
    Type VARCHAR(100),
    Comments TEXT,
    ExtraFields TEXT,
    CreatedBy VARCHAR(255),
    CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (AccountId) REFERENCES Accounts(AccountId)
);