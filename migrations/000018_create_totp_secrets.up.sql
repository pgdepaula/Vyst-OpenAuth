-- Create table for TOTP secrets (Two-Factor Authentication)
CREATE TABLE totp_secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret VARCHAR(64) NOT NULL,
    enabled BOOLEAN DEFAULT false,
    backup_codes TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Only one TOTP secret per user
    UNIQUE(user_id)
);

-- Enable RLS
ALTER TABLE totp_secrets ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only access their own TOTP secrets
CREATE POLICY tenant_isolation_policy ON totp_secrets
    USING (user_id IN (SELECT id FROM users WHERE tenant_id = COALESCE(current_setting('app.current_tenant', true)::uuid, '00000000-0000-0000-0000-000000000000'::uuid)))
    WITH CHECK (user_id IN (SELECT id FROM users WHERE tenant_id = COALESCE(current_setting('app.current_tenant', true)::uuid, '00000000-0000-0000-0000-000000000000'::uuid)));

-- Force RLS
ALTER TABLE totp_secrets FORCE ROW LEVEL SECURITY;

-- Grant necessary permissions (only if role exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vyst_app_user') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON totp_secrets TO vyst_app_user;
    END IF;
END $$;

