-- Disable Row Level Security on users table
ALTER TABLE users DISABLE ROW LEVEL SECURITY;

-- Drop the policy
DROP POLICY IF EXISTS tenant_isolation_policy ON users;
