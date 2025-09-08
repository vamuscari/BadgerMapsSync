CREATE TABLE IF NOT EXISTS DataSetValues
(
    DataSetValueId  INTEGER PRIMARY KEY AUTOINCREMENT,
    DataSetName     TEXT,
    ProfileId       INTEGER,
    Text            TEXT,
    Value           TEXT,
    DataSetPosition INTEGER,
    CreatedAt       DATETIME DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt       DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (DataSetName, ProfileId) REFERENCES DataSets (Name, ProfileId),
    FOREIGN KEY (ProfileId) REFERENCES UserProfiles (ProfileId)
);