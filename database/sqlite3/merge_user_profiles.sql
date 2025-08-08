INSERT OR REPLACE INTO UserProfiles (
    ProfileId, FirstName, LastName, Email, IsManager, Manager, CompanyId,
    CompanyName, CompanyShortName, Completed, TrialDaysLeft, HasData, DefaultApptLength,
    UpdatedAt
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);