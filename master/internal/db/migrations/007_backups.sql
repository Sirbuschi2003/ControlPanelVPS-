CREATE TABLE IF NOT EXISTS backup_configs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id      UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    storage_type   TEXT NOT NULL DEFAULT 'local',
    schedule       TEXT NOT NULL DEFAULT '0 2 * * *',
    retention_days INTEGER NOT NULL DEFAULT 7,
    include_paths  TEXT[] DEFAULT ARRAY['/etc', '/var/www', '/var/lib/mysql'],
    storage_config JSONB NOT NULL DEFAULT '{}',
    encrypt        BOOLEAN NOT NULL DEFAULT TRUE,
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS backup_jobs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id      UUID NOT NULL REFERENCES backup_configs(id) ON DELETE CASCADE,
    status         TEXT NOT NULL DEFAULT 'running',
    size_bytes     BIGINT NOT NULL DEFAULT 0,
    file_path      TEXT,
    error_message  TEXT,
    started_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_backup_jobs_config ON backup_jobs(config_id, started_at DESC);
