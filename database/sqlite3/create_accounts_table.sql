CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY,
    first_name TEXT,
    last_name TEXT,
    phone_number TEXT,
    email TEXT,
    customer_id TEXT,
    notes TEXT,
    original_address TEXT,
    crm_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
); 