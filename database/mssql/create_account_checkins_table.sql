IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='account_checkins' AND xtype='U')
CREATE TABLE account_checkins (
    id INT IDENTITY(1,1) PRIMARY KEY,
    crm_id NVARCHAR(255),
    customer INT,
    log_datetime DATETIME2,
    type NVARCHAR(100),
    comments NTEXT,
    extra_fields NTEXT,
    created_by NVARCHAR(255),
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE()
); 