SET IDENTITY_INSERT AccountCheckins ON;
MERGE [AccountCheckins] AS target
USING (SELECT ? as CheckinId, ? as CrmId, ? as AccountId, ? as LogDateTime,
       ? as Type, ? as Comments, ? as ExtraFields, ? as EndpointType, ? as CreatedBy) AS source
ON target.[CheckinId] = source.CheckinId
WHEN MATCHED THEN
	UPDATE SET 
		[CrmId] = source.CrmId,
		[AccountId] = source.AccountId,
		[LogDateTime] = source.LogDateTime,
		[Type] = source.Type,
		[Comments] = source.Comments,
		[ExtraFields] = source.ExtraFields,
		[EndpointType] = source.EndpointType,
		[CreatedBy] = source.CreatedBy,
		[UpdatedAt] = GETDATE()
WHEN NOT MATCHED THEN
	INSERT ([CheckinId], [CrmId], [AccountId], [LogDateTime], [Type], [Comments], [ExtraFields], [EndpointType], [CreatedBy])
	VALUES (source.CheckinId, source.CrmId, source.AccountId, source.LogDateTime,
	        source.Type, source.Comments, source.ExtraFields, source.EndpointType, source.CreatedBy);
SET IDENTITY_INSERT AccountCheckins OFF;
