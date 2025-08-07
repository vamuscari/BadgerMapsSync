IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='data_set_values' AND xtype='U')
CREATE TABLE data_set_values (
    id INT IDENTITY(1,1) PRIMARY KEY,
    data_set_id INT,
    value NTEXT,
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE()
); 