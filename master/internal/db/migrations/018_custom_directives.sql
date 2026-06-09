-- Custom Nginx directives and DNS record edit support
ALTER TABLE websites ADD COLUMN IF NOT EXISTS custom_directives TEXT NOT NULL DEFAULT '';
ALTER TABLE dns_records ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
