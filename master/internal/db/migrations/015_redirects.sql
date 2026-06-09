CREATE TABLE IF NOT EXISTS redirects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id   UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    source_path TEXT NOT NULL DEFAULT '/',
    target_url  TEXT NOT NULL,
    redirect_type INT NOT NULL DEFAULT 301,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
