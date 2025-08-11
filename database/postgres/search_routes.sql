SELECT Id, Name, RouteDate, StartAddress, DestinationAddress
FROM Routes
WHERE Name LIKE $1 OR StartAddress LIKE $2 OR DestinationAddress LIKE $3