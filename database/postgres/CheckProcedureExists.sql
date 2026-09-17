SELECT count(*)
FROM information_schema.routines
WHERE routine_schema = current_schema()
  AND lower(routine_name) = lower(?);
