CREATE TABLE IF NOT EXISTS firewall_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id   UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    rule_order  INTEGER NOT NULL DEFAULT 100,
    action      TEXT NOT NULL DEFAULT 'allow',
    direction   TEXT NOT NULL DEFAULT 'in',
    protocol    TEXT NOT NULL DEFAULT 'tcp',
    source      TEXT NOT NULL DEFAULT 'any',
    dest_port   TEXT,
    comment     TEXT,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_firewall_rules_server ON firewall_rules(server_id, rule_order);
