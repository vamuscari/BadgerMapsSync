IF OBJECT_ID('dbo.AccountsIndexedView', 'P') IS NULL
    EXEC(N'CREATE PROCEDURE dbo.AccountsIndexedView AS BEGIN SET NOCOUNT ON; END;');

EXEC(N'
ALTER PROCEDURE dbo.AccountsIndexedView
AS
BEGIN
    SET NOCOUNT ON;

    DECLARE @view_sql NVARCHAR(MAX);
    DECLARE @select_list NVARCHAR(MAX);
    DECLARE @profileId_str NVARCHAR(MAX);
    DECLARE @profileId INT;

    SELECT @profileId_str = SettingValue
    FROM dbo.Configurations
    WHERE SettingKey = ''ApiProfileId'';
    SET @profileId = TRY_CAST(@profileId_str AS INT);

    IF EXISTS (
        SELECT 1
        FROM dbo.DataSets ds
        JOIN INFORMATION_SCHEMA.COLUMNS mapped_column
          ON mapped_column.TABLE_SCHEMA = ''dbo''
          AND mapped_column.TABLE_NAME = ''Accounts''
          AND LOWER(mapped_column.COLUMN_NAME) = LOWER(ds.AccountField)
        WHERE ds.ProfileId = @profileId
          AND ds.Label IS NOT NULL
          AND ds.Label <> ''''
          AND DATALENGTH(ds.Label) > 256
    )
        THROW 50001, ''A DataSets label exceeds SQL Server''''s 128-character identifier limit.'', 1;

    SELECT @select_list = STUFF((
        SELECT '', '' +
            CASE
                WHEN ds.Label IS NOT NULL AND ds.Label <> '''' THEN
                    ''a.'' + QUOTENAME(account_column.COLUMN_NAME) + '' AS '' + QUOTENAME(ds.Label)
                ELSE
                    ''a.'' + QUOTENAME(account_column.COLUMN_NAME)
            END
        FROM (
            SELECT
                c.COLUMN_NAME,
                c.ORDINAL_POSITION,
                CASE
                    WHEN LOWER(c.COLUMN_NAME) LIKE ''customtext%''
                        OR LOWER(c.COLUMN_NAME) LIKE ''customnumeric%'' THEN 1
                    ELSE 0
                END AS IsCustom,
                MAX(
                    CASE
                        WHEN LOWER(c.COLUMN_NAME) LIKE ''customtext%''
                            OR LOWER(c.COLUMN_NAME) LIKE ''customnumeric%''
                        THEN c.ORDINAL_POSITION
                    END
                ) OVER () AS LastCustom
            FROM INFORMATION_SCHEMA.COLUMNS c
            WHERE c.TABLE_SCHEMA = ''dbo'' AND c.TABLE_NAME = ''Accounts''
        ) account_column
        LEFT JOIN dbo.DataSets ds
            ON LOWER(account_column.COLUMN_NAME) = LOWER(ds.AccountField)
            AND ds.ProfileId = @profileId
        WHERE account_column.IsCustom = 0 OR ds.AccountField IS NOT NULL
        ORDER BY
            CASE
                WHEN account_column.IsCustom = 1 THEN 1
                WHEN account_column.LastCustom IS NOT NULL
                    AND account_column.ORDINAL_POSITION > account_column.LastCustom THEN 2
                ELSE 0
            END,
            CASE WHEN account_column.IsCustom = 1 AND ds.Position IS NULL THEN 1 ELSE 0 END,
            CASE
                WHEN account_column.IsCustom = 1 THEN ds.Position
                ELSE account_column.ORDINAL_POSITION
            END,
            account_column.ORDINAL_POSITION
        FOR XML PATH(''''), TYPE
    ).value(''.'', ''NVARCHAR(MAX)''), 1, 2, '''');

    IF @select_list IS NULL OR LTRIM(RTRIM(@select_list)) = ''''
        THROW 50002, ''Cannot create AccountsIndexed because Accounts has no columns.'', 1;

    IF OBJECT_ID(''dbo.AccountsIndexed'', ''V'') IS NULL
        SET @view_sql = N''CREATE VIEW dbo.AccountsIndexed AS SELECT '' + @select_list + N'' FROM dbo.Accounts a;'';
    ELSE
        SET @view_sql = N''ALTER VIEW dbo.AccountsIndexed AS SELECT '' + @select_list + N'' FROM dbo.Accounts a;'';

    EXEC sp_executesql @view_sql;
END;
');
