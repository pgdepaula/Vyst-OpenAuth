-- Disable FORCE RLS so the table owner (postgres) can bypass it automatically
ALTER TABLE users NO FORCE ROW LEVEL SECURITY;

-- Ensure vyst_app user exists (retry from migration 10 failure)
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
EXCEPTION WHEN insufficient_privilege THEN
    RAISE WARNING 'Insufficient privileges to manage vyst_app user. Skipping.';
END
$$;

-- Revert the function to simple SQL (no set_config needed)
-- It will run as the owner (SECURITY DEFINER), who now bypasses RLS
CREATE OR REPLACE FUNCTION get_user_by_email_secure(p_email TEXT)
RETURNS TABLE (
    id UUID,
    email TEXT,
    password_hash TEXT,
    tenant_id UUID,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT id, email, password_hash, tenant_id, created_at, updated_at
    FROM users
    WHERE email = p_email;
$$;

-- Ensure permissions (just in case)
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'vyst_app') THEN
        GRANT EXECUTE ON FUNCTION get_user_by_email_secure(TEXT) TO vyst_app;
    END IF;
END
$$;
