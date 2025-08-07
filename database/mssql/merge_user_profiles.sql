MERGE UserProfiles AS target
USING (SELECT ? as Id, ? as FirstName, ? as LastName, ? as Email,
       ? as IsManager, ? as Manager, ? as CompanyId, ? as CompanyName,
       ? as CompanyShortName, ? as Completed, ? as TrialDaysLeft,
       ? as HasData, ? as DefaultApptLength) AS source
ON target.Id = source.Id
WHEN MATCHED THEN
	UPDATE SET 
		FirstName = source.FirstName,
		LastName = source.LastName,
		Email = source.Email,
		IsManager = source.IsManager,
		Manager = source.Manager,
		CompanyId = source.CompanyId,
		CompanyName = source.CompanyName,
		CompanyShortName = source.CompanyShortName,
		Completed = source.Completed,
		TrialDaysLeft = source.TrialDaysLeft,
		HasData = source.HasData,
		DefaultApptLength = source.DefaultApptLength,
		UpdatedAt = GETDATE()
WHEN NOT MATCHED THEN
	INSERT (Id, FirstName, LastName, Email, IsManager, Manager, CompanyId,
	        CompanyName, CompanyShortName, Completed, TrialDaysLeft, HasData, DefaultApptLength)
	VALUES (source.Id, source.FirstName, source.LastName, source.Email, source.IsManager,
	        source.Manager, source.CompanyId, source.CompanyName, source.CompanyShortName,
	        source.Completed, source.TrialDaysLeft, source.HasData, source.DefaultApptLength); 