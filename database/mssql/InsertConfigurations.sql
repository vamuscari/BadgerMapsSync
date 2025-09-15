MERGE Configurations AS target
USING (VALUES
('ApiProfileId', ''),
('ApiProfileName', ''),
('CompanyId', ''),
('CompanyName', ''),
('SqlDbUserName', '')
) AS source (SettingKey, SettingValue)
ON (target.SettingKey = source.SettingKey)
WHEN NOT MATCHED THEN
    INSERT (SettingKey, SettingValue)
    VALUES (source.SettingKey, source.SettingValue);