SELECT column_name
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND lower(table_name) = lower(?)
ORDER BY ordinal_position;
