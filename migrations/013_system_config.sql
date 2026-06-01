-- System configuration key-value store for persisting settings like sync schedule.
CREATE TABLE IF NOT EXISTS system_config (
    key       TEXT PRIMARY KEY,
    value     TEXT NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed default Tushare sync schedule.
INSERT INTO system_config (key, value) VALUES
    ('tushare_sync_enabled',  'true'),
    ('tushare_sync_time',     '21:00'),
    ('tushare_retry_enabled', 'true'),
    ('tushare_retry_time',    '23:00')
ON CONFLICT (key) DO NOTHING;
