SELECT
    pc.ChangeId,
    pc.CheckinId,
    pc.AccountId,
    pc.CrmId,
    pc.LogDatetime,
    pc.Type,
    pc.Comments,
    pc.ExtraFields,
    pc.EndpointType,
    pc.CreatedBy,
    pc.ChangeType,
    pc.Status,
    pc.CreatedAt,
    pc.ProcessedAt
FROM
    AccountCheckinsPendingChanges pc
WHERE
    pc.Status = 'pending'
ORDER BY
    pc.CreatedAt;
