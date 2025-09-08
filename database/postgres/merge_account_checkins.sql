INSERT INTO AccountCheckins (
    CheckinId, CrmId, AccountId, LogDatetime, Type, Comments, ExtraFields, CreatedBy
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (CheckinId) DO UPDATE SET
    CrmId = EXCLUDED.CrmId,
    AccountId = EXCLUDED.AccountId,
    LogDatetime = EXCLUDED.LogDatetime,
    Type = EXCLUDED.Type,
    Comments = EXCLUDED.Comments,
    ExtraFields = EXCLUDED.ExtraFields,
    CreatedBy = EXCLUDED.CreatedBy,
    UpdatedAt = CURRENT_TIMESTAMP; 