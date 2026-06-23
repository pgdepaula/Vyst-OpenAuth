-- Revert RLS policy
DROP POLICY IF EXISTS tenant_isolation_policy ON users;

CREATE POLICY tenant_isolation_policy ON users
    USING (tenant_id = COALESCE(current_setting('app.current_tenant', true)::uuid, '00000000-0000-0000-0000-000000000000'::uuid))
    WITH CHECK (tenant_id = COALESCE(current_setting('app.current_tenant', true)::uuid, '00000000-0000-0000-0000-000000000000'::uuid));

-- Revert function to SQL
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

$$;

DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'vyst_app') THEN
    GRANT EXECUTE ON FUNCTION get_user_by_email_secure(TEXT) TO vyst_app;
  END IF;
END
$$;
