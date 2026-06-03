-- 012: 沪深港通持股明细（hk_hold）
-- 覆盖所有个股北向持仓数据（补充 hsgt_top10 仅覆盖前十大成交股的不足）
-- Tushare API: hk_hold
-- 单位: vol=持股数量(万股), ratio=占流通A股(%)
-- 注意: 实时数据延迟约1-2个月

CREATE TABLE IF NOT EXISTS tushare_hk_hold (
    ts_code           VARCHAR(20) NOT NULL,        -- 股票代码
    trade_date        VARCHAR(8) NOT NULL,          -- 交易日期
    vol               NUMERIC(20,2) NOT NULL,       -- 持股数量(万股)
    ratio             NUMERIC(10,4),                -- 占流通A股(%)
    PRIMARY KEY (ts_code, trade_date)
);

CREATE INDEX IF NOT EXISTS idx_tushare_hk_hold_ts_code ON tushare_hk_hold(ts_code);
CREATE INDEX IF NOT EXISTS idx_tushare_hk_hold_trade_date ON tushare_hk_hold(trade_date DESC);
