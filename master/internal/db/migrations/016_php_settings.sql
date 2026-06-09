CREATE TABLE IF NOT EXISTS php_settings (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id         UUID UNIQUE NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    memory_limit      INT NOT NULL DEFAULT 256,
    max_execution_time INT NOT NULL DEFAULT 60,
    upload_max_filesize INT NOT NULL DEFAULT 64,
    post_max_size     INT NOT NULL DEFAULT 64,
    max_input_vars    INT NOT NULL DEFAULT 1000,
    display_errors    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
