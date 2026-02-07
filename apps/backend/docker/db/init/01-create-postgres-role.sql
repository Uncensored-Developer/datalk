DO $$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'postgres') THEN
            CREATE ROLE postgres WITH LOGIN SUPERUSER PASSWORD 'postgres';
        END IF;
    END
$$;