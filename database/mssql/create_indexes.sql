-- Create indexes for better performance
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_accounts_customer_id')
CREATE INDEX idx_accounts_customer_id ON accounts(customer_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_accounts_crm_id')
CREATE INDEX idx_accounts_crm_id ON accounts(crm_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_account_checkins_customer')
CREATE INDEX idx_account_checkins_customer ON account_checkins(customer);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_account_checkins_crm_id')
CREATE INDEX idx_account_checkins_crm_id ON account_checkins(crm_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_routes_route_date')
CREATE INDEX idx_routes_route_date ON routes(route_date);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_route_waypoints_route_id')
CREATE INDEX idx_route_waypoints_route_id ON route_waypoints(route_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_account_locations_account_id')
CREATE INDEX idx_account_locations_account_id ON account_locations(account_id); 