MERGE [account_checkins] AS target
USING (SELECT ? as Id, ? as CrmId, ? as CustomerId, ? as LogDateTime,
       ? as Type, ? as Comments, ? as ExtraFields, ? as CreatedBy) AS source
ON target.[Id] = source.Id
WHEN MATCHED THEN
	UPDATE SET 
		[CrmId] = source.CrmId,
		[CustomerId] = source.CustomerId,
		[LogDateTime] = source.LogDateTime,
		[Type] = source.Type,
		[Comments] = source.Comments,
		[ExtraFields] = source.ExtraFields,
		[CreatedBy] = source.CreatedBy,
		[UpdatedAt] = GETDATE()
WHEN NOT MATCHED THEN
	INSERT ([Id], [CrmId], [CustomerId], [LogDateTime], [Type], [Comments], [ExtraFields], [CreatedBy])
	VALUES (source.Id, source.CrmId, source.CustomerId, source.LogDateTime,
	        source.Type, source.Comments, source.ExtraFields, source.CreatedBy); 