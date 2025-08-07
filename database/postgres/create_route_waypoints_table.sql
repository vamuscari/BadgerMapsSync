CREATE TABLE IF NOT EXISTS route_waypoints (
    id SERIAL PRIMARY KEY,
    route_id INTEGER,
    name VARCHAR(255),
    address TEXT,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    layover_minutes INTEGER,
    position INTEGER,
    location_id INTEGER,
    customer_id INTEGER,
    type INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);