CREATE OR REPLACE FUNCTION refresh_field_maps()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM update_field_maps_from_datasets();
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS datasets_field_maps_update_trigger ON "DataSets";

CREATE TRIGGER datasets_field_maps_update_trigger
AFTER INSERT OR UPDATE OR DELETE ON "DataSets"
FOR EACH STATEMENT
EXECUTE FUNCTION refresh_field_maps();
