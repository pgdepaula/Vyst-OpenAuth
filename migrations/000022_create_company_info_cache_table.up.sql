CREATE TABLE IF NOT EXISTS company_info_cache (
    cnpj VARCHAR(14) PRIMARY KEY,
    razao_social VARCHAR(255) NOT NULL,
    nome_fantasia VARCHAR(255),
    situacao VARCHAR(50) NOT NULL,
    natureza_juridica VARCHAR(255),
    data_abertura DATE,
    logradouro VARCHAR(255),
    numero VARCHAR(50),
    complemento VARCHAR(255),
    bairro VARCHAR(100),
    cidade VARCHAR(100),
    uf VARCHAR(2),
    cep VARCHAR(8),
    telefones JSONB,
    emails JSONB,
    cnae_principal VARCHAR(20),
    last_fetched_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_company_info_cache_razao ON company_info_cache (razao_social);
CREATE INDEX IF NOT EXISTS idx_company_info_cache_fantasia ON company_info_cache (nome_fantasia);
