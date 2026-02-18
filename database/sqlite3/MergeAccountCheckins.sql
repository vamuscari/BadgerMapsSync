INSERT OR REPLACE INTO AccountCheckins (
    CheckinId, CrmId, AccountId, LogDatetime, Type, Comments, ExtraFields, EndpointType, CreatedBy, UpdatedAt
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);
