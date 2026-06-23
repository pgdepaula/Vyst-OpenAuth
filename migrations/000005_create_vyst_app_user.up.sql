-- Create vyst_app user for production use
-- This user has limited permissions compared to postgres superuser

-- Create the user if it doesn't exist
DO $$
BEGIN
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_user WHERE usename = 'vyst_app') THEN
            CREATE USER vyst_app WITH PASSWORD 'vyst_app_secure_password';
        END IF;
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE WARNING 'Insufficient privileges to create user vyst_app. Skipping.';
    END;
END
$$;

-- Grant connection to the database
DO $$
BEGIN
    BEGIN
        EXECUTE format('GRANT CONNECT ON DATABASE %I TO vyst_app', current_database());
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE WARNING 'Insufficient privileges to grant CONNECT. Skipping.';
    END;
END
$$;

-- Grant usage on schema
DO $$
BEGIN
    GRANT USAGE ON SCHEMA public TO vyst_app;
EXCEPTION WHEN insufficient_privilege THEN
    RAISE WARNING 'Insufficient privileges to grant USAGE on schema. Skipping.';
END
$$;

-- Grant permissions on all tables
DO $$
BEGIN
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO vyst_app;
EXCEPTION WHEN insufficient_privilege THEN
    RAISE WARNING 'Insufficient privileges to grant table permissions. Skipping.';
END
$$;

-- Grant permissions on all sequences
DO $$
BEGIN
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vyst_app;
EXCEPTION WHEN insufficient_privilege THEN
    RAISE WARNING 'Insufficient privileges to grant sequence permissions. Skipping.';
END
$$;

-- Set default privileges for future tables
DO $$
BEGIN
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO vyst_app;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO vyst_app;
EXCEPTION WHEN insufficient_privilege THEN
    RAISE WARNING 'Insufficient privileges to alter default privileges. Skipping.';
END
$$;

-- Allow vyst_app to create temporary tables (needed for some operations)
DO $$
BEGIN
    BEGIN
        EXECUTE format('GRANT TEMPORARY ON DATABASE %I TO vyst_app', current_database());
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE WARNING 'Insufficient privileges to grant TEMPORARY. Skipping.';
    END;
END
$$;

-- Grant execute on all functions (needed for RLS and other operations)
DO $$
BEGIN
    GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO vyst_app;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO vyst_app;
EXCEPTION WHEN insufficient_privilege THEN
    RAISE WARNING 'Insufficient privileges to grant function permissions. Skipping.';
END
$$;
