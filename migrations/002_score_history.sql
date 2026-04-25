CREATE TABLE IF NOT EXISTS score_history (
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(6)  NOT NULL,
    score      DOUBLE PRECISION NOT NULL,
    dimensions JSONB       NOT NULL DEFAULT '{}',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_score_history_code ON score_history(code);
CREATE INDEX IF NOT EXISTS idx_score_history_recorded_at ON score_history(recorded_at);
