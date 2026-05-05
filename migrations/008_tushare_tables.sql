-- Tushare data tables for local market data persistence
-- All tables use tushare_ prefix to namespace from application tables

-- Stock master list
CREATE TABLE IF NOT EXISTS tushare_stock_basic (
    ts_code    VARCHAR(10) PRIMARY KEY,  -- 000001.SZ
    symbol     VARCHAR(6) NOT NULL,      -- 000001
    name       VARCHAR(20) NOT NULL,     -- 平安银行
    industry   VARCHAR(20),              -- 银行
    market     VARCHAR(10),              -- 主板
    list_date  VARCHAR(8),               -- 19910403
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Daily K-line (OHLCV)
CREATE TABLE IF NOT EXISTS tushare_daily (
    ts_code    VARCHAR(10) NOT NULL,
    trade_date VARCHAR(8) NOT NULL,      -- 20260428
    open       NUMERIC(10,2),
    high       NUMERIC(10,2),
    low        NUMERIC(10,2),
    close      NUMERIC(10,2),
    pre_close  NUMERIC(10,2),
    change     NUMERIC(10,3),
    pct_chg    NUMERIC(10,3),
    vol        NUMERIC(20,2),            -- 成交量(手)
    amount     NUMERIC(20,3),            -- 成交额(千元)
    PRIMARY KEY (ts_code, trade_date)
);

-- Daily valuation indicators
CREATE TABLE IF NOT EXISTS tushare_daily_basic (
    ts_code       VARCHAR(10) NOT NULL,
    trade_date    VARCHAR(8) NOT NULL,
    close         NUMERIC(10,2),
    turnover_rate NUMERIC(10,4),
    pe            NUMERIC(10,4),
    pe_ttm        NUMERIC(10,4),
    pb            NUMERIC(10,4),
    ps            NUMERIC(10,4),
    ps_ttm        NUMERIC(10,4),
    dv_ratio      NUMERIC(10,4),
    dv_ttm        NUMERIC(10,4),
    total_share   NUMERIC(20,4),
    float_share   NUMERIC(20,4),
    free_share    NUMERIC(20,4),
    total_mv      NUMERIC(20,4),         -- 万元
    circ_mv       NUMERIC(20,4),         -- 万元
    PRIMARY KEY (ts_code, trade_date)
);

-- Money flow
CREATE TABLE IF NOT EXISTS tushare_moneyflow (
    ts_code         VARCHAR(10) NOT NULL,
    trade_date      VARCHAR(8) NOT NULL,
    buy_sm_vol      BIGINT,
    buy_sm_amount   NUMERIC(20,2),
    sell_sm_vol     BIGINT,
    sell_sm_amount  NUMERIC(20,2),
    buy_md_vol      BIGINT,
    buy_md_amount   NUMERIC(20,2),
    sell_md_vol     BIGINT,
    sell_md_amount  NUMERIC(20,2),
    buy_lg_vol      BIGINT,
    buy_lg_amount   NUMERIC(20,2),
    sell_lg_vol     BIGINT,
    sell_lg_amount  NUMERIC(20,2),
    buy_elg_vol     BIGINT,
    buy_elg_amount  NUMERIC(20,2),
    sell_elg_vol    BIGINT,
    sell_elg_amount NUMERIC(20,2),
    net_mf_vol      BIGINT,
    net_mf_amount   NUMERIC(20,2),
    PRIMARY KEY (ts_code, trade_date)
);

-- Adjustment factor
CREATE TABLE IF NOT EXISTS tushare_adj_factor (
    ts_code    VARCHAR(10) NOT NULL,
    trade_date VARCHAR(8) NOT NULL,
    adj_factor NUMERIC(20,4),
    PRIMARY KEY (ts_code, trade_date)
);

-- Index daily行情
CREATE TABLE IF NOT EXISTS tushare_index_daily (
    ts_code    VARCHAR(10) NOT NULL,
    trade_date VARCHAR(8) NOT NULL,
    open       NUMERIC(10,2),
    high       NUMERIC(10,2),
    low        NUMERIC(10,2),
    close      NUMERIC(10,4),
    pre_close  NUMERIC(10,4),
    change     NUMERIC(10,4),
    pct_chg    NUMERIC(10,4),
    vol        NUMERIC(20,2),
    amount     NUMERIC(20,3),
    PRIMARY KEY (ts_code, trade_date)
);

-- Dragon tiger board
CREATE TABLE IF NOT EXISTS tushare_top_list (
    trade_date    VARCHAR(8) NOT NULL,
    ts_code       VARCHAR(10) NOT NULL,
    name          VARCHAR(20),
    close         NUMERIC(10,2),
    pct_change    NUMERIC(10,4),
    turnover_rate NUMERIC(10,4),
    amount        NUMERIC(20,2),
    l_sell        NUMERIC(20,2),
    l_buy         NUMERIC(20,2),
    l_amount      NUMERIC(20,2),
    net_amount    NUMERIC(20,2),
    net_rate      NUMERIC(10,4),
    amount_rate   NUMERIC(10,4),
    float_values  NUMERIC(20,2),
    reason        TEXT,
    PRIMARY KEY (trade_date, ts_code)
);

-- Margin trading (market-level)
CREATE TABLE IF NOT EXISTS tushare_margin (
    trade_date  VARCHAR(8) NOT NULL,
    exchange_id VARCHAR(10) NOT NULL,
    rzye        NUMERIC(20,2),   -- 融资余额
    rzmre       NUMERIC(20,2),   -- 融资买入额
    rzche       NUMERIC(20,2),   -- 融资偿还额
    rqye        NUMERIC(20,2),   -- 融券余额
    rqmcl       NUMERIC(20,2),   -- 融券卖出量
    rzrqye      NUMERIC(20,2),   -- 融资融券余额
    PRIMARY KEY (trade_date, exchange_id)
);

-- Trading calendar
CREATE TABLE IF NOT EXISTS tushare_trade_cal (
    exchange      VARCHAR(10) NOT NULL,
    cal_date      VARCHAR(8) NOT NULL,
    is_open       INT NOT NULL,     -- 1=trading day
    pretrade_date VARCHAR(8),
    PRIMARY KEY (exchange, cal_date)
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_tushare_daily_date ON tushare_daily(trade_date);
CREATE INDEX IF NOT EXISTS idx_tushare_daily_basic_date ON tushare_daily_basic(trade_date);
CREATE INDEX IF NOT EXISTS idx_tushare_moneyflow_date ON tushare_moneyflow(trade_date);
CREATE INDEX IF NOT EXISTS idx_tushare_adj_factor_date ON tushare_adj_factor(trade_date);
CREATE INDEX IF NOT EXISTS idx_tushare_index_daily_date ON tushare_index_daily(trade_date);
CREATE INDEX IF NOT EXISTS idx_tushare_top_list_date ON tushare_top_list(trade_date);
CREATE INDEX IF NOT EXISTS idx_tushare_margin_date ON tushare_margin(trade_date);
