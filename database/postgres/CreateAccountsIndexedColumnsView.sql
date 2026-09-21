CREATE OR REPLACE VIEW accountsindexedcolumns AS
WITH account_columns AS (
    SELECT
        c.column_name,
        c.data_type,
        c.ordinal_position,
        CASE
            WHEN lower(c.column_name) LIKE 'customtext%'
                OR lower(c.column_name) LIKE 'customnumeric%'
            THEN 1
            ELSE 0
        END AS is_custom,
        MAX(
            CASE
                WHEN lower(c.column_name) LIKE 'customtext%'
                    OR lower(c.column_name) LIKE 'customnumeric%'
                THEN c.ordinal_position
            END
        ) OVER () AS last_custom
    FROM information_schema.columns c
    WHERE c.table_schema = current_schema()
      AND lower(c.table_name) = 'accounts'
)
SELECT
    COALESCE(NULLIF(data_set.Label, ''), account_column.column_name)::TEXT AS name,
    COALESCE(NULLIF(data_set.Type, ''), account_column.data_type)::TEXT AS type,
    CAST(ROW_NUMBER() OVER (
        ORDER BY
            CASE
                WHEN account_column.is_custom = 1 THEN 1
                WHEN account_column.last_custom IS NOT NULL
                    AND account_column.ordinal_position > account_column.last_custom THEN 2
                ELSE 0
            END,
            CASE WHEN account_column.is_custom = 1 AND data_set.Position IS NULL THEN 1 ELSE 0 END,
            CASE
                WHEN account_column.is_custom = 1 THEN data_set.Position
                ELSE account_column.ordinal_position
            END,
            account_column.ordinal_position
    ) AS INTEGER) AS position
FROM account_columns AS account_column
LEFT JOIN DataSets AS data_set
    ON lower(data_set.AccountField) = lower(account_column.column_name)
    AND data_set.ProfileId = (
        SELECT CAST(NULLIF(SettingValue, '') AS INTEGER)
        FROM Configurations
        WHERE SettingKey = 'ApiProfileId'
    )
WHERE account_column.is_custom = 0 OR data_set.AccountField IS NOT NULL;
