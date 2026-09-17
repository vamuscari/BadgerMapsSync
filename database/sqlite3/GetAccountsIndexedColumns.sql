WITH account_columns AS (
    SELECT
        account_column.cid,
        account_column.name,
        CASE
            WHEN lower(account_column.name) LIKE 'customtext%'
                OR lower(account_column.name) LIKE 'customnumeric%'
            THEN 1
            ELSE 0
        END AS is_custom
    FROM pragma_table_info('Accounts') AS account_column
),
custom_bounds AS (
    SELECT MIN(cid) AS first_custom, MAX(cid) AS last_custom
    FROM account_columns
    WHERE is_custom = 1
)
SELECT account_column.name, data_set.Label
FROM account_columns AS account_column
CROSS JOIN custom_bounds
LEFT JOIN DataSets AS data_set
    ON lower(data_set.AccountField) = lower(account_column.name)
    AND data_set.ProfileId = (
        SELECT CAST(NULLIF(SettingValue, '') AS INTEGER)
        FROM Configurations
        WHERE SettingKey = 'ApiProfileId'
    )
WHERE account_column.is_custom = 0 OR data_set.AccountField IS NOT NULL
ORDER BY
    CASE
        WHEN account_column.is_custom = 1 THEN 1
        WHEN custom_bounds.last_custom IS NOT NULL AND account_column.cid > custom_bounds.last_custom THEN 2
        ELSE 0
    END,
    CASE WHEN account_column.is_custom = 1 AND data_set.Position IS NULL THEN 1 ELSE 0 END,
    CASE WHEN account_column.is_custom = 1 THEN data_set.Position ELSE account_column.cid END,
    account_column.cid;
