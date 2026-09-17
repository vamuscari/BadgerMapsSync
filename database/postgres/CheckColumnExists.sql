SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND lower(table_name) = lower(?)
  AND lower(column_name) = lower(?)
