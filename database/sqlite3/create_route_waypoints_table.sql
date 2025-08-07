CREATE TABLE IF NOT EXISTS route_waypoints (
    id INTEGER PRIMARY KEY,
    route_id INTEGER,
    name TEXT,
    address TEXT,
    latitude REAL,
    longitude REAL,
    layover_minutes INTEGER,
    position INTEGER,
    location_id INTEGER,
    customer_id INTEGER,
    type INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
); 