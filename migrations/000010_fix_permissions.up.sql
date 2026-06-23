-- Ensure vyst_app user exists and has permissions
DO $$
BEGIN
    -- Create user if not exists
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_user WHERE usename = 'vyst_app') THEN
        CREATE USER vyst_app WITH PASSWORD 'vyst_app_secure_password';
    END IF;

    -- Grant permissions
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO vyst_app', current_database());
    GRANT USAGE ON SCHEMA public TO vyst_app;
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO vyst_app;
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vyst_app;
    GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO vyst_app;
    
    -- Specifically grant on the secure function
    GRANT EXECUTE ON FUNCTION get_user_by_email_secure(TEXT) TO vyst_app;

EXCEPTION WHEN insufficient_privilege THEN
    RAISE WARNING 'Insufficient privileges to manage vyst_app user. Skipping.';
END
$$;
