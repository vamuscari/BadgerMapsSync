INSERT OR REPLACE INTO AccountCheckins (
    Id, CrmId, AccountId, LogDateTime, Type, Comments, ExtraFields, CreatedBy, UpdatedAt
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP); 