CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(32) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    scopes TEXT[],
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Index for fast lookup by prefix (since we can't look up by hash directly)
    CONSTRAINT unique_key_prefix UNIQUE (key_prefix)
);

CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);

-- RLS Policies
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;

-- Users can view keys for their tenant (if they have permission, handled by app logic usually, but RLS is good backup)
-- For simplicity, we'll allow users to see keys they own or if they are tenant admins.
-- Here we just check tenant_id matches current_setting('app.current_tenant')
CREATE POLICY "Users can view keys in their tenant" ON api_keys
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid);

-- Only app logic creates/deletes usually, but if we want RLS:
CREATE POLICY "Users can manage keys in their tenant" ON api_keys
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid);

-- Grant permissions
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
    GRANT SELECT, INSERT, UPDATE, DELETE ON api_keys TO vyst_app;
  END IF;
END
$$;
