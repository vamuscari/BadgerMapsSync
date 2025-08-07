CREATE TABLE IF NOT EXISTS account_checkins (
    id SERIAL PRIMARY KEY,
    crm_id VARCHAR(255),
    customer INTEGER,
    log_datetime TIMESTAMP,
    type VARCHAR(100),
    comments TEXT,
    extra_fields TEXT,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);