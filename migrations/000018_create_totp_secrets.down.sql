-- Drop TOTP secrets table
DROP POLICY IF EXISTS tenant_isolation_policy ON totp_secrets;
DROP TABLE IF EXISTS totp_secrets;
