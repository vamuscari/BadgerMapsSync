IF OBJECT_ID('dbo.DatasetsFieldMapsUpdateTrigger', 'TR') IS NOT NULL
    DROP TRIGGER dbo.DatasetsFieldMapsUpdateTrigger;

EXEC(N'
CREATE TRIGGER dbo.DatasetsFieldMapsUpdateTrigger
ON dbo.DataSets
AFTER INSERT, UPDATE, DELETE
AS
BEGIN
    SET NOCOUNT ON;
    EXEC dbo.UpdateFieldMapsFromDatasets;
END;
');
