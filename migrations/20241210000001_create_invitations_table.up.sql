CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'expired', 'revoked');

CREATE TABLE invitations (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id),
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    token VARCHAR(255) NOT NULL,
    status invitation_status NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_company_id ON invitations(company_id);
CREATE INDEX idx_invitations_email_company ON invitations(email, company_id) WHERE status = 'pending';
