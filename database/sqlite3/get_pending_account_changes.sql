SELECT ChangeId, AccountId, ChangeType, Changes, Status, CreatedAt, ProcessedAt FROM AccountsPendingChanges WHERE Status = 'pending' ORDER BY CreatedAt;
