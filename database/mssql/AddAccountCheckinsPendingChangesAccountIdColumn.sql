ALTER TABLE AccountCheckinsPendingChanges
ADD AccountId INT NOT NULL
    CONSTRAINT DF_AccountCheckinsPendingChanges_AccountId DEFAULT 0;

EXEC('UPDATE pc
SET pc.AccountId = c.AccountId
FROM AccountCheckinsPendingChanges pc
INNER JOIN AccountCheckins c ON c.CheckinId = pc.CheckinId
WHERE pc.AccountId = 0;');
