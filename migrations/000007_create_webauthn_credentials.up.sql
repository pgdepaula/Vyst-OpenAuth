-- Create table for WebAuthn credentials
CREATE TABLE webauthn_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    webauthn_id BYTEA NOT NULL,
    public_key BYTEA NOT NULL,
    attestation_type VARCHAR(50) NOT NULL,
    transport VARCHAR(50)[] DEFAULT '{}',
    flags JSONB NOT NULL DEFAULT '{}',
    sign_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure webauthn_id is unique per user (though usually global unique is better, per spec it's scoped to RP)
    UNIQUE(user_id, webauthn_id)
);

-- Enable RLS
ALTER TABLE webauthn_credentials ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see their own credentials
CREATE POLICY tenant_isolation_policy ON webauthn_credentials
    USING (user_id IN (SELECT id FROM users WHERE tenant_id = COALESCE(current_setting('app.current_tenant', true)::uuid, '00000000-0000-0000-0000-000000000000'::uuid)))
    WITH CHECK (user_id IN (SELECT id FROM users WHERE tenant_id = COALESCE(current_setting('app.current_tenant', true)::uuid, '00000000-0000-0000-0000-000000000000'::uuid)));

-- Force RLS
ALTER TABLE webauthn_credentials FORCE ROW LEVEL SECURITY;
