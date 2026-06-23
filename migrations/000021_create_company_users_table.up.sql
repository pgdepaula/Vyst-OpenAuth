-- Migration: Create company_users table and add identity fields to users
-- This table implements the N:N relationship between users and companies.
-- A user can belong to multiple companies with different roles.

CREATE TABLE company_users (
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    
    PRIMARY KEY (company_id, user_id),
    
    -- Validate role values
    CONSTRAINT valid_company_user_role CHECK (role IN ('admin', 'member', 'viewer')),
    
    -- Validate status values
    CONSTRAINT valid_company_user_status CHECK (status IN ('pending', 'active', 'revoked'))
);

-- Performance indices for common queries
CREATE INDEX idx_company_users_user_id ON company_users(user_id);
CREATE INDEX idx_company_users_company_id ON company_users(company_id);
CREATE INDEX idx_company_users_status ON company_users(status);
CREATE INDEX idx_company_users_role ON company_users(role);

-- Add identity-related columns to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS identity_type VARCHAR(20) DEFAULT 'individual';
ALTER TABLE users ADD COLUMN IF NOT EXISTS active_company_id UUID REFERENCES companies(id) ON DELETE SET NULL;

-- Add constraint to validate identity_type
ALTER TABLE users ADD CONSTRAINT valid_identity_type CHECK (identity_type IN ('individual', 'company'));

-- Index for users with active company (for filtering company-context queries)
CREATE INDEX idx_users_active_company_id ON users(active_company_id) WHERE active_company_id IS NOT NULL;

-- Grant permissions to vyst_app user
GRANT SELECT, INSERT, UPDATE, DELETE ON company_users TO vyst_app;

COMMENT ON TABLE company_users IS 'N:N relationship between users and companies with role assignment';
COMMENT ON COLUMN company_users.role IS 'User role in the company: admin, member, or viewer';
COMMENT ON COLUMN company_users.status IS 'Membership status: pending (invited), active, or revoked';
COMMENT ON COLUMN company_users.invited_by IS 'User ID who invited this member to the company';
COMMENT ON COLUMN users.identity_type IS 'Type of user identity: individual (pessoa física) or company (pessoa jurídica)';
COMMENT ON COLUMN users.active_company_id IS 'Currently active company context for company-type logins';
