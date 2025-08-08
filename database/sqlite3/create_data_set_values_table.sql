CREATE TABLE IF NOT EXISTS DataSetValues
(
    DataSetName     TEXT,
    ProfileId       INTEGER,
    Text            TEXT,
    Value           TEXT,
    DataSetPosition INTEGER,
    CreatedAt       DATETIME DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt       DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (DataSetName) REFERENCES DataSet (Name),
    FOREIGN KEY (ProfileId) REFERENCES UserProfiles (ProfileId)
);