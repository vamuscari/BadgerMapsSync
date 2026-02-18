ALTER TABLE AccountCheckinsPendingChanges
ADD EndpointType NVARCHAR(20) NOT NULL
    CONSTRAINT DF_AccountCheckinsPendingChanges_EndpointType DEFAULT 'standard';

EXEC('ALTER TABLE AccountCheckinsPendingChanges
ADD CONSTRAINT CK_AccountCheckinsPendingChanges_EndpointType CHECK (EndpointType IN (''standard'', ''custom''));');
