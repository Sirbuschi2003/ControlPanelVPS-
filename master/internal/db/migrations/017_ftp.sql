CREATE TABLE IF NOT EXISTS ftp_accounts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id   UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    server_id   UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    username    TEXT NOT NULL,
    home_dir    TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_ftp_accounts_username ON ftp_accounts(server_id, username);
