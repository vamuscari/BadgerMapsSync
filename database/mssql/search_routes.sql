SELECT Id, Name, RouteDate, StartAddress, DestinationAddress
FROM Routes
WHERE Name LIKE @p1 OR StartAddress LIKE @p2 OR DestinationAddress LIKE @p3