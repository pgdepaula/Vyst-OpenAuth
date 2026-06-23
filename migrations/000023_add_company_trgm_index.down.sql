DROP INDEX IF EXISTS idx_company_info_cache_razao_trgm;
DROP INDEX IF EXISTS idx_company_info_cache_fantasia_trgm;

CREATE INDEX IF NOT EXISTS idx_company_info_cache_razao ON company_info_cache (razao_social);
CREATE INDEX IF NOT EXISTS idx_company_info_cache_fantasia ON company_info_cache (nome_fantasia);
