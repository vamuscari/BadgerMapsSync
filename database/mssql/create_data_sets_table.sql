IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='data_sets' AND xtype='U')
CREATE TABLE data_sets (
    id INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(255),
    description NTEXT,
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE()
); 