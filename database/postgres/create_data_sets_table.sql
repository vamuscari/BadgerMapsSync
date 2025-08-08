CREATE TABLE IF NOT EXISTS DataSets (
    Name VARCHAR(255),
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
    CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (Name, ProfileId)
);