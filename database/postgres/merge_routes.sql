INSERT INTO Routes (
    RouteId, Name, RouteDate, StartTime, Duration, StartAddress, DestinationAddress
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (RouteId) DO UPDATE SET
    Name = EXCLUDED.Name,
    RouteDate = EXCLUDED.RouteDate,
    StartTime = EXCLUDED.StartTime,
    Duration = EXCLUDED.Duration,
    StartAddress = EXCLUDED.StartAddress,
    DestinationAddress = EXCLUDED.DestinationAddress,
    UpdatedAt = CURRENT_TIMESTAMP; 