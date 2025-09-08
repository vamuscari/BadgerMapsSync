IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='AccountCheckins' AND xtype='U')
CREATE TABLE AccountCheckins (
    CheckinId INT IDENTITY(1,1) PRIMARY KEY,
    CrmId NVARCHAR(255),
    AccountId INT,
    LogDatetime DATETIME2,
    Type NVARCHAR(100),
    Comments NTEXT,
    ExtraFields NTEXT,
    CreatedBy NVARCHAR(255),
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    UpdatedAt DATETIME2 DEFAULT GETDATE(),
    FOREIGN KEY (AccountId) REFERENCES Accounts(AccountId)
); 