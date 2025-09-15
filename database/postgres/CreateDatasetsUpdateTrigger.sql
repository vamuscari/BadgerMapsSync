CREATE OR REPLACE FUNCTION RefreshAccountsWithLabelsView()
RETURNS TRIGGER AS $
BEGIN
    PERFORM AccountsWithLabelsView();
    RETURN NULL;
END;
$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS DatasetsUpdateTrigger ON "DataSets";

CREATE TRIGGER DatasetsUpdateTrigger
AFTER INSERT OR UPDATE OR DELETE ON "DataSets"
FOR EACH STATEMENT
EXECUTE FUNCTION RefreshAccountsWithLabelsView();
