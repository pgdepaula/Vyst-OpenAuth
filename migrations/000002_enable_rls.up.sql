-- Enable Row Level Security on users table
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Create policy to isolate users by tenant
-- USING: Controls which rows are visible in SELECT queries
-- WITH CHECK: Controls which rows can be inserted/updated
-- CRITICAL FIX: 
-- 1. Use current_setting with missing_ok=true to handle unset variable
-- 2. Use COALESCE to handle NULL (when not set, return impossible UUID)
-- 3. Cast both sides to UUID for proper comparison
CREATE POLICY tenant_isolation_policy ON users
    USING (tenant_id = COALESCE(current_setting('app.current_tenant', true)::uuid, '00000000-0000-0000-0000-000000000000'::uuid))
    WITH CHECK (tenant_id = COALESCE(current_setting('app.current_tenant', true)::uuid, '00000000-0000-0000-0000-000000000000'::uuid));

-- Force RLS to be applied even for the owner (superuser)
ALTER TABLE users FORCE ROW LEVEL SECURITY;
