-- Update the secure function to include all columns expected by user_repo.go
-- The existing function only returns 6 columns, but user_repo.go expects 11:
-- id, email, password_hash, tenant_id, created_at, updated_at, reset_token, reset_token_expires_at, status, verification_token, verification_token_expires_at

DROP FUNCTION IF EXISTS get_user_by_email_secure(TEXT) CASCADE;

CREATE OR REPLACE FUNCTION get_user_by_email_secure(p_email TEXT)
RETURNS TABLE (
    id UUID,
    email TEXT,
    password_hash TEXT,
    tenant_id UUID,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    reset_token TEXT,
    reset_token_expires_at TIMESTAMPTZ,
    status TEXT,
    verification_token TEXT,
    verification_token_expires_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT 
        id, 
        email, 
        password_hash, 
        tenant_id, 
        created_at, 
        updated_at, 
        reset_token::TEXT, 
        reset_token_expires_at, 
        status::TEXT, 
        verification_token::TEXT, 
        verification_token_expires_at
    FROM users
    WHERE email = p_email;
$$;

DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'vyst_app') THEN
    GRANT EXECUTE ON FUNCTION get_user_by_email_secure(TEXT) TO vyst_app;
  END IF;
END
$$;
