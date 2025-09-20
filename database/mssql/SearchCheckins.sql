SELECT TOP 50 c.CheckinId, c.AccountId, a.FullName, c.LogDatetime
FROM AccountCheckins c
JOIN Accounts a ON a.AccountId = c.AccountId
WHERE a.FullName LIKE ? OR CAST(c.CheckinId AS NVARCHAR(50)) LIKE ?
ORDER BY
  CASE
    WHEN CAST(c.CheckinId AS NVARCHAR(50)) = ? THEN 0
    WHEN LOWER(a.FullName) = LOWER(?) THEN 1
    WHEN LOWER(a.FullName) LIKE ? THEN 2
    WHEN LOWER(a.FullName) LIKE ? THEN 3
    WHEN LOWER(a.FullName) LIKE ? THEN 4
    ELSE 5
  END,
  c.LogDatetime DESC
