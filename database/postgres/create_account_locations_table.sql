CREATE TABLE IF NOT EXISTS account_locations (
    id SERIAL PRIMARY KEY,
    account_id INTEGER,
    city VARCHAR(255),
    name VARCHAR(255),
    zipcode VARCHAR(20),
    longitude DECIMAL(11, 8),
    state VARCHAR(100),
    latitude DECIMAL(10, 8),
    address_line1 TEXT,
    location TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);