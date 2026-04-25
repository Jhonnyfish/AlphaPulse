CREATE TABLE IF NOT EXISTS stock_notes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code       VARCHAR(8) NOT NULL,
    content    TEXT NOT NULL,
    tags       JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_stock_notes_user_code ON stock_notes(user_id, code);
CREATE INDEX IF NOT EXISTS idx_stock_notes_user_id   ON stock_notes(user_id);
