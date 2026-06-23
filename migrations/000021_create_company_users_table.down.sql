-- Rollback: Drop company_users table and user identity columns

-- Remove indices first
DROP INDEX IF EXISTS idx_users_active_company_id;

-- Remove constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS valid_identity_type;

-- Remove columns from users table
ALTER TABLE users DROP COLUMN IF EXISTS active_company_id;
ALTER TABLE users DROP COLUMN IF EXISTS identity_type;

-- Drop company_users table
DROP TABLE IF EXISTS company_users;
