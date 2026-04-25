CREATE TABLE IF NOT EXISTS custom_alerts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(16) NOT NULL,
    name        VARCHAR(64) DEFAULT '',
    type        VARCHAR(32) NOT NULL CHECK (type IN ('price_above', 'price_below', 'change_above', 'change_below')),
    threshold   DOUBLE PRECISION NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    triggered   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_custom_alerts_code ON custom_alerts(code);
CREATE INDEX IF NOT EXISTS idx_custom_alerts_enabled ON custom_alerts(enabled);
