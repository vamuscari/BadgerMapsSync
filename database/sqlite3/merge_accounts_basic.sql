INSERT OR REPLACE INTO accounts (
    id, first_name, last_name, updated_at
) VALUES (?, ?, ?, CURRENT_TIMESTAMP); 