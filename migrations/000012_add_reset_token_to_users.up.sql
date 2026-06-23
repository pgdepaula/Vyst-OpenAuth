ALTER TABLE users ADD COLUMN reset_token VARCHAR(255);
ALTER TABLE users ADD COLUMN reset_token_expires_at TIMESTAMP WITH TIME ZONE;

-- Update the secure function to include new columns
DROP FUNCTION IF EXISTS get_user_by_email_secure(TEXT);

CREATE OR REPLACE FUNCTION get_user_by_email_secure(p_email TEXT)
RETURNS TABLE (
    id UUID,
    email TEXT,
    password_hash TEXT,
    tenant_id UUID,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    reset_token TEXT,
    reset_token_expires_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT id, email, password_hash, tenant_id, created_at, updated_at, reset_token, reset_token_expires_at
    FROM users
    WHERE email = p_email;
$$;

-- Create the user if it doesn't exist (idempotent)
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

DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'vyst_app') THEN
    GRANT EXECUTE ON FUNCTION get_user_by_email_secure(TEXT) TO vyst_app;
  END IF;
END
$$;
