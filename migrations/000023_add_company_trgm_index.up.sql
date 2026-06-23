CREATE EXTENSION IF NOT EXISTS pg_trgm;

DROP INDEX IF EXISTS idx_company_info_cache_razao;
DROP INDEX IF EXISTS idx_company_info_cache_fantasia;

CREATE INDEX IF NOT EXISTS idx_company_info_cache_razao_trgm ON company_info_cache USING GIN (razao_social gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_company_info_cache_fantasia_trgm ON company_info_cache USING GIN (nome_fantasia gin_trgm_ops);
