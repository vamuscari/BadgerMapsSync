MERGE [Accounts] AS target
USING (SELECT ? as Id, ? as FirstName, ? as LastName) AS source
ON target.[Id] = source.Id
WHEN MATCHED THEN
	UPDATE SET 
		[FirstName] = source.FirstName,
		[LastName] = source.LastName,
		[UpdatedAt] = GETDATE()
WHEN NOT MATCHED THEN
	INSERT ([Id], [FirstName], [LastName])
	VALUES (source.Id, source.FirstName, source.LastName); 