-- Rollback: Drop companies table and related objects

DROP TRIGGER IF EXISTS trigger_companies_updated_at ON companies;
DROP FUNCTION IF EXISTS update_companies_updated_at();
DROP POLICY IF EXISTS companies_tenant_isolation ON companies;
DROP TABLE IF EXISTS companies;
