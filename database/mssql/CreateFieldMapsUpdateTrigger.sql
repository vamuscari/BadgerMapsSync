CREATE OR ALTER TRIGGER DatasetsFieldMapsUpdateTrigger
ON DataSets
AFTER INSERT, UPDATE, DELETE
AS
BEGIN
    EXEC UpdateFieldMapsFromDatasets;
END;
