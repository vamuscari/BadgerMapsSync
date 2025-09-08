SET IDENTITY_INSERT AccountCheckins ON;
MERGE [AccountCheckins] AS target
USING (SELECT ? as Id, ? as CrmId, ? as Customer, ? as LogDateTime,
       ? as Type, ? as Comments, ? as ExtraFields, ? as CreatedBy) AS source
ON target.[Id] = source.Id
WHEN MATCHED THEN
	UPDATE SET 
		[CrmId] = source.CrmId,
		[Customer] = source.Customer,
		[LogDateTime] = source.LogDateTime,
		[Type] = source.Type,
		[Comments] = source.Comments,
		[ExtraFields] = source.ExtraFields,
		[CreatedBy] = source.CreatedBy,
		[UpdatedAt] = GETDATE()
WHEN NOT MATCHED THEN
	INSERT ([Id], [CrmId], [Customer], [LogDateTime], [Type], [Comments], [ExtraFields], [CreatedBy])
	VALUES (source.Id, source.CrmId, source.Customer, source.LogDateTime,
	        source.Type, source.Comments, source.ExtraFields, source.CreatedBy);
SET IDENTITY_INSERT AccountCheckins OFF; 