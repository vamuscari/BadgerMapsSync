-- Create indexes for better performance
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxAccountsCustomerId')
CREATE INDEX IdxAccountsCustomerId ON Accounts(CustomerId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxAccountsCrmId')
CREATE INDEX IdxAccountsCrmId ON Accounts(CrmId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxAccountCheckinsCustomer')
CREATE INDEX IdxAccountCheckinsCustomer ON AccountCheckins(Customer);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxAccountCheckinsCrmId')
CREATE INDEX IdxAccountCheckinsCrmId ON AccountCheckins(CrmId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxRoutesRouteDate')
CREATE INDEX IdxRoutesRouteDate ON Routes(RouteDate);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxRouteWaypointsRouteId')
CREATE INDEX IdxRouteWaypointsRouteId ON RouteWaypoints(RouteId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxAccountLocationsAccountId')
CREATE INDEX IdxAccountLocationsAccountId ON AccountLocations(AccountId); 