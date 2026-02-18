IF OBJECT_ID('dbo.AccountsWithLabelsView', 'P') IS NULL
    EXEC(N'CREATE PROCEDURE dbo.AccountsWithLabelsView AS BEGIN SET NOCOUNT ON; END;');

EXEC(N'
ALTER PROCEDURE dbo.AccountsWithLabelsView
AS
BEGIN
    SET NOCOUNT ON;

    DECLARE @view_sql NVARCHAR(MAX);
    DECLARE @select_list NVARCHAR(MAX);
    DECLARE @profileId_str NVARCHAR(MAX);
    DECLARE @profileId INT;

    -- Get ProfileId from Configurations table
    SELECT @profileId_str = SettingValue
    FROM dbo.Configurations
    WHERE SettingKey = ''ApiProfileId'';
    SET @profileId = TRY_CAST(@profileId_str AS INT);

    -- Build the SELECT list dynamically (compatible with pre-2017 SQL Server versions)
    SELECT @select_list = STUFF((
        SELECT '', '' +
            CASE
                WHEN ds.Label IS NOT NULL AND ds.Label <> '''' THEN
                    ''a.'' + QUOTENAME(c.COLUMN_NAME) + '' AS '' + QUOTENAME(ds.Label)
                ELSE
                    ''a.'' + QUOTENAME(c.COLUMN_NAME)
            END
        FROM INFORMATION_SCHEMA.COLUMNS c
        LEFT JOIN dbo.DataSets ds
            ON c.COLUMN_NAME = ds.AccountField
            AND ds.ProfileId = @profileId
        WHERE c.TABLE_NAME = ''Accounts''
        ORDER BY c.ORDINAL_POSITION
        FOR XML PATH(''''), TYPE
    ).value(''.'', ''NVARCHAR(MAX)''), 1, 2, '''');

    IF @select_list IS NULL OR LTRIM(RTRIM(@select_list)) = ''''
    BEGIN
        SET @select_list = ''a.*'';
    END;

    IF OBJECT_ID(''dbo.AccountsWithLabels'', ''V'') IS NOT NULL
        DROP VIEW dbo.AccountsWithLabels;

    SET @view_sql = N''CREATE VIEW dbo.AccountsWithLabels AS SELECT '' + @select_list + N'' FROM dbo.Accounts a;'';
    EXEC sp_executesql @view_sql;
END;
');
