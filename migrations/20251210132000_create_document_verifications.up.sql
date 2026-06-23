CREATE TABLE IF NOT EXISTS document_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document TEXT NOT NULL,
    type VARCHAR(20) NOT NULL, -- CPF, CNPJ
    source VARCHAR(50) NOT NULL, -- SERPRO, ALGORITHM, CACHE
    valid BOOLEAN NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_document_verifications_document ON document_verifications(document);
CREATE INDEX idx_document_verifications_created_at ON document_verifications(created_at);
