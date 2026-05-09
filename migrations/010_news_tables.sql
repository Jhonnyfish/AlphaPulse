-- 010_news_tables.sql
-- 个股资讯 + 公司公告表

CREATE TABLE IF NOT EXISTS stock_news (
    id           BIGSERIAL PRIMARY KEY,
    code         VARCHAR(10) NOT NULL,
    title        TEXT NOT NULL,
    summary      TEXT,
    source       VARCHAR(100),
    url          TEXT,
    published_at TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(code, title, published_at)
);

CREATE INDEX IF NOT EXISTS idx_stock_news_code ON stock_news(code);
CREATE INDEX IF NOT EXISTS idx_stock_news_published ON stock_news(published_at DESC);

CREATE TABLE IF NOT EXISTS stock_announcements (
    id           BIGSERIAL PRIMARY KEY,
    code         VARCHAR(10) NOT NULL,
    title        TEXT NOT NULL,
    url          TEXT,
    published_at TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(code, title, published_at)
);

CREATE INDEX IF NOT EXISTS idx_stock_announcements_code ON stock_announcements(code);
CREATE INDEX IF NOT EXISTS idx_stock_announcements_published ON stock_announcements(published_at DESC);
