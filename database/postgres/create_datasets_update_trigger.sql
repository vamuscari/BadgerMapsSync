CREATE OR REPLACE FUNCTION refresh_accounts_with_labels_view()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM AccountsWithLabelsView();
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS datasets_update_trigger ON "DataSets";

CREATE TRIGGER datasets_update_trigger
AFTER INSERT OR UPDATE OR DELETE ON "DataSets"
FOR EACH STATEMENT
EXECUTE FUNCTION refresh_accounts_with_labels_view();
