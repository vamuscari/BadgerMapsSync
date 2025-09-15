SELECT COUNT(*) 
FROM sqlite_master 
WHERE type='index' AND name=? 