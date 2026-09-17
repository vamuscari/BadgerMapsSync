SELECT account_column.name, data_set.Label
FROM pragma_table_info('Accounts') AS account_column
LEFT JOIN DataSets AS data_set
    ON lower(data_set.AccountField) = lower(account_column.name)
    AND data_set.ProfileId = (
        SELECT CAST(NULLIF(SettingValue, '') AS INTEGER)
        FROM Configurations
        WHERE SettingKey = 'ApiProfileId'
    )
ORDER BY account_column.cid;
