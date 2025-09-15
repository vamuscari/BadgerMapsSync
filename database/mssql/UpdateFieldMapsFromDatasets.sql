CREATE OR ALTER PROCEDURE UpdateFieldMapsFromDatasets AS
BEGIN
    UPDATE fm
    SET
        fm.DataSetName = ds.Name,
        fm.DataSetLabel = ds.Label
    FROM FieldMaps fm
    JOIN DataSets ds ON fm.FieldName = ds.AccountField
    WHERE fm.ObjectType = 'Account';
END;
