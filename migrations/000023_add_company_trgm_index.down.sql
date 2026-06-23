-- Disable transaction for concurrent index manipulation
COMMIT;
DROP INDEX CONCURRENTLY IF NOT EXISTS idx_company_info_cache_razao_trgm;
DROP INDEX CONCURRENTLY IF NOT EXISTS idx_company_info_cache_fantasia_trgm;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_company_info_cache_razao ON company_info_cache (razao_social);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_company_info_cache_fantasia ON company_info_cache (nome_fantasia);
BEGIN;
