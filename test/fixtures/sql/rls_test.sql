-- Test RLS Configuration
-- This script verifies that Row Level Security is working correctly

-- 1. Check RLS status on users table
SELECT 
    relname AS table_name,
    relrowsecurity AS rls_enabled,
    relforcerowsecurity AS rls_forced
FROM pg_class
WHERE relname = 'users';

-- 2. List all policies on users table
SELECT 
    schemaname,
    tablename,
    policyname,
    permissive,
    roles,
    cmd,
    qual,
    with_check
FROM pg_policies
WHERE tablename = 'users';

-- 3. Create test tenants
INSERT INTO tenants (id, name) VALUES 
    ('11111111-1111-1111-1111-111111111111', 'Test Tenant A'),
    ('22222222-2222-2222-2222-222222222222', 'Test Tenant B')
ON CONFLICT (id) DO NOTHING;

-- 4. Set tenant A context and create user
BEGIN;
SET LOCAL app.current_tenant = '11111111-1111-1111-1111-111111111111';
INSERT INTO users (id, tenant_id, email, password_hash) VALUES 
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'test@tenanta.com', 'hash123');
COMMIT;

-- 5. Test: Query from Tenant A context (should see 1 user)
BEGIN;
SET LOCAL app.current_tenant = '11111111-1111-1111-1111-111111111111';
SELECT COUNT(*) AS tenant_a_sees FROM users WHERE id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa';
ROLLBACK;

-- 6. Test: Query from Tenant B context (should see 0 users - RLS should block)
BEGIN;
SET LOCAL app.current_tenant = '22222222-2222-2222-2222-222222222222';
SELECT COUNT(*) AS tenant_b_sees FROM users WHERE id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa';
ROLLBACK;

-- 7. Test: Query without setting tenant (should see 0 if FORCE RLS is enabled)
SELECT COUNT(*) AS no_tenant_context FROM users WHERE id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa';

-- Cleanup
DELETE FROM users WHERE id = 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa';
