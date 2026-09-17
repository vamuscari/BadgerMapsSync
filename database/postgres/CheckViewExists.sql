SELECT count(*)
FROM information_schema.views
WHERE table_schema = current_schema()
  AND lower(table_name) = lower(?);
