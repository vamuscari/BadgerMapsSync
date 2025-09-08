SELECT Id, Name, RouteDate, StartAddress, DestinationAddress
FROM Routes
WHERE Name LIKE ? OR StartAddress LIKE ? OR DestinationAddress LIKE ?