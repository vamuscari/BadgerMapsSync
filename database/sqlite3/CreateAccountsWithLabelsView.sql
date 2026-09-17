-- Go replaces the marked projection when profile labels are available. Executing
-- this file directly remains a valid unlabeled fallback for maintenance scripts.
DROP VIEW IF EXISTS AccountsWithLabels;
CREATE VIEW AccountsWithLabels AS
SELECT /* ACCOUNTS_WITH_LABELS_COLUMNS */ *
FROM Accounts AS a;
