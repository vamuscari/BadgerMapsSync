SELECT AccountId, FullName
FROM Accounts
WHERE FullName ILIKE ? OR CAST(AccountId AS TEXT) LIKE ?
ORDER BY
  CASE
    WHEN CAST(AccountId AS TEXT) = ? THEN 0
    WHEN LOWER(FullName) = LOWER(?) THEN 1
    WHEN LOWER(FullName) LIKE ? THEN 2
    WHEN LOWER(FullName) LIKE ? THEN 3
    WHEN LOWER(FullName) LIKE ? THEN 4
    ELSE 5
  END,
  FullName ASC
LIMIT 50
