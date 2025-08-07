MERGE Routes AS target
USING (SELECT ? as Id, ? as Name, ? as RouteDate, ? as StartTime,
       ? as Duration, ? as StartAddress, ? as DestinationAddress) AS source
ON target.Id = source.Id
WHEN MATCHED THEN
	UPDATE SET 
		Name = source.Name,
		RouteDate = source.RouteDate,
		StartTime = source.StartTime,
		Duration = source.Duration,
		StartAddress = source.StartAddress,
		DestinationAddress = source.DestinationAddress,
		UpdatedAt = GETDATE()
WHEN NOT MATCHED THEN
	INSERT (Id, Name, RouteDate, StartTime, Duration, StartAddress, DestinationAddress)
	VALUES (source.Id, source.Name, source.RouteDate, source.StartTime,
	        source.Duration, source.StartAddress, source.DestinationAddress); 