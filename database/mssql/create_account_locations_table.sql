IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='account_locations' AND xtype='U')
CREATE TABLE account_locations (
    LocationId INT IDENTITY(1,1) PRIMARY KEY,
    Id INT,
    account_id INT,
    city NVARCHAR(255),
    name NVARCHAR(255),
    zipcode NVARCHAR(20),
    longitude FLOAT,
    state NVARCHAR(100),
    latitude FLOAT,
    address_line1 NTEXT,
    location NTEXT,
    is_approximate BIT DEFAULT 0,
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE(),
    UNIQUE (Id, account_id)
); 