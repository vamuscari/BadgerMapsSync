-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_accounts_customer_id ON accounts(customer_id);
CREATE INDEX IF NOT EXISTS idx_accounts_crm_id ON accounts(crm_id);
CREATE INDEX IF NOT EXISTS idx_account_checkins_customer ON account_checkins(customer);
CREATE INDEX IF NOT EXISTS idx_account_checkins_crm_id ON account_checkins(crm_id);
CREATE INDEX IF NOT EXISTS idx_routes_route_date ON routes(route_date);
CREATE INDEX IF NOT EXISTS idx_route_waypoints_route_id ON route_waypoints(route_id);
CREATE INDEX IF NOT EXISTS idx_account_locations_account_id ON account_locations(account_id);