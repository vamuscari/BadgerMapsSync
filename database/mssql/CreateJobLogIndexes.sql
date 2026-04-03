IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxJobLogCorrelation')
CREATE UNIQUE INDEX IdxJobLogCorrelation ON JobLog(CorrelationId);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxJobLogStartedAt')
CREATE INDEX IdxJobLogStartedAt ON JobLog(StartedAt DESC);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IdxJobLogRootCorrelationId')
CREATE INDEX IdxJobLogRootCorrelationId ON JobLog(RootCorrelationId);
