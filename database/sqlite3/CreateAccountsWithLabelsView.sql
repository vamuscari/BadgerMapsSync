-- SQLite does not support dynamic SQL in triggers or stored procedures to build views
-- based on table data. This view is created as a simple copy of the Accounts table.
-- For dynamic labeled data, application-level logic is required.
CREATE VIEW IF NOT EXISTS AccountsWithLabels AS
SELECT * FROM Accounts;
