CREATE TABLE IF NOT EXISTS websites (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id       UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    domain          TEXT NOT NULL,
    aliases         TEXT[] DEFAULT '{}',
    php_version     TEXT NOT NULL DEFAULT '8.2',
    document_root   TEXT NOT NULL DEFAULT '/var/www',
    index_file      TEXT NOT NULL DEFAULT 'index.php index.html',
    ssl_enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    ssl_force_https BOOLEAN NOT NULL DEFAULT FALSE,
    ssl_cert_id     UUID,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_websites_domain ON websites(server_id, domain);
CREATE INDEX IF NOT EXISTS idx_websites_server_id ON websites(server_id);

ALTER TABLE servers ADD COLUMN IF NOT EXISTS os_type TEXT DEFAULT 'ubuntu';
ALTER TABLE servers ADD COLUMN IF NOT EXISTS os_version TEXT;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';
ALTER TABLE servers ADD COLUMN IF NOT EXISTS notes TEXT;
