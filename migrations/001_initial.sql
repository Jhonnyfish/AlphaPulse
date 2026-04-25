CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(128) NOT NULL,
    role          VARCHAR(16) NOT NULL DEFAULT 'user',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS invite_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(32) UNIQUE NOT NULL,
    created_by  UUID REFERENCES users(id),
    max_uses    INT NOT NULL DEFAULT 1,
    uses        INT NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS watchlist (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(16) NOT NULL,
    name       VARCHAR(64),
    group_name VARCHAR(64) DEFAULT 'default',
    added_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(128) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS score_history (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(6) NOT NULL,
    score       DOUBLE PRECISION NOT NULL,
    dimensions  JSONB NOT NULL DEFAULT '{}',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_score_history_code ON score_history(code);
CREATE INDEX IF NOT EXISTS idx_score_history_recorded_at ON score_history(recorded_at);
