SELECT COUNT(*)
FROM "RouteWaypoints" rw
WHERE rw."RouteId" IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM "Routes" r
    WHERE r."RouteId" = rw."RouteId"
  );
