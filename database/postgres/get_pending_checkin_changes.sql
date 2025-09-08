SELECT "ChangeId", "CheckinId", "ChangeType", "Changes", "Status", "CreatedAt", "ProcessedAt" FROM "AccountCheckinsPendingChanges" WHERE "Status" = 'pending' ORDER BY "CreatedAt";
