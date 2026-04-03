CREATE UNIQUE INDEX IF NOT EXISTS idx_joblog_correlation ON JobLog(CorrelationId);
CREATE INDEX IF NOT EXISTS idx_joblog_started_at ON JobLog(StartedAt DESC);
CREATE INDEX IF NOT EXISTS idx_joblog_root_correlation ON JobLog(RootCorrelationId);
