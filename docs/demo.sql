-- Demo dataset for BadgerMapsSync (SQLite)
-- Load into a freshly-initialized SQLite DB:
--   sqlite3 test.db < docs/demo.sql

-- Minimal Accounts
INSERT INTO Accounts (AccountId, FirstName, LastName, FullName, Email, PhoneNumber, CustomerId, Notes, AccountOwner)
VALUES
  (1, 'Ada', 'Lovelace',  'Ada Lovelace',  'ada@example.com',  '+1-555-0100', 'CUST-001', 'VIP customer', 'alice@company.com'),
  (2, 'Alan', 'Turing',   'Alan Turing',   'alan@example.com', '+1-555-0101', 'CUST-002', 'Priority support', 'bob@company.com'),
  (3, 'Grace','Hopper',   'Grace Hopper',  'grace@example.com','+1-555-0102', 'CUST-003', 'Prefers email', 'carol@company.com');

-- Minimal Routes
INSERT INTO Routes (RouteId, Name, RouteDate, Duration, StartAddress, DestinationAddress, StartTime)
VALUES
  (101, 'Morning Route',  '2025-01-05', 90, '123 Main St', '456 Market St', '08:30'),
  (102, 'Afternoon Run',  '2025-01-05', 75, '456 Market St', '789 Broadway', '13:15');

-- Minimal Account Checkins
INSERT INTO AccountCheckins (CheckinId, CrmId, AccountId, LogDatetime, Type, Comments, CreatedBy)
VALUES
  (1001, 'CHK-001', 1, '2025-01-05T09:45:00Z', 'Visit', 'Discussed renewal terms', 'alice@company.com'),
  (1002, 'CHK-002', 2, '2025-01-05T10:30:00Z', 'Call',  'Left voicemail; will follow up', 'bob@company.com');

-- Optional: some pending changes tables to light up Push card / Explorer
INSERT INTO AccountsPendingChanges (ChangeId, AccountId, ChangeType, Changes, Status)
VALUES
  (2001, 1, 'UPDATE', '{"Notes":"Upsell interest"}', 'pending');

INSERT INTO AccountCheckinsPendingChanges (ChangeId, CheckinId, ChangeType, Changes, Status)
VALUES
  (3001, 1001, 'CREATE', '{"Comments":"Added post-visit summary"}', 'pending');

-- User Profile (owner) so DataSets can reference a ProfileId
INSERT INTO UserProfiles (ProfileId, Email, FirstName, LastName, IsManager, HasData, CompanyId, CompanyName, CompanyShortName)
VALUES (42, 'owner@example.com', 'Owner', 'User', 1, 1, 77, 'Example Co', 'EXCO');

-- Example DataSets (labels for account fields)
INSERT INTO DataSets (Name, ProfileId, Filterable, Label, Position, Type, HasData, IsUserCanAddNewTextValues, AccountField)
VALUES
  ('AccountOwner', 42, 1, 'Owner', 1, 'text', 1, 0, 'AccountOwner'),
  ('Email',        42, 1, 'Email', 2, 'text', 1, 0, 'Email'),
  ('LastCheckin',  42, 1, 'Last Check-in', 3, 'date', 1, 0, 'LastCheckinDate');

-- DataSet Values (simple picklists)
INSERT INTO DataSetValues (DataSetName, ProfileId, Text, Value, DataSetPosition)
VALUES
  ('AccountOwner', 42, 'alice@company.com', 'alice@company.com', 1),
  ('AccountOwner', 42, 'bob@company.com',   'bob@company.com',   2),
  ('AccountOwner', 42, 'carol@company.com', 'carol@company.com', 3);

-- Sync History (sample runs)
INSERT INTO SyncHistory (CorrelationId, RunType, Direction, Source, Initiator, Status, ItemsProcessed, ErrorCount, StartedAt, CompletedAt, DurationSeconds, Summary, Details)
VALUES
  ('corr-0001', 'manual', 'pull', 'accounts', 'user', 'completed', 3, 0, '2025-01-05T08:00:00Z', '2025-01-05T08:00:12Z', 12, 'Pulled accounts', '3 accounts pulled successfully'),
  ('corr-0002', 'manual', 'push', 'changes',  'user', 'completed', 2, 0, '2025-01-05T08:30:00Z', '2025-01-05T08:30:08Z', 8,  'Pushed pending changes', '2 changes pushed successfully');

-- Command Log (example CLI invocations)
INSERT INTO CommandLog (Command, Args, Success, ErrorMessage)
VALUES
  ('pull', 'accounts', 1, NULL),
  ('push', 'all',      1, NULL);

-- Webhook Log (mocked events)
INSERT INTO WebhookLog (ReceivedAt, Method, Uri, Headers, Body)
VALUES
  ('2025-01-05T09:00:00Z', 'POST', '/webhook/account', '{"Content-Type":"application/json"}', '{"event":"account.updated","id":1}');
