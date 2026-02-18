IF OBJECT_ID('dbo.DatasetsUpdateTrigger', 'TR') IS NOT NULL
    DROP TRIGGER dbo.DatasetsUpdateTrigger;

EXEC(N'
CREATE TRIGGER dbo.DatasetsUpdateTrigger
ON dbo.DataSets
AFTER INSERT, UPDATE, DELETE
AS
BEGIN
    SET NOCOUNT ON;
    EXEC dbo.AccountsWithLabelsView;
END;
');
