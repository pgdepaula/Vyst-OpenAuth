-- Update RLS policy to allow bypass for login
DROP POLICY IF EXISTS tenant_isolation_policy ON users;

CREATE POLICY tenant_isolation_policy ON users
    USING (
        tenant_id = COALESCE(current_setting('app.current_tenant', true)::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
        OR current_setting('app.bypass_rls', true) = 'on'
    )
    WITH CHECK (
        tenant_id = COALESCE(current_setting('app.current_tenant', true)::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
        OR current_setting('app.bypass_rls', true) = 'on'
    );

-- Update the secure lookup function to use the bypass
CREATE OR REPLACE FUNCTION get_user_by_email_secure(p_email TEXT)
RETURNS TABLE (
    id UUID,
    email TEXT,
    password_hash TEXT,
    tenant_id UUID,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    -- Set bypass flag for this transaction
    PERFORM set_config('app.bypass_rls', 'on', true);
    
    RETURN QUERY
    SELECT u.id, u.email, u.password_hash, u.tenant_id, u.created_at, u.updated_at
    FROM users u
    WHERE u.email = p_email;
END;
$$;



DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'vyst_app') THEN
    GRANT EXECUTE ON FUNCTION get_user_by_email_secure(TEXT) TO vyst_app;
  END IF;
END
$$;
