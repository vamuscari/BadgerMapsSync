CREATE TABLE IF NOT EXISTS Routes (
    Id INTEGER PRIMARY KEY,
    Name TEXT,
    RouteDate TEXT,
    Duration INTEGER,
    StartAddress TEXT,
    DestinationAddress TEXT,
    StartTime TEXT,
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP
);