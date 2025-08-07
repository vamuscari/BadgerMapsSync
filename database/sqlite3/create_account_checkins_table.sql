CREATE TABLE IF NOT EXISTS account_checkins (
    id INTEGER PRIMARY KEY,
    crm_id TEXT,
    customer INTEGER,
    log_datetime TEXT,
    type TEXT,
    comments TEXT,
    extra_fields TEXT,
    created_by TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
); 