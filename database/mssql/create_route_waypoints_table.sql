IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='route_waypoints' AND xtype='U')
CREATE TABLE route_waypoints (
    id INT IDENTITY(1,1) PRIMARY KEY,
    route_id INT,
    name NVARCHAR(255),
    address NTEXT,
    latitude FLOAT,
    longitude FLOAT,
    layover_minutes INT,
    position INT,
    location_id INT,
    customer_id INT,
    type INT,
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE()
); 