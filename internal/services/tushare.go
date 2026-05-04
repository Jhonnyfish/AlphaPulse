package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	apperrors "alphapulse/internal/errors"
	"alphapulse/internal/logger"

	"go.uber.org/zap"
)

const tushareBaseURL = "https://api.tushare.pro"

// TushareResponse is the generic response from Tushare Pro API.
type TushareResponse struct {
	RequestID string `json:"request_id"`
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	Data      struct {
		Fields []string        `json:"fields"`
		Items  [][]interface{} `json:"items"`
	} `json:"data"`
}

// TushareRequest is the request body sent to Tushare Pro API.
type TushareRequest struct {
	APIName string            `json:"api_name"`
	Token   string            `json:"token"`
	Params  map[string]string `json:"params"`
	Fields  string            `json:"fields"`
}

// Row maps a single row from Tushare's columnar response to a map.
type Row map[string]interface{}

// TushareService is an HTTP client for the Tushare Pro API.
type TushareService struct {
	token  string
	client *http.Client
	logger *zap.Logger

	// Simple rate limiter: track last request time
	mu         sync.Mutex
	lastReq    time.Time
	minInterval time.Duration // minimum time between requests
}

// NewTushareService creates a new TushareService.
func NewTushareService(token string, timeout time.Duration) *TushareService {
	return &TushareService{
		token:       token,
		client:      &http.Client{Timeout: timeout},
		logger:      logger.L(),
		minInterval: 200 * time.Millisecond, // max ~5 req/s
	}
}

// Query sends a request to Tushare Pro API and returns the parsed response.
func (s *TushareService) Query(ctx context.Context, apiName string, params map[string]string, fields string) (*TushareResponse, error) {
	reqBody := TushareRequest{
		APIName: apiName,
		Token:   s.token,
		Params:  params,
		Fields:  fields,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, apperrors.Internal(fmt.Errorf("marshal tushare request: %w", err))
	}

	var resp *TushareResponse
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			jitter := time.Duration(rand.Int63n(int64(500))) * time.Millisecond
			time.Sleep(delay + jitter)
		}

		// Rate limit
		s.mu.Lock()
		elapsed := time.Since(s.lastReq)
		if elapsed < s.minInterval {
			time.Sleep(s.minInterval - elapsed)
		}
		s.lastReq = time.Now()
		s.mu.Unlock()

		resp, err = s.doRequest(ctx, body)
		if err != nil {
			s.logger.Warn("tushare request failed, retrying",
				zap.String("api", apiName),
				zap.Int("attempt", attempt+1),
				zap.Error(err))
			continue
		}

		if resp.Code != 0 {
			err = fmt.Errorf("tushare api error [%d]: %s", resp.Code, resp.Msg)
			s.logger.Warn("tushare api returned error",
				zap.String("api", apiName),
				zap.Int("code", resp.Code),
				zap.String("msg", resp.Msg))
			// Don't retry on business logic errors
			if resp.Code == 1001 || resp.Code == 1002 || resp.Code == 1003 {
				return nil, apperrors.Internal(err)
			}
			continue
		}

		return resp, nil
	}

	if resp != nil && resp.Code != 0 {
		return nil, apperrors.Internal(fmt.Errorf("tushare api error [%d]: %s after %d retries", resp.Code, resp.Msg, maxRetries))
	}
	return nil, apperrors.Internal(fmt.Errorf("tushare request failed after %d retries: %w", maxRetries, err))
}

func (s *TushareService) doRequest(ctx context.Context, body []byte) (*TushareResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tushareBaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tushare http status %d: %s", resp.StatusCode, string(respBody))
	}

	var tushareResp TushareResponse
	if err := json.Unmarshal(respBody, &tushareResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &tushareResp, nil
}

// parseRows converts Tushare's columnar response into a slice of Row maps.
func parseRows(resp *TushareResponse) []Row {
	if resp == nil || len(resp.Data.Fields) == 0 {
		return nil
	}
	rows := make([]Row, 0, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		row := make(Row, len(resp.Data.Fields))
		for i, field := range resp.Data.Fields {
			if i < len(item) {
				row[field] = item[i]
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// ========== Typed Fetch Methods ==========

// DailyRow represents a row from the daily API.
type DailyRow struct {
	TsCode    string
	TradeDate string
	Open      float64
	High      float64
	Low       float64
	Close     float64
	PreClose  float64
	Change    float64
	PctChg    float64
	Vol       float64
	Amount    float64
}

// FetchDaily fetches daily K-line data from Tushare.
func (s *TushareService) FetchDaily(ctx context.Context, tsCode, startDate, endDate string) ([]DailyRow, error) {
	params := map[string]string{}
	if tsCode != "" {
		params["ts_code"] = tsCode
	}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	resp, err := s.Query(ctx, "daily", params, "")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]DailyRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, DailyRow{
			TsCode:    strVal(r, "ts_code"),
			TradeDate: strVal(r, "trade_date"),
			Open:      floatVal(r, "open"),
			High:      floatVal(r, "high"),
			Low:       floatVal(r, "low"),
			Close:     floatVal(r, "close"),
			PreClose:  floatVal(r, "pre_close"),
			Change:    floatVal(r, "change"),
			PctChg:    floatVal(r, "pct_chg"),
			Vol:       floatVal(r, "vol"),
			Amount:    floatVal(r, "amount"),
		})
	}
	return result, nil
}

// DailyBasicRow represents a row from the daily_basic API.
type DailyBasicRow struct {
	TsCode       string
	TradeDate    string
	Close        float64
	TurnoverRate float64
	PE           float64
	PE_TTM       float64
	PB           float64
	PS           float64
	PS_TTM       float64
	DVRatio      float64
	DV_TTM       float64
	TotalShare   float64
	FloatShare   float64
	FreeShare    float64
	TotalMV      float64
	CircMV       float64
}

// FetchDailyBasic fetches daily valuation indicators from Tushare.
func (s *TushareService) FetchDailyBasic(ctx context.Context, tsCode, tradeDate string) ([]DailyBasicRow, error) {
	params := map[string]string{}
	if tsCode != "" {
		params["ts_code"] = tsCode
	}
	if tradeDate != "" {
		params["trade_date"] = tradeDate
	}

	resp, err := s.Query(ctx, "daily_basic", params, "")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]DailyBasicRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, DailyBasicRow{
			TsCode:       strVal(r, "ts_code"),
			TradeDate:    strVal(r, "trade_date"),
			Close:        floatVal(r, "close"),
			TurnoverRate: floatVal(r, "turnover_rate"),
			PE:           floatVal(r, "pe"),
			PE_TTM:       floatVal(r, "pe_ttm"),
			PB:           floatVal(r, "pb"),
			PS:           floatVal(r, "ps"),
			PS_TTM:       floatVal(r, "ps_ttm"),
			DVRatio:      floatVal(r, "dv_ratio"),
			DV_TTM:       floatVal(r, "dv_ttm"),
			TotalShare:   floatVal(r, "total_share"),
			FloatShare:   floatVal(r, "float_share"),
			FreeShare:    floatVal(r, "free_share"),
			TotalMV:      floatVal(r, "total_mv"),
			CircMV:       floatVal(r, "circ_mv"),
		})
	}
	return result, nil
}

// StockBasicRow represents a row from the stock_basic API.
type StockBasicRow struct {
	TsCode   string
	Symbol   string
	Name     string
	Industry string
	Market   string
	ListDate string
}

// FetchStockBasic fetches the stock master list from Tushare.
func (s *TushareService) FetchStockBasic(ctx context.Context) ([]StockBasicRow, error) {
	params := map[string]string{
		"exchange":    "",
		"list_status": "L",
	}

	resp, err := s.Query(ctx, "stock_basic", params, "ts_code,symbol,name,industry,market,list_date")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]StockBasicRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, StockBasicRow{
			TsCode:   strVal(r, "ts_code"),
			Symbol:   strVal(r, "symbol"),
			Name:     strVal(r, "name"),
			Industry: strVal(r, "industry"),
			Market:   strVal(r, "market"),
			ListDate: strVal(r, "list_date"),
		})
	}
	return result, nil
}

// MoneyFlowRow represents a row from the moneyflow API.
type MoneyFlowRow struct {
	TsCode         string
	TradeDate      string
	BuySmVol       int64
	BuySmAmount    float64
	SellSmVol      int64
	SellSmAmount   float64
	BuyMdVol       int64
	BuyMdAmount    float64
	SellMdVol      int64
	SellMdAmount   float64
	BuyLgVol       int64
	BuyLgAmount    float64
	SellLgVol      int64
	SellLgAmount   float64
	BuyElgVol      int64
	BuyElgAmount   float64
	SellElgVol     int64
	SellElgAmount  float64
	NetMfVol       int64
	NetMfAmount    float64
}

// FetchMoneyFlow fetches money flow data from Tushare.
func (s *TushareService) FetchMoneyFlow(ctx context.Context, tsCode, startDate, endDate string) ([]MoneyFlowRow, error) {
	params := map[string]string{}
	if tsCode != "" {
		params["ts_code"] = tsCode
	}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	resp, err := s.Query(ctx, "moneyflow", params, "")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]MoneyFlowRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, MoneyFlowRow{
			TsCode:        strVal(r, "ts_code"),
			TradeDate:     strVal(r, "trade_date"),
			BuySmVol:      int64Val(r, "buy_sm_vol"),
			BuySmAmount:   floatVal(r, "buy_sm_amount"),
			SellSmVol:     int64Val(r, "sell_sm_vol"),
			SellSmAmount:  floatVal(r, "sell_sm_amount"),
			BuyMdVol:      int64Val(r, "buy_md_vol"),
			BuyMdAmount:   floatVal(r, "buy_md_amount"),
			SellMdVol:     int64Val(r, "sell_md_vol"),
			SellMdAmount:  floatVal(r, "sell_md_amount"),
			BuyLgVol:      int64Val(r, "buy_lg_vol"),
			BuyLgAmount:   floatVal(r, "buy_lg_amount"),
			SellLgVol:     int64Val(r, "sell_lg_vol"),
			SellLgAmount:  floatVal(r, "sell_lg_amount"),
			BuyElgVol:     int64Val(r, "buy_elg_vol"),
			BuyElgAmount:  floatVal(r, "buy_elg_amount"),
			SellElgVol:    int64Val(r, "sell_elg_vol"),
			SellElgAmount: floatVal(r, "sell_elg_amount"),
			NetMfVol:      int64Val(r, "net_mf_vol"),
			NetMfAmount:   floatVal(r, "net_mf_amount"),
		})
	}
	return result, nil
}

// AdjFactorRow represents a row from the adj_factor API.
type AdjFactorRow struct {
	TsCode    string
	TradeDate string
	AdjFactor float64
}

// FetchAdjFactor fetches adjustment factor data from Tushare.
func (s *TushareService) FetchAdjFactor(ctx context.Context, tsCode, startDate, endDate string) ([]AdjFactorRow, error) {
	params := map[string]string{}
	if tsCode != "" {
		params["ts_code"] = tsCode
	}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	resp, err := s.Query(ctx, "adj_factor", params, "")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]AdjFactorRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, AdjFactorRow{
			TsCode:    strVal(r, "ts_code"),
			TradeDate: strVal(r, "trade_date"),
			AdjFactor: floatVal(r, "adj_factor"),
		})
	}
	return result, nil
}

// IndexDailyRow represents a row from the index_daily API.
type IndexDailyRow struct {
	TsCode    string
	TradeDate string
	Open      float64
	High      float64
	Low       float64
	Close     float64
	PreClose  float64
	Change    float64
	PctChg    float64
	Vol       float64
	Amount    float64
}

// FetchIndexDaily fetches index daily data from Tushare.
func (s *TushareService) FetchIndexDaily(ctx context.Context, tsCode, startDate, endDate string) ([]IndexDailyRow, error) {
	params := map[string]string{}
	if tsCode != "" {
		params["ts_code"] = tsCode
	}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	resp, err := s.Query(ctx, "index_daily", params, "")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]IndexDailyRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, IndexDailyRow{
			TsCode:    strVal(r, "ts_code"),
			TradeDate: strVal(r, "trade_date"),
			Open:      floatVal(r, "open"),
			High:      floatVal(r, "high"),
			Low:       floatVal(r, "low"),
			Close:     floatVal(r, "close"),
			PreClose:  floatVal(r, "pre_close"),
			Change:    floatVal(r, "change"),
			PctChg:    floatVal(r, "pct_chg"),
			Vol:       floatVal(r, "vol"),
			Amount:    floatVal(r, "amount"),
		})
	}
	return result, nil
}

// TopListRow represents a row from the top_list (dragon tiger board) API.
type TopListRow struct {
	TradeDate    string
	TsCode       string
	Name         string
	Close        float64
	PctChange    float64
	TurnoverRate float64
	Amount       float64
	LSell        float64
	LBuy         float64
	LAmount      float64
	NetAmount    float64
	NetRate      float64
	AmountRate   float64
	FloatValues  float64
	Reason       string
}

// FetchTopList fetches dragon tiger board data from Tushare.
func (s *TushareService) FetchTopList(ctx context.Context, tradeDate string) ([]TopListRow, error) {
	params := map[string]string{}
	if tradeDate != "" {
		params["trade_date"] = tradeDate
	}

	resp, err := s.Query(ctx, "top_list", params, "")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]TopListRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, TopListRow{
			TradeDate:    strVal(r, "trade_date"),
			TsCode:       strVal(r, "ts_code"),
			Name:         strVal(r, "name"),
			Close:        floatVal(r, "close"),
			PctChange:    floatVal(r, "pct_change"),
			TurnoverRate: floatVal(r, "turnover_rate"),
			Amount:       floatVal(r, "amount"),
			LSell:        floatVal(r, "l_sell"),
			LBuy:         floatVal(r, "l_buy"),
			LAmount:      floatVal(r, "l_amount"),
			NetAmount:    floatVal(r, "net_amount"),
			NetRate:      floatVal(r, "net_rate"),
			AmountRate:   floatVal(r, "amount_rate"),
			FloatValues:  floatVal(r, "float_values"),
			Reason:       strVal(r, "reason"),
		})
	}
	return result, nil
}

// MarginRow represents a row from the margin API.
type MarginRow struct {
	TradeDate string
	ExchangeID string
	Rzye       float64
	Rzmre      float64
	Rzche      float64
	Rqye       float64
	Rqmcl      float64
	Rzrqye     float64
}

// FetchMargin fetches margin trading data from Tushare.
func (s *TushareService) FetchMargin(ctx context.Context, startDate, endDate string) ([]MarginRow, error) {
	params := map[string]string{}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	resp, err := s.Query(ctx, "margin", params, "")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]MarginRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, MarginRow{
			TradeDate:  strVal(r, "trade_date"),
			ExchangeID: strVal(r, "exchange_id"),
			Rzye:       floatVal(r, "rzye"),
			Rzmre:      floatVal(r, "rzmre"),
			Rzche:      floatVal(r, "rzche"),
			Rqye:       floatVal(r, "rqye"),
			Rqmcl:      floatVal(r, "rqmcl"),
			Rzrqye:     floatVal(r, "rzrqye"),
		})
	}
	return result, nil
}

// TradeCalRow represents a row from the trade_cal API.
type TradeCalRow struct {
	Exchange     string
	CalDate      string
	IsOpen       int
	PretradeDate string
}

// FetchTradeCal fetches trading calendar data from Tushare.
func (s *TushareService) FetchTradeCal(ctx context.Context, exchange, startDate, endDate string) ([]TradeCalRow, error) {
	params := map[string]string{}
	if exchange != "" {
		params["exchange"] = exchange
	}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	resp, err := s.Query(ctx, "trade_cal", params, "")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]TradeCalRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, TradeCalRow{
			Exchange:     strVal(r, "exchange"),
			CalDate:      strVal(r, "cal_date"),
			IsOpen:       intVal(r, "is_open"),
			PretradeDate: strVal(r, "pretrade_date"),
		})
	}
	return result, nil
}

// ========== Financial Statement APIs ==========

// IncomeRow represents key fields from the income statement API.
type IncomeRow struct {
	TsCode        string
	AnnDate       string
	EndDate       string
	ReportType    string
	TotalRevenue  float64
	TotalCogs     float64
	OperateProfit float64
	TotalProfit   float64
	NetProfit     float64
	NPParent      float64
	EPS           float64
}

// FetchIncome fetches income statement data from Tushare.
func (s *TushareService) FetchIncome(ctx context.Context, tsCode, startDate, endDate string) ([]IncomeRow, error) {
	params := map[string]string{}
	if tsCode != "" {
		params["ts_code"] = tsCode
	}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	resp, err := s.Query(ctx, "income", params, "ts_code,ann_date,end_date,report_type,total_revenue,total_cogs,operate_profit,total_profit,n_income,n_income_attr_p,diluted_eps")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]IncomeRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, IncomeRow{
			TsCode:        strVal(r, "ts_code"),
			AnnDate:       strVal(r, "ann_date"),
			EndDate:       strVal(r, "end_date"),
			ReportType:    strVal(r, "report_type"),
			TotalRevenue:  floatVal(r, "total_revenue"),
			TotalCogs:     floatVal(r, "total_cogs"),
			OperateProfit: floatVal(r, "operate_profit"),
			TotalProfit:   floatVal(r, "total_profit"),
			NetProfit:     floatVal(r, "n_income"),
			NPParent:      floatVal(r, "n_income_attr_p"),
			EPS:           floatVal(r, "diluted_eps"),
		})
	}
	return result, nil
}

// BalancesheetRow represents key fields from the balance sheet API.
type BalancesheetRow struct {
	TsCode          string
	AnnDate         string
	EndDate         string
	ReportType      string
	TotalAssets     float64
	TotalLiab       float64
	TotalEquity     float64
	TotalHldrEquity float64
	Goodwill        float64
	MonetaryCapital float64
	AccountRecv     float64
	Inventory       float64
}

// FetchBalancesheet fetches balance sheet data from Tushare.
func (s *TushareService) FetchBalancesheet(ctx context.Context, tsCode, startDate, endDate string) ([]BalancesheetRow, error) {
	params := map[string]string{}
	if tsCode != "" {
		params["ts_code"] = tsCode
	}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	resp, err := s.Query(ctx, "balancesheet", params, "ts_code,ann_date,end_date,report_type,total_assets,total_liab,total_hldr_eqy_exc_min_int,total_hldr_eqy_inc_min_int,goodwill,monetary_cap,accounts_receiv,inventory")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]BalancesheetRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, BalancesheetRow{
			TsCode:          strVal(r, "ts_code"),
			AnnDate:         strVal(r, "ann_date"),
			EndDate:         strVal(r, "end_date"),
			ReportType:      strVal(r, "report_type"),
			TotalAssets:     floatVal(r, "total_assets"),
			TotalLiab:       floatVal(r, "total_liab"),
			TotalEquity:     floatVal(r, "total_hldr_eqy_exc_min_int"),
			TotalHldrEquity: floatVal(r, "total_hldr_eqy_inc_min_int"),
			Goodwill:        floatVal(r, "goodwill"),
			MonetaryCapital: floatVal(r, "monetary_cap"),
			AccountRecv:     floatVal(r, "accounts_receiv"),
			Inventory:       floatVal(r, "inventory"),
		})
	}
	return result, nil
}

// CashflowRow represents key fields from the cash flow statement API.
type CashflowRow struct {
	TsCode             string
	AnnDate            string
	EndDate            string
	ReportType         string
	NetOperateCashflow float64
	NetInvestCashflow  float64
	NetFinanceCashflow float64
	CashflowEquivalent float64
}

// FetchCashflow fetches cash flow statement data from Tushare.
func (s *TushareService) FetchCashflow(ctx context.Context, tsCode, startDate, endDate string) ([]CashflowRow, error) {
	params := map[string]string{}
	if tsCode != "" {
		params["ts_code"] = tsCode
	}
	if startDate != "" {
		params["start_date"] = startDate
	}
	if endDate != "" {
		params["end_date"] = endDate
	}

	resp, err := s.Query(ctx, "cashflow", params, "ts_code,ann_date,end_date,report_type,n_cashflow_act,n_cashflow_inv_act,n_cashflow_fnc_act,c_cashflow_equ_end_period")
	if err != nil {
		return nil, err
	}

	rows := parseRows(resp)
	result := make([]CashflowRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, CashflowRow{
			TsCode:             strVal(r, "ts_code"),
			AnnDate:            strVal(r, "ann_date"),
			EndDate:            strVal(r, "end_date"),
			ReportType:         strVal(r, "report_type"),
			NetOperateCashflow: floatVal(r, "n_cashflow_act"),
			NetInvestCashflow:  floatVal(r, "n_cashflow_inv_act"),
			NetFinanceCashflow: floatVal(r, "n_cashflow_fnc_act"),
			CashflowEquivalent: floatVal(r, "c_cashflow_equ_end_period"),
		})
	}
	return result, nil
}

// ========== Helper functions ==========

func int64Val(r Row, key string) int64 {
	v, ok := r[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	default:
		return 0
	}
}

func intVal(r Row, key string) int {
	return int(int64Val(r, key))
}

func strVal(r Row, key string) string {
	v, ok := r[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func floatVal(r Row, key string) float64 {
	v, ok := r[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	default:
		return 0
	}
}
