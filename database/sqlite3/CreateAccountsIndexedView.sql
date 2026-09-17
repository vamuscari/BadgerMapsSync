-- Go replaces the marked projection with columns selected for the active profile.
-- Direct execution falls back to fixed account fields because no profile metadata
-- can be interpolated by SQLite itself.
DROP VIEW IF EXISTS AccountsIndexed;
CREATE VIEW AccountsIndexed AS
SELECT /* ACCOUNTS_INDEXED_COLUMNS */ a.AccountId, a.FirstName, a.LastName, a.FullName, a.PhoneNumber, a.Email, a.CustomerId, a.Notes, a.OriginalAddress, a.CrmId, a.AccountOwner, a.DaysSinceLastCheckin, a.LastCheckinDate, a.LastModifiedDate, a.FollowUpDate, a.CreatedAt, a.UpdatedAt
FROM Accounts AS a;
