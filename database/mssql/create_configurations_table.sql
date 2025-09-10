IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='Configurations' AND xtype='U')
CREATE TABLE Configurations (
    SettingKey NVARCHAR(255) PRIMARY KEY,
    SettingValue NVARCHAR(MAX),
    LastModified DATETIME2 DEFAULT GETDATE()
);