-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS IdxAccountsCustomerId ON Accounts(CustomerId);
CREATE INDEX IF NOT EXISTS IdxAccountsCrmId ON Accounts(CrmId);
CREATE INDEX IF NOT EXISTS IdxAccountCheckinsAccountId ON AccountCheckins(AccountId);
CREATE INDEX IF NOT EXISTS IdxAccountCheckinsCrmId ON AccountCheckins(CrmId);
CREATE INDEX IF NOT EXISTS IdxRoutesRouteDate ON Routes(RouteDate);
CREATE INDEX IF NOT EXISTS IdxRouteWaypointsRouteId ON RouteWaypoints(RouteId);
CREATE INDEX IF NOT EXISTS IdxAccountLocationsAccountId ON AccountLocations(AccountId);
CREATE INDEX IF NOT EXISTS IdxSyncHistoryStartedAt ON SyncHistory(StartedAt DESC);
