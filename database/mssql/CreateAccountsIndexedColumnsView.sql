CREATE OR ALTER VIEW dbo.AccountsIndexedColumns
AS
WITH AccountColumns AS (
    SELECT
        c.COLUMN_NAME,
        c.DATA_TYPE,
        c.ORDINAL_POSITION,
        CASE
            WHEN LOWER(c.COLUMN_NAME) LIKE 'customtext%'
                OR LOWER(c.COLUMN_NAME) LIKE 'customnumeric%'
            THEN 1
            ELSE 0
        END AS IsCustom,
        MAX(
            CASE
                WHEN LOWER(c.COLUMN_NAME) LIKE 'customtext%'
                    OR LOWER(c.COLUMN_NAME) LIKE 'customnumeric%'
                THEN c.ORDINAL_POSITION
            END
        ) OVER () AS LastCustom
    FROM INFORMATION_SCHEMA.COLUMNS c
    WHERE c.TABLE_SCHEMA = 'dbo'
      AND c.TABLE_NAME = 'Accounts'
)
SELECT
    CAST(COALESCE(NULLIF(data_set.Label, ''), account_column.COLUMN_NAME) AS NVARCHAR(255)) AS [Name],
    CAST(COALESCE(NULLIF(data_set.[Type], ''), account_column.DATA_TYPE) AS NVARCHAR(255)) AS [Type],
    CAST(ROW_NUMBER() OVER (
        ORDER BY
            CASE
                WHEN account_column.IsCustom = 1 THEN 1
                WHEN account_column.LastCustom IS NOT NULL
                    AND account_column.ORDINAL_POSITION > account_column.LastCustom THEN 2
                ELSE 0
            END,
            CASE WHEN account_column.IsCustom = 1 AND data_set.Position IS NULL THEN 1 ELSE 0 END,
            CASE
                WHEN account_column.IsCustom = 1 THEN data_set.Position
                ELSE account_column.ORDINAL_POSITION
            END,
            account_column.ORDINAL_POSITION
    ) AS INT) AS [Position]
FROM AccountColumns AS account_column
LEFT JOIN dbo.DataSets AS data_set
    ON LOWER(data_set.AccountField) = LOWER(account_column.COLUMN_NAME)
    AND data_set.ProfileId = (
        SELECT TRY_CAST(SettingValue AS INT)
        FROM dbo.Configurations
        WHERE SettingKey = 'ApiProfileId'
    )
WHERE account_column.IsCustom = 0
   OR data_set.AccountField IS NOT NULL;
