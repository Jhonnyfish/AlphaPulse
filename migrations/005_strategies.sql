CREATE TABLE IF NOT EXISTS strategies (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(128) NOT NULL,
    description    TEXT DEFAULT '',
    type           VARCHAR(16) NOT NULL DEFAULT 'custom' CHECK (type IN ('builtin', 'custom')),
    scoring        JSONB NOT NULL DEFAULT '{}',
    dimensions     JSONB NOT NULL DEFAULT '[]',
    filters        JSONB NOT NULL DEFAULT '{}',
    max_candidates INTEGER NOT NULL DEFAULT 50,
    is_active      BOOLEAN NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_strategies_type ON strategies(type);
CREATE INDEX IF NOT EXISTS idx_strategies_is_active ON strategies(is_active);

-- Seed the built-in Alpha300 strategy
INSERT INTO strategies (id, name, description, type, scoring, dimensions, filters, max_candidates, is_active)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'Alpha300 量化排名',
    '基于Alpha300量化模型的股票排名系统',
    'builtin',
    '{"momentum": 0.3333, "trend": 0.3333, "volatility": 0.3334}',
    '["momentum", "trend", "volatility"]',
    '{"min_score": 0, "min_rank": 300}',
    50,
    true
) ON CONFLICT (id) DO NOTHING;
