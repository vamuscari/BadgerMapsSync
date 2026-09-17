CREATE OR REPLACE FUNCTION AccountsIndexedView()
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
    SELECT settingvalue INTO profile_id_str
    FROM configurations
    WHERE settingkey = 'ApiProfileId';

    IF profile_id_str IS NOT NULL AND profile_id_str != '' THEN
        profile_id := profile_id_str::integer;
    ELSE
        profile_id := NULL;
    END IF;

    WITH account_columns AS (
        SELECT
            c.column_name,
            c.ordinal_position,
            lower(c.column_name) LIKE 'customtext%'
                OR lower(c.column_name) LIKE 'customnumeric%' AS is_custom,
            MAX(
                CASE
                    WHEN lower(c.column_name) LIKE 'customtext%'
                        OR lower(c.column_name) LIKE 'customnumeric%'
                    THEN c.ordinal_position
                END
            ) OVER () AS last_custom
        FROM information_schema.columns c
        WHERE c.table_schema = current_schema() AND c.table_name = 'accounts'
    ), selected_columns AS (
        SELECT
            account_columns.column_name,
            account_columns.ordinal_position,
            account_columns.is_custom,
            account_columns.last_custom,
            ds.label,
            ds.position
        FROM account_columns
        LEFT JOIN datasets ds
            ON lower(account_columns.column_name) = lower(ds.accountfield)
            AND ds.profileid = profile_id
        WHERE NOT account_columns.is_custom OR ds.accountfield IS NOT NULL
    )
    SELECT string_agg(
               CASE
                   WHEN label IS NOT NULL AND label != '' THEN format('a.%I AS %I', column_name, label)
                   ELSE format('a.%I', column_name)
               END,
               ', ' ORDER BY
                   CASE
                       WHEN is_custom THEN 1
                       WHEN last_custom IS NOT NULL AND ordinal_position > last_custom THEN 2
                       ELSE 0
                   END,
                   CASE WHEN is_custom AND position IS NULL THEN 1 ELSE 0 END,
                   CASE WHEN is_custom THEN position ELSE ordinal_position END,
                   ordinal_position
           ),
           array_agg(
               COALESCE(NULLIF(label, ''), column_name)
               ORDER BY
                   CASE
                       WHEN is_custom THEN 1
                       WHEN last_custom IS NOT NULL AND ordinal_position > last_custom THEN 2
                       ELSE 0
                   END,
                   CASE WHEN is_custom AND position IS NULL THEN 1 ELSE 0 END,
                   CASE WHEN is_custom THEN position ELSE ordinal_position END,
                   ordinal_position
           )
    INTO select_list, desired_columns
    FROM selected_columns;

    IF select_list IS NULL OR btrim(select_list) = '' THEN
        RAISE EXCEPTION 'cannot create AccountsIndexed because Accounts has no columns';
    END IF;

    IF to_regclass('"AccountsIndexed"') IS NOT NULL AND to_regclass('accountsindexed') IS NULL THEN
        EXECUTE 'ALTER VIEW "AccountsIndexed" RENAME TO accountsindexed';
    END IF;

    view_sql := 'CREATE OR REPLACE VIEW accountsindexed AS SELECT ' || select_list || ' FROM accounts a;';

    IF to_regclass('accountsindexed') IS NOT NULL THEN
        SELECT pg_get_userbyid(c.relowner)
        INTO view_owner
        FROM pg_class c
        WHERE c.oid = 'accountsindexed'::regclass;

        SELECT array_agg(format(
                   'GRANT %s ON TABLE accountsindexed TO %s%s',
                   privilege_type,
                   CASE WHEN grantee = 'PUBLIC' THEN 'PUBLIC' ELSE quote_ident(grantee) END,
                   CASE WHEN is_grantable = 'YES' THEN ' WITH GRANT OPTION' ELSE '' END
               ))
        INTO grant_statements
        FROM information_schema.table_privileges
        WHERE table_schema = current_schema() AND table_name = 'accountsindexed';

        SELECT array_agg(attname ORDER BY attnum)
        INTO existing_columns
        FROM pg_attribute
        WHERE attrelid = 'accountsindexed'::regclass AND attnum > 0 AND NOT attisdropped;

        IF cardinality(existing_columns) = cardinality(desired_columns) THEN
            BEGIN
                FOR column_index IN 1..cardinality(existing_columns) LOOP
                    EXECUTE format(
                        'ALTER VIEW accountsindexed RENAME COLUMN %I TO %I',
                        existing_columns[column_index],
                        temporary_prefix || column_index
                    );
                END LOOP;
                FOR column_index IN 1..cardinality(desired_columns) LOOP
                    EXECUTE format(
                        'ALTER VIEW accountsindexed RENAME COLUMN %I TO %I',
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

        EXECUTE 'DROP VIEW accountsindexed';
        EXECUTE view_sql;
        IF view_owner IS NOT NULL THEN
            EXECUTE format('ALTER VIEW accountsindexed OWNER TO %I', view_owner);
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
