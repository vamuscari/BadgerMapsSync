SELECT
    ChangeId,
    CheckinId,
    AccountId,
    CrmId,
    LogDatetime,
    Type,
    Comments,
    ExtraFields,
    EndpointType,
    CreatedBy,
    ChangeType,
    Status,
    CreatedAt,
    ProcessedAt
FROM
    AccountCheckinsPendingChanges
WHERE
    Status = 'pending'
ORDER BY
    CreatedAt;
