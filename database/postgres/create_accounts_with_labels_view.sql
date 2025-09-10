CREATE OR REPLACE FUNCTION AccountsWithLabelsView()
RETURNS void AS $
DECLARE
    view_sql TEXT;
    select_list TEXT;
    profile_id_str TEXT;
    profile_id INT;
BEGIN
    -- Get ProfileId from Configurations table
    SELECT "SettingValue" INTO profile_id_str FROM "Configurations" WHERE "SettingKey" = 'ApiProfileId';
    IF profile_id_str IS NOT NULL AND profile_id_str != '' THEN
        profile_id := profile_id_str::integer;
    ELSE
        profile_id := NULL;
    END IF;

    -- Build the SELECT list dynamically
    SELECT string_agg(
        CASE
            WHEN ds."Label" IS NOT NULL AND ds."Label" != '' THEN format('a.%I AS %I', c.column_name, ds."Label")
            ELSE format('a.%I', c.column_name)
        END,
        ', '
    )
    INTO select_list
    FROM information_schema.columns c
    LEFT JOIN "DataSets" ds ON c.column_name = ds."AccountField" AND ds."ProfileId" = profile_id
    WHERE c.table_schema = 'public' AND c.table_name = 'Accounts';

    view_sql := 'CREATE OR REPLACE VIEW "AccountsWithLabels" AS SELECT ' || select_list || ' FROM "Accounts" a;';

    EXECUTE view_sql;
END;
$ LANGUAGE plpgsql;
