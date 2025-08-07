CREATE TABLE IF NOT EXISTS account_locations (
    id INTEGER PRIMARY KEY,
    account_id INTEGER,
    city TEXT,
    name TEXT,
    zipcode TEXT,
    longitude REAL,
    state TEXT,
    latitude REAL,
    address_line1 TEXT,
    location TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
); 