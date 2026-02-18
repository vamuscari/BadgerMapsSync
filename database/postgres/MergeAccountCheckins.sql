INSERT INTO AccountCheckins (
    CheckinId, CrmId, AccountId, LogDatetime, Type, Comments, ExtraFields, EndpointType, CreatedBy
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (CheckinId) DO UPDATE SET
    CrmId = EXCLUDED.CrmId,
    AccountId = EXCLUDED.AccountId,
    LogDatetime = EXCLUDED.LogDatetime,
    Type = EXCLUDED.Type,
    Comments = EXCLUDED.Comments,
    ExtraFields = EXCLUDED.ExtraFields,
    EndpointType = EXCLUDED.EndpointType,
    CreatedBy = EXCLUDED.CreatedBy,
    UpdatedAt = CURRENT_TIMESTAMP;
