CREATE TABLE IF NOT EXISTS dns_zones (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id   UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    serial      INTEGER NOT NULL DEFAULT 1,
    refresh     INTEGER NOT NULL DEFAULT 3600,
    retry       INTEGER NOT NULL DEFAULT 900,
    expire      INTEGER NOT NULL DEFAULT 604800,
    minimum     INTEGER NOT NULL DEFAULT 300,
    nameserver  TEXT NOT NULL DEFAULT 'ns1.',
    admin_email TEXT NOT NULL DEFAULT 'admin@',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_dns_zones_name ON dns_zones(server_id, name);

CREATE TABLE IF NOT EXISTS dns_records (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zone_id    UUID NOT NULL REFERENCES dns_zones(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    content    TEXT NOT NULL,
    ttl        INTEGER NOT NULL DEFAULT 3600,
    priority   INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dns_records_zone_id ON dns_records(zone_id);
