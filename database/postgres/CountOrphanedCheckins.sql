SELECT COUNT(*)
FROM "AccountCheckins" ac
WHERE ac."AccountId" IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM "Accounts" a
    WHERE a."AccountId" = ac."AccountId"
  );
