CREATE OR ALTER TRIGGER datasets_update_trigger
ON DataSets
AFTER INSERT, UPDATE, DELETE
AS
BEGIN
    EXEC AccountsWithLabelsView
END
