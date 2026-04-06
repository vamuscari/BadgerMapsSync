SELECT COUNT(*)
FROM "AccountLocations" al
WHERE al."AccountId" IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM "Accounts" a
    WHERE a."AccountId" = al."AccountId"
  );
