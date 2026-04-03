ALTER TABLE SyncHistory
ADD CompletedAtTimezone NVARCHAR(128);

UPDATE SyncHistory
SET CompletedAtTimezone = 'UTC'
WHERE CompletedAt IS NOT NULL
  AND (CompletedAtTimezone IS NULL OR CompletedAtTimezone = '');
