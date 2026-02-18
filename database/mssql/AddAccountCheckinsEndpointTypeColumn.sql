ALTER TABLE AccountCheckins
ADD EndpointType NVARCHAR(20) NOT NULL
    CONSTRAINT DF_AccountCheckins_EndpointType DEFAULT 'standard';

EXEC('ALTER TABLE AccountCheckins
ADD CONSTRAINT CK_AccountCheckins_EndpointType CHECK (EndpointType IN (''standard'', ''custom''));');
