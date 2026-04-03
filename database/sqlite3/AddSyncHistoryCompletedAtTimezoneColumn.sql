ALTER TABLE SyncHistory
ADD COLUMN CompletedAtTimezone TEXT;

UPDATE SyncHistory
SET CompletedAtTimezone = 'UTC'
WHERE CompletedAt IS NOT NULL
  AND (CompletedAtTimezone IS NULL OR CompletedAtTimezone = '');
