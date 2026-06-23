ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_token VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_token_expires_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_token VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_token_expires_at TIMESTAMP WITH TIME ZONE;

-- Update the secure function to include new columns AND status (which was missing in previous migrations)
-- Temporarily removed to isolate crash.
-- DROP FUNCTION IF EXISTS get_user_by_email_secure(TEXT) CASCADE;

-- CREATE OR REPLACE FUNCTION get_user_by_email_secure(p_email TEXT)
-- ...
-- $$;

-- DO $$
-- BEGIN
--   IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'vyst_app') THEN
--     GRANT EXECUTE ON FUNCTION get_user_by_email_secure(TEXT) TO vyst_app;
--   END IF;
-- END
-- $$;
