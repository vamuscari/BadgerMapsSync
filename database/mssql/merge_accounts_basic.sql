MERGE Accounts AS target
USING (SELECT ? AS Id, ? AS FullName) AS source
ON (target.Id = source.Id)
WHEN MATCHED THEN
    UPDATE SET FullName = source.FullName
WHEN NOT MATCHED THEN
    INSERT (Id, FullName) VALUES (source.Id, source.FullName);