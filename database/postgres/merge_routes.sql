INSERT OR REPLACE INTO Routes (
    Id, Name, RouteDate, StartTime, Duration, StartAddress, DestinationAddress, UpdatedAt
) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP); 