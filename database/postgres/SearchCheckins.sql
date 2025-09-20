SELECT c.CheckinId, c.AccountId, a.FullName, c.LogDatetime
FROM AccountCheckins c
JOIN Accounts a ON a.AccountId = c.AccountId
WHERE a.FullName ILIKE ? OR CAST(c.CheckinId AS TEXT) LIKE ?
ORDER BY
  CASE
    WHEN CAST(c.CheckinId AS TEXT) = ? THEN 0
    WHEN LOWER(a.FullName) = LOWER(?) THEN 1
    WHEN LOWER(a.FullName) LIKE ? THEN 2
    WHEN LOWER(a.FullName) LIKE ? THEN 3
    WHEN LOWER(a.FullName) LIKE ? THEN 4
    ELSE 5
  END,
  c.LogDatetime DESC
LIMIT 50
