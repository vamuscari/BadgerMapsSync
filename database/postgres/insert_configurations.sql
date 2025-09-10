INSERT INTO Configurations (SettingKey, SettingValue) VALUES
('ApiProfileId', ''),
('ApiProfileName', ''),
('CompanyId', ''),
('CompanyName', ''),
('SqlDbUserName', '')
ON CONFLICT (SettingKey) DO NOTHING;