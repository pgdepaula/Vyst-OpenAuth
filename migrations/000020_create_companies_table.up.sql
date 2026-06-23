-- Migration: Create companies table
-- Companies represent legal entities (pessoas jurídicas) in the system.
-- A company belongs to a tenant and can have multiple users as members.

CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    cnpj VARCHAR(14) NOT NULL,
    razao_social VARCHAR(255) NOT NULL,
    nome_fantasia VARCHAR(255),
    logradouro VARCHAR(255),
    numero VARCHAR(20),
    complemento VARCHAR(100),
    bairro VARCHAR(100),
    cidade VARCHAR(100),
    uf CHAR(2),
    cep VARCHAR(8),
    representante_legal VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure CNPJ is unique across all tenants
    CONSTRAINT unique_cnpj UNIQUE (cnpj),
    
    -- Validate status values
    CONSTRAINT valid_company_status CHECK (status IN ('active', 'pending', 'suspended'))
);

-- Performance indices
CREATE INDEX idx_companies_tenant_id ON companies(tenant_id);
CREATE INDEX idx_companies_cnpj ON companies(cnpj);
CREATE INDEX idx_companies_status ON companies(status);
CREATE INDEX idx_companies_razao_social ON companies(razao_social);

-- Enable Row Level Security for tenant isolation
ALTER TABLE companies ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Users can only see companies in their tenant
CREATE POLICY companies_tenant_isolation ON companies
    USING (tenant_id::text = current_setting('app.current_tenant', true));

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_companies_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_companies_updated_at
    BEFORE UPDATE ON companies
    FOR EACH ROW
    EXECUTE FUNCTION update_companies_updated_at();

-- Grant permissions to vyst_app user
GRANT SELECT, INSERT, UPDATE, DELETE ON companies TO vyst_app;

COMMENT ON TABLE companies IS 'Legal entities (pessoas jurídicas) with CNPJ registration';
COMMENT ON COLUMN companies.cnpj IS '14-digit CNPJ number without formatting';
COMMENT ON COLUMN companies.razao_social IS 'Legal registered name of the company';
COMMENT ON COLUMN companies.nome_fantasia IS 'Trade name / brand name';
COMMENT ON COLUMN companies.representante_legal IS 'Name of the legal representative';
