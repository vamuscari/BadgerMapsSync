ALTER TABLE "SyncHistory"
ADD COLUMN IF NOT EXISTS "CompletedAtTimezone" VARCHAR(128);

UPDATE "SyncHistory"
SET "CompletedAtTimezone" = 'UTC'
WHERE "CompletedAt" IS NOT NULL
  AND ("CompletedAtTimezone" IS NULL OR "CompletedAtTimezone" = '');
