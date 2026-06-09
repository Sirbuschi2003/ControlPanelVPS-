INSERT INTO panel_settings (key, value)
VALUES ('auto_update', 'false')
ON CONFLICT DO NOTHING;
