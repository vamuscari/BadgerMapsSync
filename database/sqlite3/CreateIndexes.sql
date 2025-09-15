-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_accounts_account_id ON Accounts(AccountId);
CREATE INDEX IF NOT EXISTS idx_accounts_crm_id ON Accounts(CrmId);
CREATE INDEX IF NOT EXISTS idx_account_checkins_account_id ON AccountCheckins(AccountId);
CREATE INDEX IF NOT EXISTS idx_routes_route_date ON Routes(RouteDate);
CREATE INDEX IF NOT EXISTS idx_route_waypoints_route_id ON RouteWaypoints(RouteId);
CREATE INDEX IF NOT EXISTS idx_account_locations_account_id ON AccountLocations(AccountId); 