IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='UserProfiles' AND xtype='U')
CREATE TABLE UserProfiles (
    Id INT IDENTITY(1,1) PRIMARY KEY,
    Email NVARCHAR(255),
    FirstName NVARCHAR(255),
    LastName NVARCHAR(255),
    IsManager BIT,
    Manager NTEXT,
    CompanyId INT,
    CompanyName NVARCHAR(255),
    CompanyShortName NVARCHAR(255),
    Completed BIT,
    TrialDaysLeft INT,
    HasData BIT,
    DefaultApptLength INT,
    CrmBaseUrl NTEXT,
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    UpdatedAt DATETIME2 DEFAULT GETDATE()
); 