CREATE TABLE IF NOT EXISTS policy_tuples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(100) NOT NULL,
    relation VARCHAR(50) NOT NULL,
    subject_type VARCHAR(50) NOT NULL,
    subject_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Unique constraint to prevent duplicate tuples
    CONSTRAINT unique_tuple UNIQUE (tenant_id, entity_type, entity_id, relation, subject_type, subject_id)
);

-- Indexes for performance (forward and reverse lookups)
CREATE INDEX idx_policy_tuples_entity ON policy_tuples(tenant_id, entity_type, entity_id);
CREATE INDEX idx_policy_tuples_subject ON policy_tuples(tenant_id, subject_type, subject_id);
CREATE INDEX idx_policy_tuples_relation ON policy_tuples(tenant_id, relation);
