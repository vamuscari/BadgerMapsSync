INSERT OR REPLACE INTO AccountCheckins (
    Id, CrmId, CustomerId, LogDateTime, Type, Comments, ExtraFields, CreatedBy, UpdatedAt
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP); 