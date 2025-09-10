SELECT
    pc.ChangeId,
    pc.CheckinId,
    ac.AccountId,
    pc.ChangeType,
    pc.Changes,
    pc.Status,
    pc.CreatedAt,
    pc.ProcessedAt
FROM
    AccountCheckinsPendingChanges pc
INNER JOIN
    AccountCheckins ac ON pc.CheckinId = ac.CheckinId
