CREATE OR ALTER TRIGGER datasets_field_maps_update_trigger
ON DataSets
AFTER INSERT, UPDATE, DELETE
AS
BEGIN
    EXEC update_field_maps_from_datasets;
END;
