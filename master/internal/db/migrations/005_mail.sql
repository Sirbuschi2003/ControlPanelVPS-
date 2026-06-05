CREATE TABLE IF NOT EXISTS mail_domains (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id  UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    domain     TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mail_domains_domain ON mail_domains(server_id, domain);

CREATE TABLE IF NOT EXISTS mail_accounts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id  UUID NOT NULL REFERENCES mail_domains(id) ON DELETE CASCADE,
    username   TEXT NOT NULL,
    password   TEXT NOT NULL,
    quota_mb   INTEGER NOT NULL DEFAULT 1024,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mail_accounts_user ON mail_accounts(domain_id, username);

CREATE TABLE IF NOT EXISTS mail_aliases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id   UUID NOT NULL REFERENCES mail_domains(id) ON DELETE CASCADE,
    source      TEXT NOT NULL,
    destination TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
