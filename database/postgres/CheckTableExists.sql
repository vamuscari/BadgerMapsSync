SELECT COUNT(*) 
FROM information_schema.tables 
WHERE table_schema = current_schema()
  AND lower(table_name) = lower(?)
