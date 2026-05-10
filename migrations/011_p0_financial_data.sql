-- Migration 011: P0 Financial Analysis Data Tables
-- tushare_financials, tushare_hsgt, tushare_hsgt_top10, tushare_margin_detail
-- Idempotent: uses CREATE TABLE IF NOT EXISTS throughout.

-- 1. Combined financial fundamentals (income + balance sheet + cashflow per report period)
CREATE TABLE IF NOT EXISTS tushare_financials (
    ts_code           VARCHAR(10) NOT NULL,
    end_date          VARCHAR(8) NOT NULL,   -- report period end date e.g. 20251231
    ann_date          VARCHAR(8),            -- announcement date
    report_type       VARCHAR(4),            -- 1=合并报表
    -- Income statement
    total_revenue     NUMERIC(20,4),         -- 营业总收入
    revenue_yoy       NUMERIC(10,4),         -- 营收同比增长率%
    total_cogs        NUMERIC(20,4),         -- 营业总成本
    operate_profit    NUMERIC(20,4),         -- 营业利润
    total_profit      NUMERIC(20,4),         -- 利润总额
    n_income          NUMERIC(20,4),         -- 净利润
    n_income_yoy      NUMERIC(10,4),         -- 净利润同比增长率%
    n_income_attr_p   NUMERIC(20,4),         -- 归母净利润
    diluted_eps       NUMERIC(10,4),         -- 每股收益(摊薄)
    -- Balance sheet
    total_assets      NUMERIC(20,4),         -- 总资产
    total_liab        NUMERIC(20,4),         -- 总负债
    total_equity      NUMERIC(20,4),         -- 归母股东权益
    monetary_cap      NUMERIC(20,4),         -- 货币资金
    accounts_receiv   NUMERIC(20,4),         -- 应收账款
    inventory         NUMERIC(20,4),         -- 存货
    -- Cashflow
    n_cashflow_act    NUMERIC(20,4),         -- 经营活动现金流净额
    n_cashflow_inv    NUMERIC(20,4),         -- 投资活动现金流净额
    n_cashflow_fnc    NUMERIC(20,4),         -- 筹资活动现金流净额
    -- Ratios (computed during sync or analysis)
    gross_margin      NUMERIC(10,4),         -- 毛利率%
    net_margin        NUMERIC(10,4),         -- 净利率%
    roe               NUMERIC(10,4),         -- ROE% (加权)
    roa               NUMERIC(10,4),         -- ROA%
    debt_ratio        NUMERIC(10,4),         -- 资产负债率%
    bps               NUMERIC(10,4),         -- 每股净资产
    PRIMARY KEY (ts_code, end_date)
);
CREATE INDEX IF NOT EXISTS idx_tushare_financials_ts ON tushare_financials(ts_code);

-- 2. 北向资金流向 (沪深港通)
CREATE TABLE IF NOT EXISTS tushare_hsgt (
    trade_date        VARCHAR(8) NOT NULL,
    ggt_ss            NUMERIC(20,4),         -- 港股通(沪)流入(百万)
    ggt_sz            NUMERIC(20,4),         -- 港股通(深)流入(百万)
    hgt               NUMERIC(20,4),         -- 沪股通流入(百万)
    sgt               NUMERIC(20,4),         -- 深股通流入(百万)
    north_money       NUMERIC(20,4),         -- 北向资金合计(百万)
    south_money       NUMERIC(20,4),         -- 南向资金合计(百万)
    PRIMARY KEY (trade_date)
);

-- 3. 北向资金十大成交股
CREATE TABLE IF NOT EXISTS tushare_hsgt_top10 (
    trade_date    VARCHAR(8) NOT NULL,
    ts_code       VARCHAR(10) NOT NULL,
    name          VARCHAR(20),
    close         NUMERIC(10,2),
    pct_change    NUMERIC(10,4),
    market_type   VARCHAR(4),              -- 1=沪股通 3=深股通
    rank          INT,
    amount        NUMERIC(20,4),           -- 成交额(万)
    net_amount    NUMERIC(20,4),           -- 净买入额(万)
    buy_amount    NUMERIC(20,4),           -- 买入额(万)
    sell_amount   NUMERIC(20,4),           -- 卖出额(万)
    PRIMARY KEY (trade_date, ts_code, market_type)
);

-- 4. 个股融资融券明细
CREATE TABLE IF NOT EXISTS tushare_margin_detail (
    trade_date    VARCHAR(8) NOT NULL,
    ts_code       VARCHAR(10) NOT NULL,
    rzye          NUMERIC(20,4),           -- 融资余额
    rzmre         NUMERIC(20,4),           -- 融资买入额
    rzche         NUMERIC(20,4),           -- 融资偿还额
    rqye          NUMERIC(20,4),           -- 融券余额
    rqmcl         NUMERIC(20,4),           -- 融券卖出量
    rqchl         NUMERIC(20,4),           -- 融券偿还量
    rzrqye        NUMERIC(20,4),           -- 融资融券余额
    PRIMARY KEY (trade_date, ts_code)
);
CREATE INDEX IF NOT EXISTS idx_tushare_margin_detail_ts ON tushare_margin_detail(ts_code);
