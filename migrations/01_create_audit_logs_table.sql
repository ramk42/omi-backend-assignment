

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    spec_version TEXT NOT NULL,
    source TEXT NOT NULL,
    type TEXT NOT NULL,
    subject TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor JSONB NOT NULL,
    action TEXT NOT NULL,
    resource JSONB NOT NULL,
    metadata JSONB,
    CHECK (char_length(spec_version) > 0)
);

CREATE INDEX IF NOT EXISTS idx_timestamp ON audit_logs (timestamp);
CREATE INDEX IF NOT EXISTS idx_type_action ON audit_logs (type, action);