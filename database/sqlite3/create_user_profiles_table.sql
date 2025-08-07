CREATE TABLE IF NOT EXISTS user_profiles (
    id INTEGER PRIMARY KEY,
    email TEXT,
    first_name TEXT,
    last_name TEXT,
    is_manager BOOLEAN,
    crm_base_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
); 