IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='accounts' AND xtype='U')
CREATE TABLE accounts (
    id INT IDENTITY(1,1) PRIMARY KEY,
    first_name NVARCHAR(255),
    last_name NVARCHAR(255),
    phone_number NVARCHAR(50),
    email NVARCHAR(255),
    customer_id NVARCHAR(255),
    notes NTEXT,
    original_address NTEXT,
    crm_id NVARCHAR(255),
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE()
); 