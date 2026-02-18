IF OBJECT_ID('dbo.UpdateFieldMapsFromDatasets', 'P') IS NULL
    EXEC(N'CREATE PROCEDURE dbo.UpdateFieldMapsFromDatasets AS BEGIN SET NOCOUNT ON; END;');

EXEC(N'
ALTER PROCEDURE dbo.UpdateFieldMapsFromDatasets
AS
BEGIN
    SET NOCOUNT ON;

    UPDATE fm
    SET
        fm.DataSetName = ds.Name,
        fm.DataSetLabel = ds.Label
    FROM dbo.FieldMaps fm
    JOIN dbo.DataSets ds ON fm.FieldName = ds.AccountField
    WHERE fm.ObjectType = ''Account'';
END;
');
