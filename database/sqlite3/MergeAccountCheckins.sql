INSERT OR REPLACE INTO AccountCheckins (
    CheckinId, CrmId, AccountId, LogDateTime, Type, Comments, ExtraFields, CreatedBy, UpdatedAt
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP); 