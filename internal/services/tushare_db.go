package services

import (
	"context"
	"fmt"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DailyBasicData holds valuation data from Tushare local DB.
type DailyBasicData struct {
	PE_TTM  float64 `json:"pe_ttm"`
	PB      float64 `json:"pb"`
	TotalMV float64 `json:"total_mv"`
	CircMV  float64 `json:"circ_mv"`
	DVRatio float64 `json:"dv_ratio"`
}

// TushareDB provides read-only access to local Tushare data tables.
type TushareDB struct {
	db     *pgxpool.Pool
	logger *zap.Logger

	klineCache    *cache.Cache[[]models.KlinePoint]
	basicCache    *cache.Cache[DailyBasicData]
	flowCache     *cache.Cache[[]models.MoneyFlowDay]
	industryCache *cache.Cache[string]
	nameCache     *cache.Cache[string]
}

// NewTushareDB creates a new TushareDB service.
func NewTushareDB(db *pgxpool.Pool, log *zap.Logger) *TushareDB {
	return &TushareDB{
		db:            db,
		logger:        log,
		klineCache:    cache.New[[]models.KlinePoint](),
		basicCache:    cache.New[DailyBasicData](),
		flowCache:     cache.New[[]models.MoneyFlowDay](),
		industryCache: cache.New[string](),
		nameCache:     cache.New[string](),
	}
}

// ==================== K-line Data ====================

// FetchKline fetches daily K-line data from local Tushare tables.
func (s *TushareDB) FetchKline(ctx context.Context, code string, days int) ([]models.KlinePoint, error) {
	tsCode := ToTsCode(code)
	cacheKey := fmt.Sprintf("kline:%s:%d", tsCode, days)

	if cached, ok := s.klineCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT trade_date, open, high, low, close, vol, amount
		FROM tushare_daily
		WHERE ts_code = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, tsCode, days)
	if err != nil {
		return nil, fmt.Errorf("query tushare_daily: %w", err)
	}
	defer rows.Close()

	var klines []models.KlinePoint
	for rows.Next() {
		var k models.KlinePoint
		var tradeDate string
		if err := rows.Scan(&tradeDate, &k.Open, &k.High, &k.Low, &k.Close, &k.Volume, &k.Amount); err != nil {
			s.logger.Warn("scan kline", zap.Error(err))
			continue
		}
		k.Date = FormatDate(tradeDate)
		klines = append(klines, k)
	}

	if len(klines) == 0 {
		return nil, fmt.Errorf("no kline data for %s", code)
	}

	// Reverse to ascending order
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	s.klineCache.Set(cacheKey, klines, 5*time.Minute)
	return klines, nil
}

// ==================== Daily Basic (Valuation) ====================

// FetchDailyBasic fetches latest valuation data from local Tushare tables.
func (s *TushareDB) FetchDailyBasic(ctx context.Context, code string) (DailyBasicData, error) {
	tsCode := ToTsCode(code)
	cacheKey := fmt.Sprintf("basic:%s", tsCode)

	if cached, ok := s.basicCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT pe_ttm, pb, total_mv, circ_mv, dv_ratio
		FROM tushare_daily_basic
		WHERE ts_code = $1
		ORDER BY trade_date DESC
		LIMIT 1
	`

	var b DailyBasicData
	err := s.db.QueryRow(ctx, query, tsCode).Scan(&b.PE_TTM, &b.PB, &b.TotalMV, &b.CircMV, &b.DVRatio)
	if err != nil {
		return DailyBasicData{}, fmt.Errorf("query tushare_daily_basic: %w", err)
	}

	s.basicCache.Set(cacheKey, b, 10*time.Minute)
	return b, nil
}

// ==================== Money Flow ====================

// FetchMoneyFlow fetches money flow data from local Tushare tables.
func (s *TushareDB) FetchMoneyFlow(ctx context.Context, code string, days int) ([]models.MoneyFlowDay, error) {
	tsCode := ToTsCode(code)
	cacheKey := fmt.Sprintf("flow:%s:%d", tsCode, days)

	if cached, ok := s.flowCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT trade_date,
		       net_mf_amount,
		       buy_elg_amount + buy_lg_amount AS big_buy,
		       sell_elg_amount + sell_lg_amount AS big_sell,
		       buy_sm_amount AS small_buy,
		       sell_sm_amount AS small_sell
		FROM tushare_moneyflow
		WHERE ts_code = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, tsCode, days)
	if err != nil {
		return nil, fmt.Errorf("query tushare_moneyflow: %w", err)
	}
	defer rows.Close()

	var flows []models.MoneyFlowDay
	for rows.Next() {
		var f models.MoneyFlowDay
		var tradeDate string
		var mainNet, bigBuy, bigSell, smallBuy, smallSell *float64
		if err := rows.Scan(&tradeDate, &mainNet, &bigBuy, &bigSell, &smallBuy, &smallSell); err != nil {
			s.logger.Warn("scan moneyflow", zap.Error(err))
			continue
		}
		f.Date = FormatDate(tradeDate)
		// Tushare moneyflow amounts are in 万元
		if mainNet != nil {
			f.MainNet = *mainNet
			f.HugeNet = *mainNet // net_mf_amount is total net
		}
		if bigBuy != nil && bigSell != nil {
			f.BigNet = *bigBuy - *bigSell
		}
		if smallBuy != nil && smallSell != nil {
			f.SmallNet = *smallBuy - *smallSell
		}
		flows = append(flows, f)
	}

	if len(flows) == 0 {
		return nil, fmt.Errorf("no moneyflow data for %s", code)
	}

	// Reverse to ascending order
	for i, j := 0, len(flows)-1; i < j; i, j = i+1, j-1 {
		flows[i], flows[j] = flows[j], flows[i]
	}

	s.flowCache.Set(cacheKey, flows, 3*time.Minute)
	return flows, nil
}

// ==================== Industry ====================

// FetchIndustry fetches industry classification from local Tushare tables.
func (s *TushareDB) FetchIndustry(ctx context.Context, code string) (string, error) {
	tsCode := ToTsCode(code)
	cacheKey := fmt.Sprintf("industry:%s", tsCode)

	if cached, ok := s.industryCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `SELECT industry FROM tushare_stock_basic WHERE ts_code = $1 LIMIT 1`
	var industry string
	err := s.db.QueryRow(ctx, query, tsCode).Scan(&industry)
	if err != nil {
		return "", fmt.Errorf("query tushare_stock_basic industry: %w", err)
	}

	s.industryCache.Set(cacheKey, industry, 1*time.Hour)
	return industry, nil
}

// ==================== Stock Name ====================

// FetchStockName fetches stock name from local Tushare tables.
func (s *TushareDB) FetchStockName(ctx context.Context, code string) (string, error) {
	tsCode := ToTsCode(code)
	cacheKey := fmt.Sprintf("name:%s", tsCode)

	if cached, ok := s.nameCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `SELECT name FROM tushare_stock_basic WHERE ts_code = $1 LIMIT 1`
	var name string
	err := s.db.QueryRow(ctx, query, tsCode).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("query tushare_stock_basic name: %w", err)
	}

	s.nameCache.Set(cacheKey, name, 1*time.Hour)
	return name, nil
}

// ==================== Health Check ====================

// Ping checks if the database is reachable.
func (s *TushareDB) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

// HasData checks if any data exists in the tushare tables.
func (s *TushareDB) HasData(ctx context.Context) bool {
	var count int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM tushare_stock_basic LIMIT 1`).Scan(&count)
	return err == nil && count > 0
}

// ==================== Adj Factor ====================

// AdjFactorDay holds one day's adjustment factor.
type AdjFactorDay struct {
	Date      string  `json:"date"`
	AdjFactor float64 `json:"adj_factor"`
}

// FetchAdjFactor fetches adjustment factor data from local Tushare tables.
func (s *TushareDB) FetchAdjFactor(ctx context.Context, code string, days int) ([]AdjFactorDay, error) {
	tsCode := ToTsCode(code)
	query := `
		SELECT trade_date, adj_factor
		FROM tushare_adj_factor
		WHERE ts_code = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`
	rows, err := s.db.Query(ctx, query, tsCode, days)
	if err != nil {
		return nil, fmt.Errorf("query tushare_adj_factor: %w", err)
	}
	defer rows.Close()

	var result []AdjFactorDay
	for rows.Next() {
		var a AdjFactorDay
		var tradeDate string
		if err := rows.Scan(&tradeDate, &a.AdjFactor); err != nil {
			continue
		}
		a.Date = FormatDate(tradeDate)
		result = append(result, a)
	}
	return result, nil
}

// ==================== Index Daily ====================

// FetchIndexDaily fetches index K-line data from local Tushare tables.
func (s *TushareDB) FetchIndexDaily(ctx context.Context, tsCode string, days int) ([]models.KlinePoint, error) {
	query := `
		SELECT trade_date, open, high, low, close, vol, amount
		FROM tushare_index_daily
		WHERE ts_code = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`
	rows, err := s.db.Query(ctx, query, tsCode, days)
	if err != nil {
		return nil, fmt.Errorf("query tushare_index_daily: %w", err)
	}
	defer rows.Close()

	var klines []models.KlinePoint
	for rows.Next() {
		var k models.KlinePoint
		var tradeDate string
		if err := rows.Scan(&tradeDate, &k.Open, &k.High, &k.Low, &k.Close, &k.Volume, &k.Amount); err != nil {
			continue
		}
		k.Date = FormatDate(tradeDate)
		klines = append(klines, k)
	}

	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}
	return klines, nil
}

// ==================== Top List (Dragon Tiger) ====================

// TopListItem holds dragon tiger board data.
type TopListItem struct {
	TradeDate    string  `json:"trade_date"`
	TsCode       string  `json:"ts_code"`
	Name         string  `json:"name"`
	Close        float64 `json:"close"`
	PctChange    float64 `json:"pct_change"`
	TurnoverRate float64 `json:"turnover_rate"`
	Amount       float64 `json:"amount"`
	NetAmount    float64 `json:"net_amount"`
	Reason       string  `json:"reason"`
}

// FetchTopList fetches dragon tiger board data from local Tushare tables.
func (s *TushareDB) FetchTopList(ctx context.Context, tradeDate string) ([]TopListItem, error) {
	query := `
		SELECT trade_date, ts_code, name, close, pct_change, turnover_rate,
		       amount, net_amount, reason
		FROM tushare_top_list
		WHERE trade_date = $1
		ORDER BY amount DESC
	`
	rows, err := s.db.Query(ctx, query, tradeDate)
	if err != nil {
		return nil, fmt.Errorf("query tushare_top_list: %w", err)
	}
	defer rows.Close()

	var items []TopListItem
	for rows.Next() {
		var item TopListItem
		if err := rows.Scan(&item.TradeDate, &item.TsCode, &item.Name, &item.Close,
			&item.PctChange, &item.TurnoverRate, &item.Amount, &item.NetAmount, &item.Reason); err != nil {
			continue
		}
		item.TradeDate = FormatDate(item.TradeDate)
		item.TsCode = ToDisplayCode(item.TsCode)
		items = append(items, item)
	}
	return items, nil
}

// ==================== Margin ====================

// MarginDayData holds market-level margin data.
type MarginDayData struct {
	TradeDate string  `json:"trade_date"`
	Exchange  string  `json:"exchange"`
	Rzye      float64 `json:"rzye"`
	Rzmre     float64 `json:"rzmre"`
	Rqye      float64 `json:"rqye"`
	Rzrqye    float64 `json:"rzrqye"`
}

// FetchMargin fetches margin trading data from local Tushare tables.
func (s *TushareDB) FetchMargin(ctx context.Context, days int) ([]MarginDayData, error) {
	query := `
		SELECT trade_date, exchange_id, rzye, rzmre, rqye, rzrqye
		FROM tushare_margin
		ORDER BY trade_date DESC
		LIMIT $1
	`
	rows, err := s.db.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("query tushare_margin: %w", err)
	}
	defer rows.Close()

	var items []MarginDayData
	for rows.Next() {
		var item MarginDayData
		if err := rows.Scan(&item.TradeDate, &item.Exchange, &item.Rzye, &item.Rzmre, &item.Rqye, &item.Rzrqye); err != nil {
			continue
		}
		item.TradeDate = FormatDate(item.TradeDate)
		items = append(items, item)
	}
	return items, nil
}

// ==================== Trading Calendar ====================

// IsTradingDay checks if a given date is a trading day.
func (s *TushareDB) IsTradingDay(ctx context.Context, date string) bool {
	var isOpen int
	err := s.db.QueryRow(ctx,
		`SELECT is_open FROM tushare_trade_cal WHERE exchange = 'SSE' AND cal_date = $1`,
		date).Scan(&isOpen)
	return err == nil && isOpen == 1
}

// LatestTradeDate returns the most recent trading date in the daily table.
func (s *TushareDB) LatestTradeDate(ctx context.Context) (string, error) {
	var date string
	err := s.db.QueryRow(ctx, `SELECT MAX(trade_date) FROM tushare_daily`).Scan(&date)
	if err != nil {
		return "", err
	}
	return date, nil
}

// ==================== Quote from DB ====================

// FetchQuoteFromDB constructs a Quote from the latest tushare_daily + tushare_daily_basic rows.
// Returns the latest quote data purely from the local database — no external API calls.
func (s *TushareDB) FetchQuoteFromDB(ctx context.Context, code string) (models.Quote, error) {
	tsCode := ToTsCode(code)
	query := `
		SELECT d.trade_date, d.open, d.high, d.low, d.close, d.pre_close,
		       d.change, d.pct_chg, d.vol, d.amount,
		       b.pe_ttm, b.pb, b.total_mv, b.circ_mv,
		       COALESCE(s.name, ''), COALESCE(s.industry, '')
		FROM tushare_daily d
		LEFT JOIN tushare_daily_basic b ON d.ts_code = b.ts_code AND d.trade_date = b.trade_date
		LEFT JOIN tushare_stock_basic s ON d.ts_code = s.ts_code
		WHERE d.ts_code = $1
		ORDER BY d.trade_date DESC
		LIMIT 1
	`

	var q models.Quote
	var tradeDate string
	var peTTM, pb, totalMV, circMV *float64
	var industry string // discarded, used for scan alignment

	err := s.db.QueryRow(ctx, query, tsCode).Scan(
		&tradeDate, &q.Open, &q.High, &q.Low, &q.Price, &q.PrevClose,
		&q.Change, &q.ChangePercent, &q.Volume, &q.Turnover,
		&peTTM, &pb, &totalMV, &circMV,
		&q.Name, &industry,
	)
	if err != nil {
		return models.Quote{}, fmt.Errorf("query quote from db: %w", err)
	}

	if peTTM != nil {
		q.PE = *peTTM
	}
	if pb != nil {
		q.PB = *pb
	}
	if totalMV != nil {
		q.TotalMV = *totalMV
	}
	q.Code = StockCode6(code)
	q.UpdatedAt = tradeDate

	// Calculate amplitude
	if q.PrevClose > 0 {
		q.Amplitude = (q.High - q.Low) / q.PrevClose * 100
	}

	return q, nil
}

// FetchIndustryFromDB fetches industry from tushare_stock_basic (alias for FetchIndustry).
func (s *TushareDB) FetchIndustryFromDB(ctx context.Context, code string) (string, error) {
	return s.FetchIndustry(ctx, code)
}

// ==================== Adjusted K-line ====================

// FetchAdjKline fetches forward-adjusted K-line data by joining daily with adj_factor.
// Forward adjustment formula: price * adj_factor / latest_adj_factor
func (s *TushareDB) FetchAdjKline(ctx context.Context, code string, days int) ([]models.KlinePoint, error) {
	tsCode := ToTsCode(code)
	cacheKey := fmt.Sprintf("adj_kline:%s:%d", tsCode, days)

	if cached, ok := s.klineCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT d.trade_date,
		       d.open  * a.adj_factor / l.latest_adj,
		       d.high  * a.adj_factor / l.latest_adj,
		       d.low   * a.adj_factor / l.latest_adj,
		       d.close * a.adj_factor / l.latest_adj,
		       d.vol,
		       d.amount
		FROM tushare_daily d
		JOIN tushare_adj_factor a ON d.ts_code = a.ts_code AND d.trade_date = a.trade_date
		CROSS JOIN LATERAL (
			SELECT adj_factor AS latest_adj
			FROM tushare_adj_factor
			WHERE ts_code = d.ts_code
			ORDER BY trade_date DESC LIMIT 1
		) l
		WHERE d.ts_code = $1
		ORDER BY d.trade_date DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, tsCode, days)
	if err != nil {
		return nil, fmt.Errorf("query adj kline: %w", err)
	}
	defer rows.Close()

	var klines []models.KlinePoint
	for rows.Next() {
		var k models.KlinePoint
		var tradeDate string
		if err := rows.Scan(&tradeDate, &k.Open, &k.High, &k.Low, &k.Close, &k.Volume, &k.Amount); err != nil {
			s.logger.Warn("scan adj kline", zap.Error(err))
			continue
		}
		k.Date = FormatDate(tradeDate)
		klines = append(klines, k)
	}

	if len(klines) == 0 {
		return nil, fmt.Errorf("no adj kline data for %s", code)
	}

	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	s.klineCache.Set(cacheKey, klines, 5*time.Minute)
	return klines, nil
}
