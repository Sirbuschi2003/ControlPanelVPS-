CREATE TABLE IF NOT EXISTS cron_jobs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id   UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    command     TEXT NOT NULL,
    schedule    TEXT NOT NULL,
    run_as_user TEXT NOT NULL DEFAULT 'www-data',
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    last_run    TIMESTAMPTZ,
    last_status TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cron_jobs_server ON cron_jobs(server_id);
