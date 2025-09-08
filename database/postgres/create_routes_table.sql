CREATE TABLE IF NOT EXISTS Routes (
    RouteId SERIAL PRIMARY KEY,
    Name VARCHAR(255),
    RouteDate DATE,
    Duration INTEGER,
    StartAddress TEXT,
    DestinationAddress TEXT,
    StartTime TIME,
    CreatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);