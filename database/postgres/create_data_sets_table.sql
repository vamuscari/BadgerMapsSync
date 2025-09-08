CREATE TABLE IF NOT EXISTS DataSets (
    Name TEXT,
    ProfileId INTEGER,
    Filterable BOOLEAN,
    Label TEXT,
    Position INTEGER,
    Type TEXT,
    HasData BOOLEAN,
    IsUserCanAddNewTextValues BOOLEAN,
    RawMin NUMERIC,
    Min NUMERIC,
    Max NUMERIC,
    RawMax NUMERIC,
    AccountField TEXT,
    CreatedAt TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (Name, ProfileId),
    FOREIGN KEY (ProfileId) REFERENCES UserProfiles (ProfileId)
);