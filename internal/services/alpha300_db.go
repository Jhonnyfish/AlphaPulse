package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Alpha300DBService provides read-only access to the Alpha300 database.
type Alpha300DBService struct {
	db  *pgxpool.Pool
	log *zap.Logger

	// Caches
	klineCache    *cache.Cache[[]models.KlinePoint]
	basicCache    *cache.Cache[dailyBasicData]
	flowCache     *cache.Cache[[]models.MoneyFlowDay]
	industryCache *cache.Cache[string]
	lhbCache      *cache.Cache[[]LhbItem]
	marginCache   *cache.Cache[[]MarginDay]
	rankCache     *cache.Cache[[]RankSnapshot]
	factorCache   *cache.Cache[RankFactors]
}

// dailyBasicData holds valuation data from Alpha300.
type dailyBasicData struct {
	PE_TTM   float64 `json:"pe_ttm"`
	PB       float64 `json:"pb"`
	TotalMV  float64 `json:"total_mv"`
	CircMV   float64 `json:"circ_mv"`
	DVRatio  float64 `json:"dv_ratio"`
}

// LhbItem holds 龙虎榜 data.
type LhbItem struct {
	TradeDate        string  `json:"trade_date"`
	NetBuyAmount     float64 `json:"net_buy_amount"`
	BuyAmount        float64 `json:"buy_amount"`
	SellAmount       float64 `json:"sell_amount"`
	DealAmount       float64 `json:"deal_amount"`
	MarketDealAmount float64 `json:"market_deal_amount"`
	NetBuyPct        float64 `json:"net_buy_pct"`
	DealPct          float64 `json:"deal_pct"`
	TurnoverPct      float64 `json:"turnover_pct"`
	Reason           string  `json:"reason"`
	Interpretation   string  `json:"interpretation"`
}

// MarginDay holds 融资融券 data.
type MarginDay struct {
	TradeDate          string  `json:"trade_date"`
	MarginBuyAmount    float64 `json:"margin_buy_amount"`
	MarginBalance      float64 `json:"margin_balance"`
	ShortSellVolume    float64 `json:"short_sell_volume"`
	ShortVolume        float64 `json:"short_volume"`
	ShortBalance       float64 `json:"short_balance"`
	MarginShortBalance float64 `json:"margin_short_balance"`
}

// RankSnapshot holds ranking data.
type RankSnapshot struct {
	AsOfDate string  `json:"as_of_date"`
	TsCode   string  `json:"ts_code"`
	Score    float64 `json:"score"`
	Rank     int     `json:"rank"`
}

// RankFactors holds factor data for a stock.
type RankFactors struct {
	MomentumRaw      float64 `json:"momentum_raw"`
	MomentumZ        float64 `json:"momentum_z"`
	TrendRaw         float64 `json:"trend_raw"`
	TrendZ           float64 `json:"trend_z"`
	VolatilityRaw    float64 `json:"volatility_raw"`
	VolatilityZ      float64 `json:"volatility_z"`
	LiquidityRaw     float64 `json:"liquidity_raw"`
	LiquidityZ       float64 `json:"liquidity_z"`
	Reversal5dRaw    float64 `json:"reversal_5d_raw"`
	Reversal5dZ      float64 `json:"reversal_5d_z"`
	CapitalFlowRaw   float64 `json:"capital_flow_raw"`
	CapitalFlowZ     float64 `json:"capital_flow_z"`
}

// NewAlpha300DBService creates a new Alpha300DBService.
func NewAlpha300DBService(db *pgxpool.Pool, log *zap.Logger) *Alpha300DBService {
	return &Alpha300DBService{
		db:            db,
		log:           log,
		klineCache:    cache.New[[]models.KlinePoint](),
		basicCache:    cache.New[dailyBasicData](),
		flowCache:     cache.New[[]models.MoneyFlowDay](),
		industryCache: cache.New[string](),
		lhbCache:      cache.New[[]LhbItem](),
		marginCache:   cache.New[[]MarginDay](),
		rankCache:     cache.New[[]RankSnapshot](),
		factorCache:   cache.New[RankFactors](),
	}
}

// normalizeCode is defined in alpha300.go

// toDisplayCode converts ts_code to 6-digit code (e.g., 600519.SH → 600519)
func toDisplayCode(tsCode string) string {
	return strings.Split(tsCode, ".")[0]
}

// formatDate converts YYYYMMDD to YYYY-MM-DD
func formatDate(date string) string {
	if len(date) == 8 {
		return date[:4] + "-" + date[4:6] + "-" + date[6:8]
	}
	return date
}

// ==================== K-line Data ====================

// FetchKline fetches daily K-line data from Alpha300.
func (s *Alpha300DBService) FetchKline(ctx context.Context, code string, days int) ([]models.KlinePoint, error) {
	tsCode := normalizeCode(code)
	cacheKey := fmt.Sprintf("kline:%s:%d", tsCode, days)

	if cached, ok := s.klineCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT trade_date, open, high, low, close, vol, amount
		FROM daily
		WHERE ts_code = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, tsCode, days)
	if err != nil {
		return nil, fmt.Errorf("query kline: %w", err)
	}
	defer rows.Close()

	var klines []models.KlinePoint
	for rows.Next() {
		var k models.KlinePoint
		var tradeDate string
		if err := rows.Scan(&tradeDate, &k.Open, &k.High, &k.Low, &k.Close, &k.Volume, &k.Amount); err != nil {
			s.log.Warn("scan kline", zap.Error(err))
			continue
		}
		k.Date = formatDate(tradeDate)
		klines = append(klines, k)
	}

	// Reverse to ascending order
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	s.klineCache.Set(cacheKey, klines, 5*time.Minute)
	return klines, nil
}

// ==================== Daily Basic (Valuation) ====================

// FetchDailyBasic fetches latest valuation data from Alpha300.
func (s *Alpha300DBService) FetchDailyBasic(ctx context.Context, code string) (dailyBasicData, error) {
	tsCode := normalizeCode(code)
	cacheKey := fmt.Sprintf("basic:%s", tsCode)

	if cached, ok := s.basicCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT pe_ttm, pb, total_mv, circ_mv, dv_ratio
		FROM daily_basic
		WHERE ts_code = $1
		ORDER BY trade_date DESC
		LIMIT 1
	`

	var b dailyBasicData
	err := s.db.QueryRow(ctx, query, tsCode).Scan(&b.PE_TTM, &b.PB, &b.TotalMV, &b.CircMV, &b.DVRatio)
	if err != nil {
		return dailyBasicData{}, fmt.Errorf("query daily_basic: %w", err)
	}

	s.basicCache.Set(cacheKey, b, 10*time.Minute)
	return b, nil
}

// ==================== Money Flow ====================

// FetchMoneyFlow fetches money flow data from Alpha300.
func (s *Alpha300DBService) FetchMoneyFlow(ctx context.Context, code string, days int) ([]models.MoneyFlowDay, error) {
	tsCode := normalizeCode(code)
	cacheKey := fmt.Sprintf("flow:%s:%d", tsCode, days)

	if cached, ok := s.flowCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT trade_date, main_net_amount, main_net_pct, super_net_amount, super_net_pct, large_net_amount, large_net_pct
		FROM stock_moneyflow
		WHERE ts_code = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, tsCode, days)
	if err != nil {
		return nil, fmt.Errorf("query moneyflow: %w", err)
	}
	defer rows.Close()

	var flows []models.MoneyFlowDay
	for rows.Next() {
		var f models.MoneyFlowDay
		var tradeDate string
		var mainNet, mainNetPct, superNet, superNetPct, largeNet, largeNetPct float64
		if err := rows.Scan(&tradeDate, &mainNet, &mainNetPct, &superNet, &superNetPct, &largeNet, &largeNetPct); err != nil {
			s.log.Warn("scan moneyflow", zap.Error(err))
			continue
		}
		f.Date = formatDate(tradeDate)
		// Convert from raw amount to 万 (10000) and map to model fields
		f.MainNet = mainNet / 10000
		f.MainNetPct = mainNetPct
		f.HugeNet = superNet / 10000
		f.HugeNetPct = superNetPct
		f.BigNet = largeNet / 10000
		f.BigNetPct = largeNetPct
		flows = append(flows, f)
	}

	// Reverse to ascending order
	for i, j := 0, len(flows)-1; i < j; i, j = i+1, j-1 {
		flows[i], flows[j] = flows[j], flows[i]
	}

	s.flowCache.Set(cacheKey, flows, 3*time.Minute)
	return flows, nil
}

// ==================== Industry ====================

// FetchIndustry fetches industry classification from Alpha300.
func (s *Alpha300DBService) FetchIndustry(ctx context.Context, code string) (string, error) {
	tsCode := normalizeCode(code)
	cacheKey := fmt.Sprintf("industry:%s", tsCode)

	if cached, ok := s.industryCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `SELECT industry FROM stock_industry WHERE ts_code = $1 LIMIT 1`
	var industry string
	err := s.db.QueryRow(ctx, query, tsCode).Scan(&industry)
	if err != nil {
		return "", fmt.Errorf("query industry: %w", err)
	}

	s.industryCache.Set(cacheKey, industry, 1*time.Hour)
	return industry, nil
}

// ==================== 龙虎榜 ====================

// FetchLhb fetches 龙虎榜 data from Alpha300.
func (s *Alpha300DBService) FetchLhb(ctx context.Context, code string, days int) ([]LhbItem, error) {
	tsCode := normalizeCode(code)
	cacheKey := fmt.Sprintf("lhb:%s:%d", tsCode, days)

	if cached, ok := s.lhbCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT trade_date, net_buy_amount, buy_amount, sell_amount, deal_amount, 
		       market_deal_amount, net_buy_pct, deal_pct, turnover_pct, reason, interpretation
		FROM stock_lhb
		WHERE ts_code = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, tsCode, days)
	if err != nil {
		return nil, fmt.Errorf("query lhb: %w", err)
	}
	defer rows.Close()

	var items []LhbItem
	for rows.Next() {
		var item LhbItem
		if err := rows.Scan(&item.TradeDate, &item.NetBuyAmount, &item.BuyAmount, &item.SellAmount,
			&item.DealAmount, &item.MarketDealAmount, &item.NetBuyPct, &item.DealPct,
			&item.TurnoverPct, &item.Reason, &item.Interpretation); err != nil {
			s.log.Warn("scan lhb", zap.Error(err))
			continue
		}
		item.TradeDate = formatDate(item.TradeDate)
		items = append(items, item)
	}

	s.lhbCache.Set(cacheKey, items, 30*time.Minute)
	return items, nil
}

// ==================== 融资融券 ====================

// FetchMargin fetches 融资融券 data from Alpha300.
func (s *Alpha300DBService) FetchMargin(ctx context.Context, code string, days int) ([]MarginDay, error) {
	tsCode := normalizeCode(code)
	cacheKey := fmt.Sprintf("margin:%s:%d", tsCode, days)

	if cached, ok := s.marginCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT trade_date, margin_buy_amount, margin_balance, short_sell_volume, 
		       short_volume, short_balance, margin_short_balance
		FROM stock_margin
		WHERE ts_code = $1
		ORDER BY trade_date DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, tsCode, days)
	if err != nil {
		return nil, fmt.Errorf("query margin: %w", err)
	}
	defer rows.Close()

	var items []MarginDay
	for rows.Next() {
		var item MarginDay
		if err := rows.Scan(&item.TradeDate, &item.MarginBuyAmount, &item.MarginBalance,
			&item.ShortSellVolume, &item.ShortVolume, &item.ShortBalance, &item.MarginShortBalance); err != nil {
			s.log.Warn("scan margin", zap.Error(err))
			continue
		}
		item.TradeDate = formatDate(item.TradeDate)
		items = append(items, item)
	}

	s.marginCache.Set(cacheKey, items, 10*time.Minute)
	return items, nil
}

// ==================== Ranking ====================

// FetchRankSnapshot fetches ranking snapshot from Alpha300.
func (s *Alpha300DBService) FetchRankSnapshot(ctx context.Context, date string) ([]RankSnapshot, error) {
	cacheKey := fmt.Sprintf("rank:%s", date)

	if cached, ok := s.rankCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT as_of_date, ts_code, score, rank
		FROM rank_snapshot
		WHERE as_of_date = $1
		ORDER BY rank ASC
	`

	rows, err := s.db.Query(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("query rank_snapshot: %w", err)
	}
	defer rows.Close()

	var items []RankSnapshot
	for rows.Next() {
		var item RankSnapshot
		if err := rows.Scan(&item.AsOfDate, &item.TsCode, &item.Score, &item.Rank); err != nil {
			s.log.Warn("scan rank_snapshot", zap.Error(err))
			continue
		}
		item.AsOfDate = formatDate(item.AsOfDate)
		items = append(items, item)
	}

	s.rankCache.Set(cacheKey, items, 5*time.Minute)
	return items, nil
}

// FetchLatestRankSnapshot fetches the latest ranking snapshot.
func (s *Alpha300DBService) FetchLatestRankSnapshot(ctx context.Context) ([]RankSnapshot, error) {
	query := `SELECT MAX(as_of_date) FROM rank_snapshot`
	var date string
	if err := s.db.QueryRow(ctx, query).Scan(&date); err != nil {
		return nil, fmt.Errorf("query latest rank date: %w", err)
	}
	return s.FetchRankSnapshot(ctx, date)
}

// FetchRankFactors fetches rank factors for a stock.
func (s *Alpha300DBService) FetchRankFactors(ctx context.Context, code string) (RankFactors, error) {
	tsCode := normalizeCode(code)
	cacheKey := fmt.Sprintf("factor:%s", tsCode)

	if cached, ok := s.factorCache.Get(cacheKey); ok {
		return cached, nil
	}

	query := `
		SELECT momentum_raw, momentum_z, trend_raw, trend_z, 
		       volatility_raw, volatility_z, liquidity_raw, liquidity_z,
		       reversal_5d_raw, reversal_5d_z, capital_flow_raw, capital_flow_z
		FROM rank_factor
		WHERE ts_code = $1
		ORDER BY run_id DESC
		LIMIT 1
	`

	var f RankFactors
	err := s.db.QueryRow(ctx, query, tsCode).Scan(
		&f.MomentumRaw, &f.MomentumZ, &f.TrendRaw, &f.TrendZ,
		&f.VolatilityRaw, &f.VolatilityZ, &f.LiquidityRaw, &f.LiquidityZ,
		&f.Reversal5dRaw, &f.Reversal5dZ, &f.CapitalFlowRaw, &f.CapitalFlowZ,
	)
	if err != nil {
		return RankFactors{}, fmt.Errorf("query rank_factor: %w", err)
	}

	s.factorCache.Set(cacheKey, f, 5*time.Minute)
	return f, nil
}

// ==================== Stock Basic ====================

// FetchStockName fetches stock name from Alpha300.
func (s *Alpha300DBService) FetchStockName(ctx context.Context, code string) (string, error) {
	tsCode := normalizeCode(code)
	query := `SELECT name FROM stock_basic WHERE ts_code = $1 LIMIT 1`
	var name string
	err := s.db.QueryRow(ctx, query, tsCode).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("query stock_basic: %w", err)
	}
	return name, nil
}

// ==================== Health Check ====================

// Ping checks if the Alpha300 database is reachable.
func (s *Alpha300DBService) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}
