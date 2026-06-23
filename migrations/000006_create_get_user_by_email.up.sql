-- Create a secure function to look up users by email bypassing RLS
-- This is required for the login flow where the tenant context is not yet known
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

-- Grant execute permission to the application user
GRANT EXECUTE ON FUNCTION get_user_by_email_secure(TEXT) TO vyst_app;
