CREATE OR ALTER PROCEDURE AccountsWithLabelsView AS
BEGIN
    DECLARE @view_sql NVARCHAR(MAX);
    DECLARE @select_list NVARCHAR(MAX);
    DECLARE @profileId_str NVARCHAR(MAX);
    DECLARE @profileId INT;

    -- Get ProfileId from Configurations table
    SELECT @profileId_str = SettingValue FROM Configurations WHERE SettingKey = 'ApiProfileId';
    SET @profileId = TRY_CAST(@profileId_str AS INT);

    -- Build the SELECT list dynamically
    SELECT @select_list = STRING_AGG(
        CASE
            WHEN ds.Label IS NOT NULL AND ds.Label != '' THEN CONCAT('a.', QUOTENAME(c.COLUMN_NAME), ' AS ', QUOTENAME(ds.Label))
            ELSE CONCAT('a.', QUOTENAME(c.COLUMN_NAME))
        END,
        ', '
    )
    FROM information_schema.columns c
    LEFT JOIN DataSets ds ON c.COLUMN_NAME = ds.AccountField AND ds.ProfileId = @profileId
    WHERE c.TABLE_NAME = 'Accounts';

    SET @view_sql = 'CREATE OR ALTER VIEW AccountsWithLabels AS SELECT ' + @select_list + ' FROM Accounts a;';

    EXEC sp_executesql @view_sql;
END;
