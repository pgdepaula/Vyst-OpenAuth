CREATE TABLE IF NOT EXISTS login_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address INET NOT NULL,
    user_agent TEXT,
    login_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    latitude FLOAT,
    longitude FLOAT
);

CREATE INDEX idx_login_history_user_date ON login_history(user_id, login_at DESC);
