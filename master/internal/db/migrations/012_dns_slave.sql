-- DNS master/slave support
ALTER TABLE dns_zones ADD COLUMN IF NOT EXISTS zone_type  TEXT NOT NULL DEFAULT 'master';
ALTER TABLE dns_zones ADD COLUMN IF NOT EXISTS master_ip  TEXT;
