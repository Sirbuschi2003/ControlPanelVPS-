CREATE TABLE IF NOT EXISTS managed_databases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id   UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    db_type     TEXT NOT NULL DEFAULT 'mysql',
    db_user     TEXT NOT NULL,
    db_password TEXT NOT NULL,
    charset     TEXT NOT NULL DEFAULT 'utf8mb4',
    db_collation TEXT NOT NULL DEFAULT 'utf8mb4_unicode_ci',
    size_bytes  BIGINT NOT NULL DEFAULT 0,
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_databases_name ON managed_databases(server_id, db_type, name);
CREATE INDEX IF NOT EXISTS idx_databases_server_id ON managed_databases(server_id);
