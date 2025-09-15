CREATE OR REPLACE FUNCTION UpdateFieldMapsFromDatasets()
RETURNS void AS $
BEGIN
    UPDATE FieldMaps fm
    SET
        DataSetName = ds.Name,
        DataSetLabel = ds.Label
    FROM DataSets ds
    WHERE fm.FieldName = ds.AccountField AND fm.ObjectType = 'Account';
END;
$ LANGUAGE plpgsql;
