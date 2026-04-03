CREATE UNIQUE INDEX IF NOT EXISTS IdxJobLogCorrelation ON JobLog(CorrelationId);
CREATE INDEX IF NOT EXISTS IdxJobLogStartedAt ON JobLog(StartedAt DESC);
CREATE INDEX IF NOT EXISTS IdxJobLogRootCorrelationId ON JobLog(RootCorrelationId);
