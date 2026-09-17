CREATE OR REPLACE FUNCTION AccountsWithLabelsView()
RETURNS void AS $$
DECLARE
    view_sql TEXT;
    select_list TEXT;
    profile_id_str TEXT;
    profile_id INT;
    desired_columns TEXT[];
    existing_columns TEXT[];
    column_index INT;
    temporary_prefix TEXT := '__badgermaps_' || txid_current() || '_';
    view_owner TEXT;
    grant_statements TEXT[];
    grant_statement TEXT;
BEGIN
    SELECT settingvalue INTO profile_id_str FROM configurations WHERE settingkey = 'ApiProfileId';
    IF profile_id_str IS NOT NULL AND profile_id_str != '' THEN
        profile_id := profile_id_str::integer;
    ELSE
        profile_id := NULL;
    END IF;

    SELECT string_agg(
               CASE
                   WHEN ds.label IS NOT NULL AND ds.label != '' THEN format('a.%I AS %I', c.column_name, ds.label)
                   ELSE format('a.%I', c.column_name)
               END,
               ', ' ORDER BY c.ordinal_position
           ),
           array_agg(COALESCE(NULLIF(ds.label, ''), c.column_name) ORDER BY c.ordinal_position)
    INTO select_list, desired_columns
    FROM information_schema.columns c
    LEFT JOIN datasets ds ON lower(c.column_name) = lower(ds.accountfield) AND ds.profileid = profile_id
    WHERE c.table_schema = current_schema() AND c.table_name = 'accounts';

    IF select_list IS NULL OR btrim(select_list) = '' THEN
        RAISE EXCEPTION 'cannot create AccountsWithLabels because Accounts has no columns';
    END IF;

    IF to_regclass('"AccountsWithLabels"') IS NOT NULL AND to_regclass('accountswithlabels') IS NULL THEN
        EXECUTE 'ALTER VIEW "AccountsWithLabels" RENAME TO accountswithlabels';
    END IF;

    view_sql := 'CREATE OR REPLACE VIEW accountswithlabels AS SELECT ' || select_list || ' FROM accounts a;';

    IF to_regclass('accountswithlabels') IS NOT NULL THEN
        SELECT pg_get_userbyid(c.relowner)
        INTO view_owner
        FROM pg_class c
        WHERE c.oid = 'accountswithlabels'::regclass;

        SELECT array_agg(format(
                   'GRANT %s ON TABLE accountswithlabels TO %s%s',
                   privilege_type,
                   CASE WHEN grantee = 'PUBLIC' THEN 'PUBLIC' ELSE quote_ident(grantee) END,
                   CASE WHEN is_grantable = 'YES' THEN ' WITH GRANT OPTION' ELSE '' END
               ))
        INTO grant_statements
        FROM information_schema.table_privileges
        WHERE table_schema = current_schema() AND table_name = 'accountswithlabels';

        SELECT array_agg(attname ORDER BY attnum)
        INTO existing_columns
        FROM pg_attribute
        WHERE attrelid = 'accountswithlabels'::regclass AND attnum > 0 AND NOT attisdropped;

        IF cardinality(existing_columns) = cardinality(desired_columns) THEN
            BEGIN
                FOR column_index IN 1..cardinality(existing_columns) LOOP
                    EXECUTE format(
                        'ALTER VIEW accountswithlabels RENAME COLUMN %I TO %I',
                        existing_columns[column_index],
                        temporary_prefix || column_index
                    );
                END LOOP;
                FOR column_index IN 1..cardinality(desired_columns) LOOP
                    EXECUTE format(
                        'ALTER VIEW accountswithlabels RENAME COLUMN %I TO %I',
                        temporary_prefix || column_index,
                        desired_columns[column_index]
                    );
                END LOOP;
                EXECUTE view_sql;
                RETURN;
            EXCEPTION WHEN invalid_table_definition THEN
                NULL;
            END;
        END IF;

        EXECUTE 'DROP VIEW accountswithlabels';
        EXECUTE view_sql;
        IF view_owner IS NOT NULL THEN
            EXECUTE format('ALTER VIEW accountswithlabels OWNER TO %I', view_owner);
        END IF;
        IF grant_statements IS NOT NULL THEN
            FOREACH grant_statement IN ARRAY grant_statements LOOP
                EXECUTE grant_statement;
            END LOOP;
        END IF;
        RETURN;
    END IF;

    EXECUTE view_sql;
END;
$$ LANGUAGE plpgsql;
