IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='routes' AND xtype='U')
CREATE TABLE routes (
    id INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(255),
    route_date DATE,
    duration INT,
    start_address NTEXT,
    destination_address NTEXT,
    start_time TIME,
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE()
); 