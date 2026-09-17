SELECT count(*)
FROM information_schema.triggers
WHERE trigger_schema = current_schema()
  AND lower(trigger_name) = lower(?);
