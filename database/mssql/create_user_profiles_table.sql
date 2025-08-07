IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='user_profiles' AND xtype='U')
CREATE TABLE user_profiles (
    id INT IDENTITY(1,1) PRIMARY KEY,
    email NVARCHAR(255),
    first_name NVARCHAR(255),
    last_name NVARCHAR(255),
    is_manager BIT,
    crm_base_url NTEXT,
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE()
); 