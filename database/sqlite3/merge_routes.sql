INSERT OR REPLACE INTO Routes (
    RouteId, Name, RouteDate, StartTime, Duration, StartAddress, DestinationAddress, UpdatedAt
) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP); 