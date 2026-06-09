-- Central domain entity (Plesk-like subscription model)
CREATE TABLE IF NOT EXISTS domains (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id       UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    owner_user_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    document_root   TEXT NOT NULL DEFAULT '/var/www',
    php_version     TEXT NOT NULL DEFAULT '8.2',
    status          TEXT NOT NULL DEFAULT 'provisioning',
    website_id      UUID,
    dns_zone_id     UUID,
    mail_domain_id  UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_server_name ON domains(server_id, name);
CREATE INDEX IF NOT EXISTS idx_domains_owner ON domains(owner_user_id);

-- User-domain access (non-admin users see only their assigned domains)
CREATE TABLE IF NOT EXISTS domain_users (
    domain_id  UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (domain_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_domain_users_user_id ON domain_users(user_id);

-- Link existing resource tables to domains (nullable for backwards compat)
ALTER TABLE websites          ADD COLUMN IF NOT EXISTS domain_id UUID REFERENCES domains(id) ON DELETE SET NULL;
ALTER TABLE dns_zones         ADD COLUMN IF NOT EXISTS domain_id UUID REFERENCES domains(id) ON DELETE SET NULL;
ALTER TABLE mail_domains      ADD COLUMN IF NOT EXISTS domain_id UUID REFERENCES domains(id) ON DELETE SET NULL;
ALTER TABLE managed_databases ADD COLUMN IF NOT EXISTS domain_id UUID REFERENCES domains(id) ON DELETE SET NULL;
ALTER TABLE ssl_certs         ADD COLUMN IF NOT EXISTS domain_id UUID REFERENCES domains(id) ON DELETE SET NULL;
ALTER TABLE cron_jobs         ADD COLUMN IF NOT EXISTS domain_id UUID REFERENCES domains(id) ON DELETE SET NULL;
