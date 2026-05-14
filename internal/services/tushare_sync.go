package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// TushareSync handles syncing Tushare data to local PostgreSQL.
type TushareSync struct {
	ts        *TushareService
	eastMoney *EastMoneyService
	db        *pgxpool.Pool
	logger    *zap.Logger
}

// NewTushareSync creates a new TushareSync instance.
func NewTushareSync(ts *TushareService, eastMoney *EastMoneyService, db *pgxpool.Pool, logger *zap.Logger) *TushareSync {
	return &TushareSync{
		ts:        ts,
		eastMoney: eastMoney,
		db:        db,
		logger:    logger,
	}
}

// RunDaily executes the full daily sync pipeline.
func (s *TushareSync) RunDaily(ctx context.Context) {
	start := time.Now()
	log.Printf("[tushare-sync] starting daily sync...")

	if err := s.SyncStockBasic(ctx); err != nil {
		log.Printf("[tushare-sync] stock_basic sync failed: %v", err)
	}

	// Sync trade calendar for next 30 days
	if err := s.SyncTradeCal(ctx); err != nil {
		log.Printf("[tushare-sync] trade_cal sync failed: %v", err)
	}

	// Backfill: find recent trading days missing from DB and sync them
	missingDates := s.getMissingTradeDates(ctx)
	if len(missingDates) > 0 {
		log.Printf("[tushare-sync] found %d missing trade date(s): %v", len(missingDates), missingDates)
	}
	for _, date := range missingDates {
		s.syncAllForDate(ctx, date)
	}

	// Today's date
	tradeDate := time.Now().Format("20060102")

	// Check if today is a trading day
	isOpen, err := s.isTradeDay(ctx, tradeDate)
	if err != nil {
		log.Printf("[tushare-sync] failed to check trade calendar: %v", err)
	}
	if !isOpen {
		log.Printf("[tushare-sync] %s is not a trading day, skipping today's sync", tradeDate)
	} else {
		s.syncAllForDate(ctx, tradeDate)
	}

	// Sync news and announcements for watchlist stocks
	if err := s.SyncNews(ctx); err != nil {
		log.Printf("[tushare-sync] news sync failed: %v", err)
	}

	if err := s.SyncAnnouncements(ctx); err != nil {
		log.Printf("[tushare-sync] announcements sync failed: %v", err)
	}

	log.Printf("[tushare-sync] daily sync completed in %v", time.Since(start))
}

// syncAllForDate syncs all data tables for a specific trade date.
func (s *TushareSync) syncAllForDate(ctx context.Context, tradeDate string) {
	if err := s.SyncDaily(ctx, tradeDate); err != nil {
		log.Printf("[tushare-sync] daily sync failed for %s: %v", tradeDate, err)
	}

	if err := s.SyncDailyBasic(ctx, tradeDate); err != nil {
		log.Printf("[tushare-sync] daily_basic sync failed for %s: %v", tradeDate, err)
	}

	if err := s.SyncMoneyFlow(ctx, tradeDate); err != nil {
		log.Printf("[tushare-sync] moneyflow sync failed for %s: %v", tradeDate, err)
	}

	if err := s.SyncAdjFactor(ctx, tradeDate); err != nil {
		log.Printf("[tushare-sync] adj_factor sync failed for %s: %v", tradeDate, err)
	}

	if err := s.SyncIndexDaily(ctx, tradeDate); err != nil {
		log.Printf("[tushare-sync] index_daily sync failed for %s: %v", tradeDate, err)
	}

	if err := s.SyncTopList(ctx, tradeDate); err != nil {
		log.Printf("[tushare-sync] top_list sync failed for %s: %v", tradeDate, err)
	}

	if err := s.SyncMargin(ctx, tradeDate); err != nil {
		log.Printf("[tushare-sync] margin sync failed for %s: %v", tradeDate, err)
	}

	if err := s.SyncHsgt(ctx, tradeDate, tradeDate); err != nil {
		log.Printf("[tushare-sync] hsgt sync failed for %s: %v", tradeDate, err)
	}

	if err := s.SyncHsgtTop10(ctx, tradeDate); err != nil {
		log.Printf("[tushare-sync] hsgt_top10 sync failed for %s: %v", tradeDate, err)
	}

	if err := s.SyncMarginDetail(ctx, tradeDate); err != nil {
		log.Printf("[tushare-sync] margin_detail sync failed for %s: %v", tradeDate, err)
	}
}

// isTradeDay checks if a given date is a trading day according to the trade calendar.
func (s *TushareSync) isTradeDay(ctx context.Context, date string) (bool, error) {
	var isOpen int
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE((SELECT is_open FROM tushare_trade_cal WHERE cal_date = $1 AND exchange = 'SSE' LIMIT 1), 0)`,
		date).Scan(&isOpen)
	if err != nil {
		return false, err
	}
	return isOpen == 1, nil
}

// getMissingTradeDates finds recent trading days (within last 14 days) that have
// zero rows in tushare_daily. Returns dates in ascending order (oldest first).
func (s *TushareSync) getMissingTradeDates(ctx context.Context) []string {
	rows, err := s.db.Query(ctx, `
		SELECT cal_date FROM tushare_trade_cal
		WHERE exchange = 'SSE'
		  AND is_open = 1
		  AND cal_date < to_char(now(), 'YYYYMMDD')
		  AND cal_date >= to_char(now() - interval '14 days', 'YYYYMMDD')
		  AND cal_date NOT IN (
		    SELECT DISTINCT trade_date FROM tushare_daily
		    WHERE trade_date >= to_char(now() - interval '14 days', 'YYYYMMDD')
		  )
		ORDER BY cal_date ASC
	`)
	if err != nil {
		log.Printf("[tushare-sync] failed to query missing dates: %v", err)
		return nil
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if rows.Scan(&d) == nil {
			dates = append(dates, d)
		}
	}
	return dates
}

// GetWatchlistCodes returns all stock codes in the watchlist table.
func (s *TushareSync) GetWatchlistCodes(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT code FROM watchlist ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			continue
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

// SyncStockBasic syncs the stock master list from Tushare.
func (s *TushareSync) SyncStockBasic(ctx context.Context) error {
	start := time.Now()
	stocks, err := s.ts.FetchStockBasic(ctx)
	if err != nil {
		return fmt.Errorf("fetch stock_basic: %w", err)
	}

	batch := &strings.Builder{}
	for i, stock := range stocks {
		if i > 0 {
			batch.WriteString(",")
		}
		name := strings.ReplaceAll(stock.Name, "'", "''")
		industry := strings.ReplaceAll(stock.Industry, "'", "''")
		batch.WriteString(fmt.Sprintf("('%s','%s','%s','%s','%s','%s')",
			stock.TsCode, stock.Symbol, name, industry, stock.Market, stock.ListDate))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_stock_basic (ts_code, symbol, name, industry, market, list_date)
		VALUES %s
		ON CONFLICT (ts_code) DO UPDATE SET
			name = EXCLUDED.name,
			industry = EXCLUDED.industry,
			market = EXCLUDED.market,
			updated_at = NOW()
	`, batch.String())

	_, err = s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("upsert stock_basic: %w", err)
	}

	log.Printf("[tushare-sync] synced %d stocks in %v", len(stocks), time.Since(start))
	return nil
}

// SyncDaily syncs daily K-line data for a specific trade date.
func (s *TushareSync) SyncDaily(ctx context.Context, tradeDate string) error {
	start := time.Now()
	rows, err := s.ts.FetchDaily(ctx, "", "", tradeDate)
	if err != nil {
		return fmt.Errorf("fetch daily for %s: %w", tradeDate, err)
	}

	if len(rows) == 0 {
		log.Printf("[tushare-sync] no daily data for %s (possibly non-trading day)", tradeDate)
		return nil
	}

	// Verify that returned data actually matches the requested trade date.
	// Tushare API may return stale data if today's data isn't published yet.
	actualDate := rows[0].TradeDate
	if actualDate != tradeDate {
		log.Printf("[tushare-sync] WARNING: requested daily for %s but Tushare returned data for %s (data not yet published?), skipping insert",
			tradeDate, actualDate)
		return nil
	}

	batch := &strings.Builder{}
	for i, r := range rows {
		if i > 0 {
			batch.WriteString(",")
		}
		batch.WriteString(fmt.Sprintf("('%s','%s',%s,%s,%s,%s,%s,%s,%s,%s,%s)",
			r.TsCode, r.TradeDate,
			floatStr(r.Open), floatStr(r.High), floatStr(r.Low), floatStr(r.Close),
			floatStr(r.PreClose), floatStr(r.Change), floatStr(r.PctChg),
			floatStr(r.Vol), floatStr(r.Amount)))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_daily (ts_code, trade_date, open, high, low, close, pre_close, change, pct_chg, vol, amount)
		VALUES %s
		ON CONFLICT (ts_code, trade_date) DO NOTHING
	`, batch.String())

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert daily: %w", err)
	}

	log.Printf("[tushare-sync] synced %d daily rows for %s, inserted %d in %v",
		len(rows), tradeDate, tag.RowsAffected(), time.Since(start))
	return nil
}

// SyncDailyBasic syncs daily valuation indicators for a specific trade date.
func (s *TushareSync) SyncDailyBasic(ctx context.Context, tradeDate string) error {
	start := time.Now()
	rows, err := s.ts.FetchDailyBasic(ctx, "", tradeDate)
	if err != nil {
		return fmt.Errorf("fetch daily_basic for %s: %w", tradeDate, err)
	}

	if len(rows) == 0 {
		log.Printf("[tushare-sync] no daily_basic data for %s", tradeDate)
		return nil
	}

	// Verify data date matches request
	if rows[0].TradeDate != tradeDate {
		log.Printf("[tushare-sync] WARNING: requested daily_basic for %s but got %s, skipping",
			tradeDate, rows[0].TradeDate)
		return nil
	}

	batch := &strings.Builder{}
	for i, r := range rows {
		if i > 0 {
			batch.WriteString(",")
		}
		batch.WriteString(fmt.Sprintf("('%s','%s',%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)",
			r.TsCode, r.TradeDate,
			floatStr(r.Close), floatStr(r.TurnoverRate),
			floatStr(r.PE), floatStr(r.PE_TTM), floatStr(r.PB),
			floatStr(r.PS), floatStr(r.PS_TTM),
			floatStr(r.DVRatio), floatStr(r.DV_TTM),
			floatStr(r.TotalShare), floatStr(r.FloatShare), floatStr(r.FreeShare),
			floatStr(r.TotalMV), floatStr(r.CircMV)))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_daily_basic (ts_code, trade_date, close, turnover_rate,
			pe, pe_ttm, pb, ps, ps_ttm, dv_ratio, dv_ttm,
			total_share, float_share, free_share, total_mv, circ_mv)
		VALUES %s
		ON CONFLICT (ts_code, trade_date) DO NOTHING
	`, batch.String())

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert daily_basic: %w", err)
	}

	log.Printf("[tushare-sync] synced %d daily_basic rows for %s, inserted %d in %v",
		len(rows), tradeDate, tag.RowsAffected(), time.Since(start))
	return nil
}

// floatStr formats a float64 for SQL insertion, returning "NULL" for zero.
func floatStr(f float64) string {
	if f == 0 {
		return "NULL"
	}
	return fmt.Sprintf("%.4f", f)
}

// int64Str formats an int64 for SQL insertion.
func int64Str(v int64) string {
	if v == 0 {
		return "NULL"
	}
	return fmt.Sprintf("%d", v)
}

// ==================== Extended Sync Methods ====================

// SyncTradeCal syncs the trading calendar for the next 30 days.
func (s *TushareSync) SyncTradeCal(ctx context.Context) error {
	today := time.Now()
	start := today.Format("20060102")
	end := today.Add(30 * 24 * time.Hour).Format("20060102")

	rows, err := s.ts.FetchTradeCal(ctx, "SSE", start, end)
	if err != nil {
		return fmt.Errorf("fetch trade_cal: %w", err)
	}

	if len(rows) == 0 {
		return nil
	}

	batch := &strings.Builder{}
	for i, r := range rows {
		if i > 0 {
			batch.WriteString(",")
		}
		batch.WriteString(fmt.Sprintf("('%s','%s',%d,'%s')",
			r.Exchange, r.CalDate, r.IsOpen, r.PretradeDate))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_trade_cal (exchange, cal_date, is_open, pretrade_date)
		VALUES %s
		ON CONFLICT (exchange, cal_date) DO UPDATE SET
			is_open = EXCLUDED.is_open,
			pretrade_date = EXCLUDED.pretrade_date
	`, batch.String())

	_, err = s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("upsert trade_cal: %w", err)
	}

	log.Printf("[tushare-sync] synced %d trade_cal rows", len(rows))
	return nil
}

// SyncMoneyFlow syncs money flow data for a specific trade date.
func (s *TushareSync) SyncMoneyFlow(ctx context.Context, tradeDate string) error {
	start := time.Now()
	rows, err := s.ts.FetchMoneyFlow(ctx, "", "", tradeDate)
	if err != nil {
		return fmt.Errorf("fetch moneyflow for %s: %w", tradeDate, err)
	}

	if len(rows) == 0 {
		log.Printf("[tushare-sync] no moneyflow data for %s", tradeDate)
		return nil
	}

	// Verify data date matches request
	if rows[0].TradeDate != tradeDate {
		log.Printf("[tushare-sync] WARNING: requested moneyflow for %s but got %s, skipping",
			tradeDate, rows[0].TradeDate)
		return nil
	}

	batch := &strings.Builder{}
	for i, r := range rows {
		if i > 0 {
			batch.WriteString(",")
		}
		batch.WriteString(fmt.Sprintf("('%s','%s',%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)",
			r.TsCode, r.TradeDate,
			int64Str(r.BuySmVol), floatStr(r.BuySmAmount), int64Str(r.SellSmVol), floatStr(r.SellSmAmount),
			int64Str(r.BuyMdVol), floatStr(r.BuyMdAmount), int64Str(r.SellMdVol), floatStr(r.SellMdAmount),
			int64Str(r.BuyLgVol), floatStr(r.BuyLgAmount), int64Str(r.SellLgVol), floatStr(r.SellLgAmount),
			int64Str(r.BuyElgVol), floatStr(r.BuyElgAmount), int64Str(r.SellElgVol), floatStr(r.SellElgAmount),
			int64Str(r.NetMfVol), floatStr(r.NetMfAmount)))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_moneyflow (ts_code, trade_date,
			buy_sm_vol, buy_sm_amount, sell_sm_vol, sell_sm_amount,
			buy_md_vol, buy_md_amount, sell_md_vol, sell_md_amount,
			buy_lg_vol, buy_lg_amount, sell_lg_vol, sell_lg_amount,
			buy_elg_vol, buy_elg_amount, sell_elg_vol, sell_elg_amount,
			net_mf_vol, net_mf_amount)
		VALUES %s
		ON CONFLICT (ts_code, trade_date) DO NOTHING
	`, batch.String())

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert moneyflow: %w", err)
	}

	log.Printf("[tushare-sync] synced %d moneyflow rows for %s, inserted %d in %v",
		len(rows), tradeDate, tag.RowsAffected(), time.Since(start))
	return nil
}

// SyncAdjFactor syncs adjustment factor data for a specific trade date.
func (s *TushareSync) SyncAdjFactor(ctx context.Context, tradeDate string) error {
	start := time.Now()
	rows, err := s.ts.FetchAdjFactor(ctx, "", "", tradeDate)
	if err != nil {
		return fmt.Errorf("fetch adj_factor for %s: %w", tradeDate, err)
	}

	if len(rows) == 0 {
		log.Printf("[tushare-sync] no adj_factor data for %s", tradeDate)
		return nil
	}

	batch := &strings.Builder{}
	for i, r := range rows {
		if i > 0 {
			batch.WriteString(",")
		}
		batch.WriteString(fmt.Sprintf("('%s','%s',%s)",
			r.TsCode, r.TradeDate, floatStr(r.AdjFactor)))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_adj_factor (ts_code, trade_date, adj_factor)
		VALUES %s
		ON CONFLICT (ts_code, trade_date) DO NOTHING
	`, batch.String())

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert adj_factor: %w", err)
	}

	log.Printf("[tushare-sync] synced %d adj_factor rows for %s, inserted %d in %v",
		len(rows), tradeDate, tag.RowsAffected(), time.Since(start))
	return nil
}

// SyncIndexDaily syncs index daily data for major indices.
func (s *TushareSync) SyncIndexDaily(ctx context.Context, tradeDate string) error {
	start := time.Now()
	indices := []string{"000001.SH", "399001.SZ", "399006.SZ"}

	for _, idx := range indices {
		rows, err := s.ts.FetchIndexDaily(ctx, idx, "", tradeDate)
		if err != nil {
			log.Printf("[tushare-sync] fetch index_daily %s failed: %v", idx, err)
			continue
		}

		if len(rows) == 0 {
			continue
		}

		batch := &strings.Builder{}
		for i, r := range rows {
			if i > 0 {
				batch.WriteString(",")
			}
			batch.WriteString(fmt.Sprintf("('%s','%s',%s,%s,%s,%s,%s,%s,%s,%s,%s)",
				r.TsCode, r.TradeDate,
				floatStr(r.Open), floatStr(r.High), floatStr(r.Low), floatStr(r.Close),
				floatStr(r.PreClose), floatStr(r.Change), floatStr(r.PctChg),
				floatStr(r.Vol), floatStr(r.Amount)))
		}

		query := fmt.Sprintf(`
			INSERT INTO tushare_index_daily (ts_code, trade_date, open, high, low, close, pre_close, change, pct_chg, vol, amount)
			VALUES %s
			ON CONFLICT (ts_code, trade_date) DO NOTHING
		`, batch.String())

		s.db.Exec(ctx, query)
	}

	log.Printf("[tushare-sync] synced index_daily for %d indices in %v", len(indices), time.Since(start))
	return nil
}

// SyncTopList syncs dragon tiger board data for a specific trade date.
func (s *TushareSync) SyncTopList(ctx context.Context, tradeDate string) error {
	start := time.Now()
	rows, err := s.ts.FetchTopList(ctx, tradeDate)
	if err != nil {
		return fmt.Errorf("fetch top_list for %s: %w", tradeDate, err)
	}

	if len(rows) == 0 {
		log.Printf("[tushare-sync] no top_list data for %s", tradeDate)
		return nil
	}

	batch := &strings.Builder{}
	for i, r := range rows {
		if i > 0 {
			batch.WriteString(",")
		}
		name := strings.ReplaceAll(r.Name, "'", "''")
		reason := strings.ReplaceAll(r.Reason, "'", "''")
		batch.WriteString(fmt.Sprintf("('%s','%s','%s',%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,'%s')",
			r.TradeDate, r.TsCode, name,
			floatStr(r.Close), floatStr(r.PctChange), floatStr(r.TurnoverRate),
			floatStr(r.Amount), floatStr(r.LSell), floatStr(r.LBuy), floatStr(r.LAmount),
			floatStr(r.NetAmount), floatStr(r.NetRate), floatStr(r.AmountRate),
			floatStr(r.FloatValues), reason))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_top_list (trade_date, ts_code, name, close, pct_change, turnover_rate,
			amount, l_sell, l_buy, l_amount, net_amount, net_rate, amount_rate, float_values, reason)
		VALUES %s
		ON CONFLICT (trade_date, ts_code) DO NOTHING
	`, batch.String())

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert top_list: %w", err)
	}

	log.Printf("[tushare-sync] synced %d top_list rows for %s, inserted %d in %v",
		len(rows), tradeDate, tag.RowsAffected(), time.Since(start))
	return nil
}

// SyncMargin syncs margin trading data for a specific trade date.
func (s *TushareSync) SyncMargin(ctx context.Context, tradeDate string) error {
	start := time.Now()
	rows, err := s.ts.FetchMargin(ctx, tradeDate, tradeDate)
	if err != nil {
		return fmt.Errorf("fetch margin for %s: %w", tradeDate, err)
	}

	if len(rows) == 0 {
		log.Printf("[tushare-sync] no margin data for %s", tradeDate)
		return nil
	}

	batch := &strings.Builder{}
	for i, r := range rows {
		if i > 0 {
			batch.WriteString(",")
		}
		batch.WriteString(fmt.Sprintf("('%s','%s',%s,%s,%s,%s,%s,%s)",
			r.TradeDate, r.ExchangeID,
			floatStr(r.Rzye), floatStr(r.Rzmre), floatStr(r.Rzche),
			floatStr(r.Rqye), floatStr(r.Rqmcl), floatStr(r.Rzrqye)))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_margin (trade_date, exchange_id, rzye, rzmre, rzche, rqye, rqmcl, rzrqye)
		VALUES %s
		ON CONFLICT (trade_date, exchange_id) DO NOTHING
	`, batch.String())

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert margin: %w", err)
	}

	log.Printf("[tushare-sync] synced %d margin rows for %s, inserted %d in %v",
		len(rows), tradeDate, tag.RowsAffected(), time.Since(start))
	return nil
}

// SyncFinancials syncs financial fundamentals for watchlist stocks.
func (s *TushareSync) SyncFinancials(ctx context.Context, startDate, endDate string) error {
	start := time.Now()
	log.Printf("[tushare-sync] starting financials sync for %s to %s...", startDate, endDate)

	codes, err := s.GetWatchlistCodes(ctx)
	if err != nil {
		return fmt.Errorf("get watchlist codes: %w", err)
	}
	if len(codes) == 0 {
		log.Printf("[tushare-sync] no watchlist stocks to sync financials for")
		return nil
	}

	totalInserted := 0
	for _, code := range codes {
		tsCode := ToTsCode(code)
		rows, err := s.ts.FetchFinaIndicator(ctx, tsCode, startDate, endDate)
		if err != nil {
			log.Printf("[tushare-sync] fetch fina_indicator for %s failed: %v", code, err)
			continue
		}
		if len(rows) == 0 {
			continue
		}

		batch := &strings.Builder{}
		for i, r := range rows {
			if i > 0 {
				batch.WriteString(",")
			}
			annDate := "NULL"
			if r.AnnDate != "" {
				annDate = "'" + r.AnnDate + "'"
			}
			batch.WriteString(fmt.Sprintf("('%s','%s',%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)",
				r.TsCode, r.EndDate, annDate,
				floatStr(r.ROE), floatStr(r.ROA), floatStr(r.GrossMargin), floatStr(r.NetMargin),
				floatStr(r.EPS), floatStr(r.BPS), floatStr(r.DebtToAssets),
				floatStr(r.RevenueYoY), floatStr(r.NetProfitYoY)))
		}

		query := fmt.Sprintf(`
			INSERT INTO tushare_financials (ts_code, end_date, ann_date,
				roe, roa, gross_margin, net_margin, diluted_eps, bps, debt_ratio,
				revenue_yoy, n_income_yoy)
			VALUES %s
			ON CONFLICT (ts_code, end_date) DO NOTHING
		`, batch.String())

		tag, err := s.db.Exec(ctx, query)
		if err != nil {
			log.Printf("[tushare-sync] insert financials for %s failed: %v", code, err)
			continue
		}
		totalInserted += int(tag.RowsAffected())

		// Respect rate limits
		time.Sleep(200 * time.Millisecond)
	}

	log.Printf("[tushare-sync] financials sync completed: %d stocks, %d rows inserted in %v",
		len(codes), totalInserted, time.Since(start))
	return nil
}

// SyncHsgt syncs northbound money flow data (沪深港通).
func (s *TushareSync) SyncHsgt(ctx context.Context, startDate, endDate string) error {
	start := time.Now()
	rows, err := s.ts.FetchHsgt(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("fetch hsgt for %s-%s: %w", startDate, endDate, err)
	}

	if len(rows) == 0 {
		log.Printf("[tushare-sync] no hsgt data for %s to %s", startDate, endDate)
		return nil
	}

	batch := &strings.Builder{}
	for i, r := range rows {
		if i > 0 {
			batch.WriteString(",")
		}
		batch.WriteString(fmt.Sprintf("('%s',%s,%s,%s,%s,%s,%s)",
			r.TradeDate,
			floatStr(r.GgtSS), floatStr(r.GgtSZ),
			floatStr(r.Hgt), floatStr(r.Sgt),
			floatStr(r.NorthMoney), floatStr(r.SouthMoney)))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_hsgt (trade_date, ggt_ss, ggt_sz, hgt, sgt, north_money, south_money)
		VALUES %s
		ON CONFLICT (trade_date) DO NOTHING
	`, batch.String())

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert hsgt: %w", err)
	}

	log.Printf("[tushare-sync] synced %d hsgt rows for %s to %s, inserted %d in %v",
		len(rows), startDate, endDate, tag.RowsAffected(), time.Since(start))
	return nil
}

// SyncHsgtTop10 syncs top 10 northbound trading stocks for a specific trade date.
func (s *TushareSync) SyncHsgtTop10(ctx context.Context, tradeDate string) error {
	start := time.Now()
	rows, err := s.ts.FetchHsgtTop10(ctx, tradeDate, "", 0)
	if err != nil {
		return fmt.Errorf("fetch hsgt_top10 for %s: %w", tradeDate, err)
	}

	if len(rows) == 0 {
		log.Printf("[tushare-sync] no hsgt_top10 data for %s", tradeDate)
		return nil
	}

	batch := &strings.Builder{}
	for i, r := range rows {
		if i > 0 {
			batch.WriteString(",")
		}
		name := strings.ReplaceAll(r.Name, "'", "''")
		batch.WriteString(fmt.Sprintf("('%s','%s','%s',%s,%s,%s,%s,%s,%s,%s,%s)",
			r.TradeDate, r.TsCode, name,
			floatStr(r.Close), floatStr(r.PctChange),
			int64Str(int64(r.MarketType)),
			int64Str(int64(r.Rank)),
			floatStr(r.Amount), floatStr(r.NetAmount),
			floatStr(r.BuyAmount), floatStr(r.SellAmount)))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_hsgt_top10 (trade_date, ts_code, name, close, pct_change, market_type, rank, amount, net_amount, buy_amount, sell_amount)
		VALUES %s
		ON CONFLICT (trade_date, ts_code, market_type) DO UPDATE
		SET amount = EXCLUDED.amount, net_amount = EXCLUDED.net_amount,
		    buy_amount = EXCLUDED.buy_amount, sell_amount = EXCLUDED.sell_amount
	`, batch.String())

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert hsgt_top10: %w", err)
	}

	log.Printf("[tushare-sync] synced %d hsgt_top10 rows for %s, inserted %d in %v",
		len(rows), tradeDate, tag.RowsAffected(), time.Since(start))
	return nil
}

// SyncMarginDetail syncs margin trading detail data by stock for a specific trade date.
func (s *TushareSync) SyncMarginDetail(ctx context.Context, tradeDate string) error {
	start := time.Now()
	rows, err := s.ts.FetchMarginDetail(ctx, tradeDate, "")
	if err != nil {
		return fmt.Errorf("fetch margin_detail for %s: %w", tradeDate, err)
	}

	if len(rows) == 0 {
		log.Printf("[tushare-sync] no margin_detail data for %s", tradeDate)
		return nil
	}

	batch := &strings.Builder{}
	for i, r := range rows {
		if i > 0 {
			batch.WriteString(",")
		}
		batch.WriteString(fmt.Sprintf("('%s','%s',%s,%s,%s,%s,%s,%s,%s)",
			r.TradeDate, r.TsCode,
			floatStr(r.Rzye), floatStr(r.Rzmre), floatStr(r.Rzche),
			floatStr(r.Rqye), floatStr(r.Rqmcl), floatStr(r.Rqchl),
			floatStr(r.Rzrqye)))
	}

	query := fmt.Sprintf(`
		INSERT INTO tushare_margin_detail (trade_date, ts_code, rzye, rzmre, rzche, rqye, rqmcl, rqchl, rzrqye)
		VALUES %s
		ON CONFLICT (trade_date, ts_code) DO NOTHING
	`, batch.String())

	tag, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("insert margin_detail: %w", err)
	}

	log.Printf("[tushare-sync] synced %d margin_detail rows for %s, inserted %d in %v",
		len(rows), tradeDate, tag.RowsAffected(), time.Since(start))
	return nil
}

// RunBackfill populates historical data for a date range.
func (s *TushareSync) RunBackfill(ctx context.Context, startDate, endDate string) error {
	log.Printf("[tushare-backfill] starting backfill from %s to %s", startDate, endDate)

	// Parse dates
	start, err := time.Parse("20060102", startDate)
	if err != nil {
		return fmt.Errorf("parse start_date: %w", err)
	}
	end, err := time.Parse("20060102", endDate)
	if err != nil {
		return fmt.Errorf("parse end_date: %w", err)
	}

	// First sync stock basic
	if err := s.SyncStockBasic(ctx); err != nil {
		log.Printf("[tushare-backfill] stock_basic sync failed: %v", err)
	}

	// Iterate through dates
	for d := start; !d.After(end); d = d.Add(24 * time.Hour) {
		// Skip weekends
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}

		tradeDate := d.Format("20060102")
		log.Printf("[tushare-backfill] syncing %s...", tradeDate)

		if err := s.SyncDaily(ctx, tradeDate); err != nil {
			log.Printf("[tushare-backfill] daily failed for %s: %v", tradeDate, err)
		}
		if err := s.SyncDailyBasic(ctx, tradeDate); err != nil {
			log.Printf("[tushare-backfill] daily_basic failed for %s: %v", tradeDate, err)
		}
		if err := s.SyncMoneyFlow(ctx, tradeDate); err != nil {
			log.Printf("[tushare-backfill] moneyflow failed for %s: %v", tradeDate, err)
		}
		if err := s.SyncAdjFactor(ctx, tradeDate); err != nil {
			log.Printf("[tushare-backfill] adj_factor failed for %s: %v", tradeDate, err)
		}
		if err := s.SyncIndexDaily(ctx, tradeDate); err != nil {
			log.Printf("[tushare-backfill] index_daily failed for %s: %v", tradeDate, err)
		}

		if err := s.SyncHsgt(ctx, tradeDate, tradeDate); err != nil {
			log.Printf("[tushare-backfill] hsgt failed for %s: %v", tradeDate, err)
		}
		if err := s.SyncMarginDetail(ctx, tradeDate); err != nil {
			log.Printf("[tushare-backfill] margin_detail failed for %s: %v", tradeDate, err)
		}

		// Respect rate limits
		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("[tushare-backfill] completed")
	return nil
}

// SyncNews syncs news data for all watchlist stocks from EastMoney to DB.
func (s *TushareSync) SyncNews(ctx context.Context) error {
	start := time.Now()
	log.Printf("[tushare-sync] starting news sync...")

	// Get watchlist codes
	codes, err := s.GetWatchlistCodes(ctx)
	if err != nil {
		return fmt.Errorf("get watchlist codes: %w", err)
	}

	if len(codes) == 0 {
		log.Printf("[tushare-sync] no watchlist stocks to sync news for")
		return nil
	}

	totalInserted := 0
	for _, code := range codes {
		// Fetch news from EastMoney
		items, err := s.eastMoney.FetchStockNews(ctx, code, 20)
		if err != nil {
			s.logger.Warn("failed to fetch news for stock", zap.String("code", code), zap.Error(err))
			continue
		}

		// Store in DB
		for _, item := range items {
			_, err := s.db.Exec(ctx, `
				INSERT INTO stock_news (code, title, summary, source, url, published_at)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (code, title, published_at) DO NOTHING
			`, item.Code, item.Title, item.Summary, item.Source, item.URL, item.PublishedAt)
			if err != nil {
				s.logger.Warn("failed to store news", zap.String("code", code), zap.Error(err))
				continue
			}
			totalInserted++
		}

		// Respect rate limits
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[tushare-sync] news sync completed: %d stocks, %d items inserted in %v",
		len(codes), totalInserted, time.Since(start))
	return nil
}

// SyncAnnouncements syncs announcement data for all watchlist stocks from EastMoney to DB.
func (s *TushareSync) SyncAnnouncements(ctx context.Context) error {
	start := time.Now()
	log.Printf("[tushare-sync] starting announcements sync...")

	// Get watchlist codes
	codes, err := s.GetWatchlistCodes(ctx)
	if err != nil {
		return fmt.Errorf("get watchlist codes: %w", err)
	}

	if len(codes) == 0 {
		log.Printf("[tushare-sync] no watchlist stocks to sync announcements for")
		return nil
	}

	totalInserted := 0
	for _, code := range codes {
		// Fetch announcements from EastMoney
		items, err := s.eastMoney.FetchStockAnnouncements(ctx, code, 20)
		if err != nil {
			s.logger.Warn("failed to fetch announcements for stock", zap.String("code", code), zap.Error(err))
			continue
		}

		// Store in DB
		for _, item := range items {
			_, err := s.db.Exec(ctx, `
				INSERT INTO stock_announcements (code, title, url, published_at)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (code, title, published_at) DO NOTHING
			`, code, item.Title, item.URL, item.PublishedAt)
			if err != nil {
				s.logger.Warn("failed to store announcement", zap.String("code", code), zap.Error(err))
				continue
			}
			totalInserted++
		}

		// Respect rate limits
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[tushare-sync] announcements sync completed: %d stocks, %d items inserted in %v",
		len(codes), totalInserted, time.Since(start))
	return nil
}
