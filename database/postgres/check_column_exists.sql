SELECT COUNT(*) 
FROM information_schema.columns 
WHERE table_name = ? AND column_name = ? 