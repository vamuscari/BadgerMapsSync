-- Create indexes for better performance
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxAccountsCustomerId')
CREATE INDEX IdxAccountsCustomerId ON Accounts(CustomerId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxAccountsCrmId')
CREATE INDEX IdxAccountsCrmId ON Accounts(CrmId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxAccountCheckinsAccountId')
CREATE INDEX IdxAccountCheckinsAccountId ON AccountCheckins(AccountId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxAccountCheckinsCrmId')
CREATE INDEX IdxAccountCheckinsCrmId ON AccountCheckins(CrmId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxRoutesRouteDate')
CREATE INDEX IdxRoutesRouteDate ON Routes(RouteDate);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxRouteWaypointsRouteId')
CREATE INDEX IdxRouteWaypointsRouteId ON RouteWaypoints(RouteId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxAccountLocationsAccountId')
CREATE INDEX IdxAccountLocationsAccountId ON AccountLocations(AccountId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxSyncHistoryStartedAt')
CREATE INDEX IdxSyncHistoryStartedAt ON SyncHistory(StartedAt DESC);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxJobLogCorrelation')
CREATE UNIQUE INDEX IdxJobLogCorrelation ON JobLog(CorrelationId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxJobLogStartedAt')
CREATE INDEX IdxJobLogStartedAt ON JobLog(StartedAt DESC);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxJobLogRootCorrelationId')
CREATE INDEX IdxJobLogRootCorrelationId ON JobLog(RootCorrelationId);
