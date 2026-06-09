CREATE TABLE IF NOT EXISTS subdomains (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id     UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    server_id     UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    document_root TEXT NOT NULL,
    php_version   TEXT NOT NULL DEFAULT '8.2',
    enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subdomains_domain_name ON subdomains(domain_id, name);
