IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='route_waypoints' AND xtype='U')
CREATE TABLE route_waypoints (
    id INT IDENTITY(1,1) PRIMARY KEY,
    route_id INT,
    name NVARCHAR(255),
    address NTEXT,
    suite NTEXT,
    city NVARCHAR(255),
    state NVARCHAR(255),
    zipcode NVARCHAR(255),
    location NTEXT,
    latitude FLOAT,
    longitude FLOAT,
    layover_minutes INT,
    position INT,
    complete_address NTEXT,
    location_id INT,
    customer_id INT,
    appt_time DATETIME2,
    type INT,
    place_id NVARCHAR(255),
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE()
); 