CREATE TABLE IF NOT EXISTS trading_journal (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code           VARCHAR(16) NOT NULL,
    name           VARCHAR(64) DEFAULT '',
    type           VARCHAR(8) NOT NULL CHECK (type IN ('buy','sell')),
    price          DOUBLE PRECISION NOT NULL CHECK (price > 0),
    quantity       INTEGER NOT NULL CHECK (quantity > 0),
    fees           DOUBLE PRECISION DEFAULT 0,
    date           DATE NOT NULL,
    notes          TEXT DEFAULT '',
    strategy_label VARCHAR(64) DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trading_journal_code ON trading_journal(code);
CREATE INDEX IF NOT EXISTS idx_trading_journal_date ON trading_journal(date);
CREATE INDEX IF NOT EXISTS idx_trading_journal_type ON trading_journal(type);
