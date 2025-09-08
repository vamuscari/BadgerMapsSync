UPDATE "%s" SET "Status" = $1, "ProcessedAt" = CURRENT_TIMESTAMP WHERE "ChangeId" = $2;
