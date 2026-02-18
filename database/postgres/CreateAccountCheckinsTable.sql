CREATE TABLE IF NOT EXISTS AccountCheckins (
    CheckinId SERIAL PRIMARY KEY,
    CrmId VARCHAR(255),
    AccountId INTEGER,
    LogDatetime TIMESTAMP,
    Type VARCHAR(100),
    Comments TEXT,
    ExtraFields TEXT,
    EndpointType VARCHAR(20) NOT NULL DEFAULT 'standard' CHECK(EndpointType IN ('standard', 'custom')),
    CreatedBy VARCHAR(255),
    CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (AccountId) REFERENCES Accounts(AccountId)
);
