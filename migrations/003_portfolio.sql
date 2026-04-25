CREATE TABLE IF NOT EXISTS portfolio (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(16) NOT NULL,
    name        VARCHAR(64) DEFAULT '',
    cost_price  DOUBLE PRECISION NOT NULL CHECK (cost_price > 0),
    quantity    INTEGER NOT NULL CHECK (quantity > 0),
    notes       TEXT DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_portfolio_code ON portfolio(code);
