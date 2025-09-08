MERGE Accounts AS target
USING (SELECT ? AS AccountId, ? AS FullName) AS source
ON (target.AccountId = source.AccountId)
WHEN MATCHED THEN
    UPDATE SET FullName = source.FullName
WHEN NOT MATCHED THEN
    INSERT (AccountId, FullName) VALUES (source.AccountId, source.FullName);