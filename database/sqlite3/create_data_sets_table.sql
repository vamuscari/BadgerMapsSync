CREATE TABLE IF NOT EXISTS DataSets (
    Name TEXT,
    ProfileId INTEGER,
    Filterable BOOLEAN,
    Label TEXT,
    Position INTEGER,
    Type TEXT,
    HasData BOOLEAN,
    IsUserCanAddNewTextValues BOOLEAN,
    RawMin REAL,
    Min REAL,
    Max REAL,
    RawMax REAL,
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (Name, ProfileId),
    FOREIGN KEY (ProfileId) REFERENCES UserProfiles (ProfileId)
);