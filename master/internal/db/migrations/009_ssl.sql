CREATE TABLE IF NOT EXISTS ssl_certs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id   UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    domain      TEXT NOT NULL,
    san_domains TEXT[] DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'pending',
    issuer      TEXT,
    issued_at   TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,
    auto_renew  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ssl_certs_domain ON ssl_certs(server_id, domain);

ALTER TABLE websites ADD CONSTRAINT fk_ssl_cert
    FOREIGN KEY (ssl_cert_id) REFERENCES ssl_certs(id) ON DELETE SET NULL
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE IF NOT EXISTS panel_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO panel_settings (key, value) VALUES
    ('panel_name', 'ControlPanelVPS'),
    ('panel_timezone', 'Europe/Berlin'),
    ('backup_encryption_key', encode(gen_random_bytes(32), 'hex')),
    ('smtp_host', ''),
    ('smtp_port', '587'),
    ('smtp_user', ''),
    ('smtp_pass', ''),
    ('smtp_from', ''),
    ('notify_email', '')
ON CONFLICT DO NOTHING;
