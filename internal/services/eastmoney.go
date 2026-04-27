package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	randv2 "math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	apperrors "alphapulse/internal/errors"
	"alphapulse/internal/models"
	"go.uber.org/zap"
)

// eastMoneyCache is an in-memory TTL cache for HTTP response bodies.
type eastMoneyCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	data   []byte
	expiry time.Time
}

// emRateLimiter is a simple token-bucket rate limiter using a ticker.
type emRateLimiter struct {
	ticker *time.Ticker
	ch     chan struct{}
	stop   chan struct{}
}

func newEMRateLimiter(rps int) *emRateLimiter {
	interval := time.Second / time.Duration(rps)
	rl := &emRateLimiter{
		ticker: time.NewTicker(interval),
		ch:     make(chan struct{}, 1),
		stop:   make(chan struct{}),
	}
	// Pre-fill one token so the first request goes through immediately.
	rl.ch <- struct{}{}
	go rl.run()
	return rl
}

func (rl *emRateLimiter) run() {
	for {
		select {
		case <-rl.ticker.C:
			// Non-blocking send: only add a token if the channel isn't full.
			select {
			case rl.ch <- struct{}{}:
			default:
			}
		case <-rl.stop:
			rl.ticker.Stop()
			return
		}
	}
}

// Wait blocks until a rate-limit token is available or ctx is done.
func (rl *emRateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func newEastMoneyCache() *eastMoneyCache {
	c := &eastMoneyCache{
		entries: make(map[string]cacheEntry),
	}
	go c.cleanup()
	return c
}

func (c *eastMoneyCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, e := range c.entries {
			if now.After(e.expiry) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}

func (c *eastMoneyCache) get(key string) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiry) {
		return nil, false
	}
	return e.data, true
}

func (c *eastMoneyCache) set(key string, data []byte, ttl time.Duration) {
	c.mu.Lock()
	c.entries[key] = cacheEntry{data: data, expiry: time.Now().Add(ttl)}
	c.mu.Unlock()
}

// cacheTTLForURL returns the appropriate TTL for a given request URL.
func cacheTTLForURL(u string) time.Duration {
	if strings.Contains(u, "push2his.eastmoney.com") ||
		strings.Contains(u, "push2.eastmoney.com") ||
		strings.Contains(u, "datacenter-web.eastmoney.com") {
		return 60 * time.Second
	}
	return 30 * time.Second
}

type EastMoneyService struct {
	client  *http.Client
	cache   *eastMoneyCache
	limiter *emRateLimiter
}

func NewEastMoneyService(timeout time.Duration) *EastMoneyService {
	return &EastMoneyService{
		client:  &http.Client{Timeout: timeout},
		cache:   newEastMoneyCache(),
		limiter: newEMRateLimiter(2),
	}
}

func (s *EastMoneyService) FetchKline(ctx context.Context, code string, days int) ([]models.KlinePoint, error) {
	params := url.Values{}
	params.Set("secid", EastMoneySecID(code))
	params.Set("klt", "101")
	params.Set("fqt", "1")
	params.Set("lmt", strconv.Itoa(days))
	params.Set("end", "20500101")

	var response struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	err := s.getJSON(ctx, "https://push2his.eastmoney.com/api/qt/stock/kline/get", params, &response)

	points := make([]models.KlinePoint, 0, days)
	if err == nil {
		for _, line := range response.Data.Klines {
			parts := strings.Split(line, ",")
			if len(parts) < 7 {
				continue
			}
			point, pErr := parseKlinePoint(parts)
			if pErr != nil {
				continue
			}
			if vErr := point.Validate(); vErr != nil {
				continue
			}
			points = append(points, point)
		}
	}

	// Fallback to Sina Finance API if EastMoney returned empty or errored
	if len(points) == 0 {
		sinaPoints, sinaErr := s.fetchKlineFromSina(ctx, code, days, 240)
		if sinaErr == nil && len(sinaPoints) > 0 {
			return sinaPoints, nil
		}
		// If Sina also failed, return the original EastMoney error (if any)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("kline data unavailable for %s", code)
	}

	return points, nil
}

// sinaSymbol converts a stock code to Sina Finance symbol format.
// Shanghai: "sh" prefix, Shenzhen: "sz" prefix.
func sinaSymbol(code string) string {
	if IsShanghai(code) {
		return "sh" + code
	}
	return "sz" + code
}

// fetchKlineFromSina fetches kline data from Sina Finance API.
// scale: 5=5min, 15=15min, 30=30min, 60=60min, 240=daily, 1200=weekly, 7200=monthly
func (s *EastMoneyService) fetchKlineFromSina(ctx context.Context, code string, days int, scale int) ([]models.KlinePoint, error) {
	symbol := sinaSymbol(code)
	sinaURL := fmt.Sprintf(
		"https://money.finance.sina.com.cn/quotes_service/api/json_v2.php/CN_MarketData.getKLineData?symbol=%s&scale=%d&ma=no&datalen=%d",
		symbol, scale, days,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sinaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("sina kline request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://finance.sina.com.cn/")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sina kline fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sina kline read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sina kline status %d", resp.StatusCode)
	}

	// Sina returns JSON array directly
	var sinaData []struct {
		Day    string `json:"day"`
		Open   string `json:"open"`
		High   string `json:"high"`
		Low    string `json:"low"`
		Close  string `json:"close"`
		Volume string `json:"volume"`
	}

	if err := json.Unmarshal(body, &sinaData); err != nil {
		return nil, fmt.Errorf("sina kline parse: %w", err)
	}

	points := make([]models.KlinePoint, 0, len(sinaData))
	for _, item := range sinaData {
		open, err := strconv.ParseFloat(item.Open, 64)
		if err != nil {
			continue
		}
		high, err := strconv.ParseFloat(item.High, 64)
		if err != nil {
			continue
		}
		low, err := strconv.ParseFloat(item.Low, 64)
		if err != nil {
			continue
		}
		closePrice, err := strconv.ParseFloat(item.Close, 64)
		if err != nil {
			continue
		}
		volume, err := strconv.ParseFloat(item.Volume, 64)
		if err != nil {
			continue
		}

		point := models.KlinePoint{
			Date:   item.Day,
			Open:   open,
			Close:  closePrice,
			High:   high,
			Low:    low,
			Volume: volume,
			Amount: 0, // Sina API doesn't provide amount
		}
		if vErr := point.Validate(); vErr != nil {
			continue
		}
		points = append(points, point)
	}

	return points, nil
}

func (s *EastMoneyService) FetchSectors(ctx context.Context) ([]models.Sector, error) {
	params := url.Values{}
	params.Set("pn", "1")
	params.Set("pz", "50")
	params.Set("fs", "m:90")
	params.Set("fields", "f2,f3,f4,f12,f14")

	var response struct {
		Data struct {
			Diff json.RawMessage `json:"diff"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/clist/get", params, &response); err != nil {
		return nil, err
	}

	type sectorFields struct {
		Price         float64 `json:"f2"`
		ChangePercent float64 `json:"f3"`
		Change        float64 `json:"f4"`
		Code          string  `json:"f12"`
		Name          string  `json:"f14"`
	}

	var items []sectorFields
	// Try parsing as array first
	if err := json.Unmarshal(response.Data.Diff, &items); err != nil || len(items) == 0 {
		// Try parsing as map (EastMoney sometimes returns diff as {"0": {...}, "1": {...}})
		var m map[string]sectorFields
		if err := json.Unmarshal(response.Data.Diff, &m); err == nil {
			for _, v := range m {
				items = append(items, v)
			}
		}
	}

	sectors := make([]models.Sector, 0, len(items))
	for _, item := range items {
		sector := models.Sector{
			Code:          item.Code,
			Name:          item.Name,
			Price:         item.Price,
			Change:        item.Change,
			ChangePercent: item.ChangePercent,
		}
		if err := sector.Validate(); err != nil {
			continue
		}
		sectors = append(sectors, sector)
	}

	return sectors, nil
}

func (s *EastMoneyService) FetchOverview(ctx context.Context) (models.MarketOverview, error) {
	params := url.Values{}
	params.Set("fltt", "2")
	params.Set("fields", "f1,f2,f3,f4,f12,f14,f104,f105,f106")

	var response struct {
		Data struct {
			Diff []struct {
				Code          string  `json:"f12"`
				Name          string  `json:"f14"`
				Price         float64 `json:"f2"`
				ChangePercent float64 `json:"f3"`
				Change        float64 `json:"f4"`
				AdvanceCount  int     `json:"f104"`
				DeclineCount  int     `json:"f105"`
				FlatCount     int     `json:"f106"`
			} `json:"diff"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/ulist.np/get", params, &response); err != nil {
		return models.MarketOverview{}, err
	}

	overview := models.MarketOverview{
		Indices:   make([]models.OverviewIndex, 0, len(response.Data.Diff)),
		UpdatedAt: time.Now(),
	}
	for index, item := range response.Data.Diff {
		oi := models.OverviewIndex{
			Code:          item.Code,
			Name:          item.Name,
			Price:         item.Price,
			Change:        item.Change,
			ChangePercent: item.ChangePercent,
			AdvanceCount:  item.AdvanceCount,
			DeclineCount:  item.DeclineCount,
			FlatCount:     item.FlatCount,
		}
		if err := oi.Validate(); err != nil {
			continue
		}
		overview.Indices = append(overview.Indices, oi)
		if index == 0 {
			overview.AdvanceCount = item.AdvanceCount
			overview.DeclineCount = item.DeclineCount
			overview.FlatCount = item.FlatCount
		}
	}

	return overview, nil
}

func (s *EastMoneyService) FetchNews(ctx context.Context, limit int) ([]models.NewsItem, error) {
	params := url.Values{}
	params.Set("client", "web")
	params.Set("biz", "web_news_col")
	params.Set("column", "345")
	params.Set("order", "1")
	params.Set("needInteractData", "0")
	params.Set("page_index", "1")
	params.Set("page_size", strconv.Itoa(limit))
	params.Set("fields", "code,title,showtime,source,url,digest")

	var response struct {
		Data struct {
			List []struct {
				Code     string `json:"code"`
				Title    string `json:"title"`
				ShowTime string `json:"showtime"`
				Source   string `json:"source"`
				URL      string `json:"url"`
				Digest   string `json:"digest"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://np-listapi.eastmoney.com/comm/web/getNewsByColumns", params, &response); err != nil {
		return nil, err
	}

	items := make([]models.NewsItem, 0, len(response.Data.List))
	for _, item := range response.Data.List {
		items = append(items, models.NewsItem{
			Code:        item.Code,
			Title:       item.Title,
			Summary:     item.Digest,
			Source:      item.Source,
			URL:         item.URL,
			PublishedAt: parseEastMoneyTime(item.ShowTime),
		})
	}

	return items, nil
}

func (s *EastMoneyService) FetchMoneyFlow(ctx context.Context, code string, days int) ([]models.MoneyFlowDay, error) {
	if days <= 0 {
		days = 10
	}

	params := url.Values{}
	params.Set("secid", EastMoneySecID(code))
	params.Set("klt", "101")
	params.Set("lmt", strconv.Itoa(days))
	params.Set("fields1", "f1,f2,f3,f7")
	params.Set("fields2", "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61")

	var response struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://push2his.eastmoney.com/api/qt/stock/fflow/daykline/get", params, &response); err != nil {
		// Fallback: try push2.eastmoney.com instead of push2his.eastmoney.com
		if fbErr := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/stock/fflow/daykline/get", params, &response); fbErr != nil {
			// Both endpoints unreachable — graceful degradation
			return []models.MoneyFlowDay{}, nil
		}
	}

	// Fallback if primary returned empty
	if len(response.Data.Klines) == 0 {
		if fbErr := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/stock/fflow/daykline/get", params, &response); fbErr != nil || len(response.Data.Klines) == 0 {
			return []models.MoneyFlowDay{}, nil
		}
	}

	flows := make([]models.MoneyFlowDay, 0, len(response.Data.Klines))
	for _, line := range response.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 11 {
			continue
		}
		flows = append(flows, models.MoneyFlowDay{
			Date:         parts[0],
			MainNet:      parseMoneyFlowValue(parts[1]),
			SmallNet:     parseMoneyFlowValue(parts[2]),
			MiddleNet:    parseMoneyFlowValue(parts[3]),
			BigNet:       parseMoneyFlowValue(parts[4]),
			HugeNet:      parseMoneyFlowValue(parts[5]),
			MainNetPct:   parseFloatString(parts[6]),
			SmallNetPct:  parseFloatString(parts[7]),
			MiddleNetPct: parseFloatString(parts[8]),
			BigNetPct:    parseFloatString(parts[9]),
			HugeNetPct:   parseFloatString(parts[10]),
		})
	}

	return flows, nil
}

func (s *EastMoneyService) FetchStockSectors(ctx context.Context, code string) ([]models.StockSector, error) {
	params := url.Values{}
	params.Set("reportName", "RPT_F10_CORETHEME_BOARDTYPE")
	params.Set("columns", "BOARD_NAME,NEW_BOARD_CODE")
	params.Set("filter", fmt.Sprintf(`(SECUCODE="%s")`, eastMoneySecuCode(code)))
	params.Set("pageNumber", "1")
	params.Set("pageSize", "20")
	params.Set("sortTypes", "1")
	params.Set("sortColumns", "BOARD_RANK")
	params.Set("source", "WEB")
	params.Set("client", "WEB")

	var response struct {
		Result struct {
			Data []struct {
				Name string `json:"BOARD_NAME"`
				Code string `json:"NEW_BOARD_CODE"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := s.getJSON(ctx, "https://datacenter-web.eastmoney.com/api/data/v1/get", params, &response); err != nil {
		return nil, err
	}

	sectors := make([]models.StockSector, 0, len(response.Result.Data))
	for _, item := range response.Result.Data {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		sectors = append(sectors, models.StockSector{
			Name: name,
			Code: strings.TrimSpace(item.Code),
		})
	}
	return sectors, nil
}

func (s *EastMoneyService) FetchSectorMembers(ctx context.Context, boardCode string, pageSize int) ([]models.SectorMember, error) {
	boardCode = strings.TrimSpace(boardCode)
	if boardCode == "" {
		return nil, apperrors.BadRequest("board code is required")
	}
	if pageSize <= 0 {
		pageSize = 200
	}
	if pageSize > 500 {
		pageSize = 500
	}

	params := url.Values{}
	params.Set("pn", "1")
	params.Set("pz", strconv.Itoa(pageSize))
	params.Set("po", "1")
	params.Set("np", "1")
	params.Set("fltt", "2")
	params.Set("invt", "2")
	params.Set("fid", "f3")
	params.Set("fs", "b:"+boardCode)
	params.Set("fields", "f2,f3,f12,f14,f9,f23,f22")

	var response struct {
		Data struct {
			Diff json.RawMessage `json:"diff"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/clist/get", params, &response); err != nil {
		return nil, err
	}

	items, err := parseSectorMembersDiff(response.Data.Diff)
	if err != nil {
		return nil, err
	}

	members := make([]models.SectorMember, 0, len(items))
	for _, item := range items {
		code := strings.TrimSpace(item.Code)
		name := strings.TrimSpace(item.Name)
		if code == "" || name == "" {
			continue
		}
		members = append(members, models.SectorMember{
			Name:      name,
			Code:      code,
			ChangePct: item.ChangePct,
			PE:        item.PE,
			PB:        item.PB,
			Amount:    item.Amount,
		})
	}

	return members, nil
}

func (s *EastMoneyService) FetchStockNews(ctx context.Context, code string, limit int) ([]models.NewsItem, error) {
	if limit <= 0 {
		limit = 10
	}

	params := url.Values{}
	params.Set("cb", "jQuery112409538187255818151_1710000000000")
	params.Set("keyword", code)
	params.Set("type", "cmsArticleWebOld")
	params.Set("pageIndex", "1")
	params.Set("pageSize", strconv.Itoa(limit))

	body, err := s.getBody(ctx, "https://search-api-web.eastmoney.com/search/jsonp", params)
	if err != nil {
		return nil, err
	}

	payload, err := parseJSONP(body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result struct {
			CMSArticleWebOld []struct {
				Title       string `json:"title"`
				Content     string `json:"content"`
				Source      string `json:"mediaName"`
				URL         string `json:"url"`
				PublishTime string `json:"date"`
			} `json:"cmsArticleWebOld"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, err
	}

	items := make([]models.NewsItem, 0, len(response.Result.CMSArticleWebOld))
	for _, item := range response.Result.CMSArticleWebOld {
		title := strings.TrimSpace(stripHTML(item.Title))
		if title == "" {
			continue
		}
		items = append(items, models.NewsItem{
			Code:        code,
			Title:       title,
			Summary:     strings.TrimSpace(stripHTML(item.Content)),
			Source:      strings.TrimSpace(item.Source),
			URL:         strings.TrimSpace(item.URL),
			PublishedAt: parseEastMoneyTime(item.PublishTime),
		})
	}

	return items, nil
}

func (s *EastMoneyService) FetchStockAnnouncements(ctx context.Context, code string, limit int) ([]models.Announcement, error) {
	if limit <= 0 {
		limit = 10
	}

	params := url.Values{}
	params.Set("page_size", strconv.Itoa(limit))
	params.Set("page_index", "1")
	params.Set("ann_type", "A")
	params.Set("stock_list", code)

	var response struct {
		Data struct {
			List []struct {
				Title      string `json:"title"`
				NoticeDate string `json:"notice_date"`
				ArtCode    string `json:"art_code"`
				Columns    []struct {
					ShortName string `json:"column_name"`
				} `json:"columns"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://np-anotice-stock.eastmoney.com/api/security/ann", params, &response); err != nil {
		return nil, err
	}

	items := make([]models.Announcement, 0, len(response.Data.List))
	for _, item := range response.Data.List {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		items = append(items, models.Announcement{
			Title:       title,
			Date:        normalizeEastMoneyDate(item.NoticeDate),
			URL:         buildAnnouncementURL(item.ArtCode),
			ArtCode:     strings.TrimSpace(item.ArtCode),
			Source:      "eastmoney",
			PublishedAt: parseEastMoneyTime(item.NoticeDate),
		})
	}

	return items, nil
}

func (s *EastMoneyService) FetchDragonTiger(ctx context.Context) ([]models.DragonTigerItem, error) {
	startDate := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	items, err := s.fetchDragonTigerBoard(ctx, dragonTigerBoardFilter{
		startDate:  startDate,
		exactDate:  "",
		withDetail: true,
	})
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return []models.DragonTigerItem{}, nil
	}

	latestDate := ""
	for _, item := range items {
		if item.TradeDate > latestDate {
			latestDate = item.TradeDate
		}
	}
	if latestDate == "" {
		return items, nil
	}

	filtered := make([]models.DragonTigerItem, 0, len(items))
	for _, item := range items {
		if item.TradeDate == latestDate {
			filtered = append(filtered, item)
		}
	}

	sortDragonTigerItems(filtered)
	return filtered, nil
}

func (s *EastMoneyService) FetchDragonTigerHistory(ctx context.Context, days int) (*models.DragonTigerHistoryResponse, error) {
	if days <= 0 {
		days = 5
	}

	type stockAgg struct {
		name  string
		total float64
		dates map[string]struct{}
	}
	type instAgg struct {
		total float64
		dates map[string]struct{}
	}

	dates := make([]string, 0, days)
	summaries := make([]models.DailySummary, 0, days)
	recurring := make(map[string]*stockAgg)
	institutions := make(map[string]*instAgg)

	for offset := 0; offset < days; offset++ {
		date := time.Now().AddDate(0, 0, -offset).Format("2006-01-02")
		items, err := s.fetchDragonTigerBoard(ctx, dragonTigerBoardFilter{
			exactDate:  date,
			withDetail: true,
		})
		if err != nil {
			zap.L().Error("fetch dragon tiger history failed", zap.Error(err), zap.String("date", date))
			return nil, err
		}
		if len(items) == 0 {
			continue
		}

		sortDragonTigerItems(items)
		dates = append(dates, date)
		summaries = append(summaries, buildDragonTigerDailySummary(date, items))

		for _, item := range items {
			agg, ok := recurring[item.Code]
			if !ok {
				agg = &stockAgg{name: item.Name, dates: make(map[string]struct{})}
				recurring[item.Code] = agg
			}
			agg.name = item.Name
			agg.total += item.NetBuy
			agg.dates[date] = struct{}{}

			for _, dept := range item.Departments {
				name := strings.TrimSpace(dept.Name)
				if name == "" {
					continue
				}
				inst, ok := institutions[name]
				if !ok {
					inst = &instAgg{dates: make(map[string]struct{})}
					institutions[name] = inst
				}
				inst.total += dept.Net
				inst.dates[date] = struct{}{}
			}
		}
	}

	recurringStocks := make([]models.RecurringStock, 0, len(recurring))
	for code, agg := range recurring {
		recurringStocks = append(recurringStocks, models.RecurringStock{
			Code:        code,
			Name:        agg.name,
			Appearances: len(agg.dates),
			TotalNet:    agg.total,
			Dates:       mapKeysSortedDesc(agg.dates),
		})
	}
	sort.Slice(recurringStocks, func(i, j int) bool {
		if recurringStocks[i].Appearances != recurringStocks[j].Appearances {
			return recurringStocks[i].Appearances > recurringStocks[j].Appearances
		}
		if recurringStocks[i].TotalNet != recurringStocks[j].TotalNet {
			return recurringStocks[i].TotalNet > recurringStocks[j].TotalNet
		}
		return recurringStocks[i].Code < recurringStocks[j].Code
	})

	institutionStats := make([]models.InstitutionStat, 0, len(institutions))
	for name, agg := range institutions {
		institutionStats = append(institutionStats, models.InstitutionStat{
			Name:        name,
			Appearances: len(agg.dates),
			TotalNet:    agg.total,
			Dates:       mapKeysSortedDesc(agg.dates),
		})
	}
	sort.Slice(institutionStats, func(i, j int) bool {
		if institutionStats[i].Appearances != institutionStats[j].Appearances {
			return institutionStats[i].Appearances > institutionStats[j].Appearances
		}
		if institutionStats[i].TotalNet != institutionStats[j].TotalNet {
			return institutionStats[i].TotalNet > institutionStats[j].TotalNet
		}
		return institutionStats[i].Name < institutionStats[j].Name
	})

	return &models.DragonTigerHistoryResponse{
		OK:               true,
		Dates:            dates,
		DailySummary:     summaries,
		InstitutionStats: institutionStats,
		RecurringStocks:  recurringStocks,
		Cached:           false,
	}, nil
}

func (s *EastMoneyService) FetchInstitutionTracker(ctx context.Context, days int) ([]models.InstitutionStat, error) {
	history, err := s.FetchDragonTigerHistory(ctx, days)
	if err != nil {
		return nil, err
	}
	return history.InstitutionStats, nil
}

// FetchTopMovers fetches top gaining and losing A-share stocks.
// sort: "asc" for top losers, "desc" for top gainers.
func (s *EastMoneyService) FetchTopMovers(ctx context.Context, sort string, limit int) ([]models.TopMover, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	params := url.Values{}
	params.Set("pn", "1")
	params.Set("pz", strconv.Itoa(limit))
	params.Set("po", "1") // descending
	params.Set("np", "1")
	params.Set("fltt", "2")
	params.Set("invt", "2")
	params.Set("fid", "f3") // sort by change percent
	if sort == "asc" {
		params.Set("po", "0") // ascending for losers
	}
	params.Set("fs", "m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23,m:0+t:81+s:2048") // A-share stocks
	params.Set("fields", "f2,f3,f4,f5,f6,f7,f12,f14")                     // price, change%, change, volume, amount, amplitude, code, name

	var response struct {
		Data struct {
			Diff []struct {
				Price         float64 `json:"f2"`
				ChangePercent float64 `json:"f3"`
				Change        float64 `json:"f4"`
				Volume        float64 `json:"f5"`
				Amount        float64 `json:"f6"`
				Amplitude     float64 `json:"f7"`
				Code          string  `json:"f12"`
				Name          string  `json:"f14"`
			} `json:"diff"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/clist/get", params, &response); err != nil {
		return nil, err
	}

	movers := make([]models.TopMover, 0, len(response.Data.Diff))
	for _, item := range response.Data.Diff {
		m := models.TopMover{
			Code:          item.Code,
			Name:          item.Name,
			Price:         item.Price,
			Change:        item.Change,
			ChangePercent: item.ChangePercent,
			Volume:        item.Volume,
			Amount:        item.Amount,
			Amplitude:     item.Amplitude,
		}
		if err := m.Validate(); err != nil {
			continue
		}
		movers = append(movers, m)
	}

	return movers, nil
}

// SearchSuggestionResult holds search results from the suggest API.
type SearchSuggestionResult struct {
	Code string `json:"Code"`
	Name string `json:"Name"`
}

func (s *EastMoneyService) SearchStocks(ctx context.Context, query string, limit int) ([]models.SearchSuggestion, error) {
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	params := url.Values{}
	params.Set("input", query)
	params.Set("type", "14")
	params.Set("token", "D43BF722C8E33BDC906FB84D85E326E8")
	params.Set("count", strconv.Itoa(limit))

	var response struct {
		QuotationCodeTable struct {
			Data []struct {
				Code string `json:"Code"`
				Name string `json:"Name"`
				Type string `json:"SecurityTypeName"`
			} `json:"Data"`
		} `json:"QuotationCodeTable"`
	}
	if err := s.getJSON(ctx, "https://searchapi.eastmoney.com/api/suggest/get", params, &response); err != nil {
		return nil, err
	}

	suggestions := make([]models.SearchSuggestion, 0, len(response.QuotationCodeTable.Data))
	for _, item := range response.QuotationCodeTable.Data {
		if item.Code == "" || item.Name == "" {
			continue
		}
		// Only include A-share stocks (6-digit codes)
		if len(item.Code) != 6 {
			continue
		}
		suggestions = append(suggestions, models.SearchSuggestion{
			Code: item.Code,
			Name: item.Name,
		})
	}

	return suggestions, nil
}

// FetchHotConcepts fetches top 30 hot concept sectors from EastMoney.
func (s *EastMoneyService) FetchHotConcepts(ctx context.Context) ([]models.HotConcept, error) {
	params := url.Values{}
	params.Set("fid", "f3")
	params.Set("po", "1")
	params.Set("pz", "30")
	params.Set("pn", "1")
	params.Set("np", "1")
	params.Set("fltt", "2")
	params.Set("invt", "2")
	params.Set("fs", "m:90+t:3")
	params.Set("fields", "f2,f3,f4,f12,f14,f104,f105,f128")

	// Use json.RawMessage for diff since EastMoney can return either a map or array
	var response struct {
		Data struct {
			Diff json.RawMessage `json:"diff"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/clist/get", params, &response); err != nil {
		return nil, err
	}

	type conceptFields struct {
		Code        string  `json:"f12"`
		Name        string  `json:"f14"`
		Price       float64 `json:"f2"`
		ChangePct   float64 `json:"f3"`
		Change      float64 `json:"f4"`
		RiseCount   int     `json:"f104"`
		FallCount   int     `json:"f105"`
		LeaderStock string  `json:"f128"`
	}

	var items []conceptFields
	// Try parsing as array first
	if err := json.Unmarshal(response.Data.Diff, &items); err != nil || len(items) == 0 {
		// Try parsing as map
		var m map[string]conceptFields
		if err := json.Unmarshal(response.Data.Diff, &m); err == nil {
			for _, v := range m {
				items = append(items, v)
			}
		}
	}

	concepts := make([]models.HotConcept, 0, len(items))
	for _, item := range items {
		concepts = append(concepts, models.HotConcept{
			Code:        item.Code,
			Name:        item.Name,
			Price:       item.Price,
			ChangePct:   item.ChangePct,
			Change:      item.Change,
			RiseCount:   item.RiseCount,
			FallCount:   item.FallCount,
			LeaderStock: strings.TrimSpace(item.LeaderStock),
		})
	}
	return concepts, nil
}

// FetchMarketBreadth fetches market-wide advance/decline/flat counts from EastMoney.
func (s *EastMoneyService) FetchMarketBreadth(ctx context.Context) (models.MarketBreadth, error) {
	params := url.Values{}
	params.Set("pn", "1")
	params.Set("pz", "1")
	params.Set("po", "1")
	params.Set("np", "1")
	params.Set("fltt", "2")
	params.Set("invt", "2")
	params.Set("fid", "f3")
	params.Set("fs", "m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23")
	params.Set("fields", "f104,f105,f106")

	// Use json.RawMessage for diff since EastMoney can return either a map or array
	var response struct {
		Data struct {
			Diff json.RawMessage `json:"diff"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/clist/get", params, &response); err != nil {
		return models.MarketBreadth{}, err
	}

	type breadthFields struct {
		UpCount   int `json:"f104"`
		DownCount int `json:"f105"`
		FlatCount int `json:"f106"`
	}

	var upCount, downCount, flatCount int
	// Try parsing as array first
	var arr []breadthFields
	if err := json.Unmarshal(response.Data.Diff, &arr); err == nil && len(arr) > 0 {
		for _, item := range arr {
			upCount += item.UpCount
			downCount += item.DownCount
			flatCount += item.FlatCount
		}
	} else {
		// Try parsing as map
		var m map[string]breadthFields
		if err := json.Unmarshal(response.Data.Diff, &m); err == nil {
			for _, item := range m {
				upCount += item.UpCount
				downCount += item.DownCount
				flatCount += item.FlatCount
			}
		}
	}

	total := upCount + downCount + flatCount
	var ratio float64
	var sentiment string
	if total > 0 {
		ratio = float64(upCount) / float64(total)
	} else {
		ratio = 0.5
	}
	switch {
	case ratio >= 0.75:
		sentiment = "极度贪婪"
	case ratio >= 0.58:
		sentiment = "贪婪"
	case ratio >= 0.42:
		sentiment = "中性"
	case ratio >= 0.25:
		sentiment = "恐惧"
	default:
		sentiment = "极度恐惧"
	}

	return models.MarketBreadth{
		UpCount:        upCount,
		DownCount:      downCount,
		FlatCount:      flatCount,
		LimitUp:        0,
		LimitDown:      0,
		Sentiment:      sentiment,
		SentimentRatio: math.Round(ratio*1000) / 10,
	}, nil
}

// ZDFenBuResponse represents the EastMoney 涨跌分布 API response
type ZDFenBuResponse struct {
	Data struct {
		SZJS  int `json:"szjs"` // 上涨家数
		XDJS  int `json:"xdjs"` // 下跌家数
		PGJS  int `json:"pgjs"` // 平盘家数
		ZJS   int `json:"zjs"`  // 炸板家数
		DJS   int `json:"djs"`  // 跌停家数
		Fenbu []struct {
			ZDF string `json:"zdf"` // 涨跌幅范围
			JS  int    `json:"js"`  // 家数
		} `json:"fenbu"`
	} `json:"data"`
}

// QKListResponse represents the EastMoney 涨停/跌停 API response
type QKListResponse struct {
	Data struct {
		ZTCount int `json:"ztCount"` // 涨停家数
		DTCount int `json:"dtCount"` // 跌停家数
		ZT      int `json:"zt"`
		DT      int `json:"dt"`
	} `json:"data"`
}

// SectorFlowResponse represents the EastMoney sector fund flow API response
type SectorFlowResponse struct {
	Data struct {
		Total int `json:"total"`
		Diff  []struct {
			F14 string  `json:"f14"` // sector name
			F62 int64   `json:"f62"` // net fund flow
			F3  float64 `json:"f3"`  // change pct
		} `json:"diff"`
	} `json:"data"`
}

// FetchZDFenBu fetches advance/decline/flat distribution from EastMoney
func (s *EastMoneyService) FetchZDFenBu(ctx context.Context) (upCount, downCount, flatCount int, distribution []models.BreadthDistributionItem, err error) {
	var resp ZDFenBuResponse
	if err = s.getJSON(ctx, "http://push2ex.eastmoney.com/getTopicZDFenBu",
		url.Values{
			"ut":      {"7eea3edcaed734bea9004f1ac72e3b0a"},
			"dession": {"0922e092-2040-4a5f-82f4-b4df49f55b67"},
		}, &resp); err != nil {
		return
	}

	d := resp.Data
	upCount = d.SZJS + d.ZJS
	downCount = d.XDJS + d.DJS
	flatCount = d.PGJS

	// Fallback: if all zero, don't try alternate fields
	for _, item := range d.Fenbu {
		distribution = append(distribution, models.BreadthDistributionItem{
			Range: item.ZDF,
			Count: item.JS,
		})
	}
	return
}

// FetchLimitUpDown fetches limit-up and limit-down counts from EastMoney
func (s *EastMoneyService) FetchLimitUpDown(ctx context.Context) (limitUp, limitDown int, err error) {
	var resp QKListResponse
	if err = s.getJSON(ctx, "http://push2ex.eastmoney.com/getTopicQKList",
		url.Values{
			"ut": {"7eea3edcaed734bea9004f1ac72e3b0a"},
		}, &resp); err != nil {
		return
	}

	limitUp = resp.Data.ZTCount
	if limitUp == 0 {
		limitUp = resp.Data.ZT
	}
	limitDown = resp.Data.DTCount
	if limitDown == 0 {
		limitDown = resp.Data.DT
	}
	return
}

// FetchSectorVolumes fetches sector fund flow data for volume breakdown
func (s *EastMoneyService) FetchSectorVolumes(ctx context.Context) (sectors []models.SectorVolume, volumeToday int64, err error) {
	params := url.Values{}
	params.Set("pn", "1")
	params.Set("pz", "10")
	params.Set("po", "1")
	params.Set("np", "1")
	params.Set("ut", "bd1d9ddb04089700cf9c27f6f7426281")
	params.Set("fltt", "2")
	params.Set("invt", "2")
	params.Set("fid", "f62")
	params.Set("fs", "m:90+t:2")
	params.Set("fields", "f12,f14,f62,f184,f66,f69,f72,f75,f78,f81,f84,f87,f204,f205,f124")

	var resp SectorFlowResponse
	if err = s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/clist/get", params, &resp); err != nil {
		return
	}

	volumeToday = int64(resp.Data.Total)
	sectors = make([]models.SectorVolume, 0, len(resp.Data.Diff))
	for _, item := range resp.Data.Diff {
		sectors = append(sectors, models.SectorVolume{
			Name:      item.F14,
			Volume:    absInt64(item.F62),
			ChangePct: item.F3,
		})
	}
	return
}

// FetchMarketBreadthDetail fetches comprehensive market breadth data matching the Python /api/market-breadth
func (s *EastMoneyService) FetchMarketBreadthDetail(ctx context.Context) (models.MarketBreadthDetail, error) {
	type zdResult struct {
		up, down, flat int
		distribution   []models.BreadthDistributionItem
		err            error
	}
	type qkResult struct {
		limitUp, limitDown int
		err                error
	}
	type volResult struct {
		sectors     []models.SectorVolume
		volumeToday int64
		err         error
	}

	zdCh := make(chan zdResult, 1)
	qkCh := make(chan qkResult, 1)
	volCh := make(chan volResult, 1)

	go func() {
		up, down, flat, dist, err := s.FetchZDFenBu(ctx)
		zdCh <- zdResult{up, down, flat, dist, err}
	}()
	go func() {
		lu, ld, err := s.FetchLimitUpDown(ctx)
		qkCh <- qkResult{lu, ld, err}
	}()
	go func() {
		sectors, vol, err := s.FetchSectorVolumes(ctx)
		volCh <- volResult{sectors, vol, err}
	}()

	zr := <-zdCh
	qr := <-qkCh
	vr := <-volCh

	// Use best-effort: if one fails, use zeros
	upCount, downCount, flatCount := 0, 0, 0
	var distribution []models.BreadthDistributionItem
	if zr.err == nil {
		upCount, downCount, flatCount = zr.up, zr.down, zr.flat
		distribution = zr.distribution
	}

	limitUp, limitDown := 0, 0
	if qr.err == nil {
		limitUp, limitDown = qr.limitUp, qr.limitDown
	}

	var volumeStats models.VolumeStats
	var sectors []models.SectorVolume
	if vr.err == nil {
		sectors = vr.sectors
		for _, sv := range sectors {
			if sv.ChangePct > 0 {
				volumeStats.UpVolume += sv.Volume
			} else if sv.ChangePct < 0 {
				volumeStats.DownVolume += sv.Volume
			} else {
				volumeStats.FlatVolume += sv.Volume
			}
		}
	}

	total := upCount + downCount + flatCount
	adRatio := 0.0
	if downCount > 0 {
		adRatio = round2(float64(upCount) / float64(downCount))
	} else if upCount > 0 {
		adRatio = 99.0
	}

	breadthThrust := 0.0
	if upCount+downCount > 0 {
		breadthThrust = round4(float64(upCount) / float64(upCount+downCount))
	}

	limitRatio := 0.0
	if limitDown > 0 {
		limitRatio = round2(float64(limitUp) / float64(limitDown))
	} else if limitUp > 0 {
		limitRatio = 99.0
	}

	return models.MarketBreadthDetail{
		Advancing:     upCount,
		Declining:     downCount,
		Flat:          flatCount,
		LimitUp:       limitUp,
		LimitDown:     limitDown,
		ADRatio:       adRatio,
		BreadthThrust: breadthThrust,
		LimitRatio:    limitRatio,
		Total:         total,
		VolumeStats:   volumeStats,
		Distribution:  distribution,
		Timestamp:     time.Now().In(marketTZ).Format(time.RFC3339),
	}, nil
}

// FetchMarketSentimentData fetches data needed for market sentiment calculation
func (s *EastMoneyService) FetchMarketSentimentData(ctx context.Context) (models.MarketSentimentResponse, error) {
	type zdResult struct {
		up, down, flat int
		err            error
	}
	type qkResult struct {
		limitUp, limitDown int
		err                error
	}
	type volResult struct {
		sectors     []models.SectorVolume
		volumeToday int64
		err         error
	}

	zdCh := make(chan zdResult, 1)
	qkCh := make(chan qkResult, 1)
	volCh := make(chan volResult, 1)

	go func() {
		up, down, flat, _, err := s.FetchZDFenBu(ctx)
		zdCh <- zdResult{up, down, flat, err}
	}()
	go func() {
		lu, ld, err := s.FetchLimitUpDown(ctx)
		qkCh <- qkResult{lu, ld, err}
	}()
	go func() {
		sectors, vol, err := s.FetchSectorVolumes(ctx)
		volCh <- volResult{sectors, vol, err}
	}()

	zr := <-zdCh
	qr := <-qkCh
	vr := <-volCh

	upCount, downCount, flatCount := 0, 0, 0
	if zr.err == nil {
		upCount, downCount, flatCount = zr.up, zr.down, zr.flat
	}

	limitUp, limitDown := 0, 0
	if qr.err == nil {
		limitUp, limitDown = qr.limitUp, qr.limitDown
	}

	var sectors []models.SectorVolume
	var volumeToday int64
	if vr.err == nil {
		sectors = vr.sectors
		volumeToday = vr.volumeToday
	}

	total := upCount + downCount + flatCount
	volumeRatio := 1.0

	// Calculate Fear/Greed Index (matching Python logic)
	fearGreed, fearGreedLabel := calculateFearGreed(upCount, downCount, flatCount, limitUp, limitDown, volumeRatio)

	// Calculate Temperature
	temperature := calculateTemperature(fearGreed, upCount, downCount, limitUp, limitDown)

	return models.MarketSentimentResponse{
		OK:             true,
		FearGreedIndex: fearGreed,
		FearGreedLabel: fearGreedLabel,
		UpCount:        upCount,
		DownCount:      downCount,
		FlatCount:      flatCount,
		TotalCount:     total,
		LimitUp:        limitUp,
		LimitDown:      limitDown,
		VolumeToday:    volumeToday,
		VolumeAvg5D:    0,
		SectorVolumes:  sectors,
		Temperature:    temperature,
		ServerTime:     time.Now().In(marketTZ).Format(time.RFC3339),
	}, nil
}

// marketTZ is the China Standard Time timezone
var marketTZ = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}()

func calculateFearGreed(upCount, downCount, flatCount, limitUp, limitDown int, volumeRatio float64) (int, string) {
	total := upCount + downCount + flatCount
	if total == 0 {
		return 50, "中性"
	}

	// Metric 1: Up/down ratio (0-100)
	udRatio := float64(upCount) / math.Max(float64(upCount+downCount), 1)
	udScore := udRatio * 100

	// Metric 2: Limit up/down ratio (0-100)
	totalLimit := limitUp + limitDown
	limitScore := 50.0
	if totalLimit > 0 {
		limitScore = float64(limitUp) / float64(totalLimit) * 100
	}

	// Metric 3: Volume ratio (0-100)
	volScore := 50.0
	if volumeRatio > 1.5 {
		if udRatio > 0.5 {
			volScore = 65
		} else {
			volScore = 35
		}
	} else if volumeRatio < 0.5 {
		volScore = 40
	}

	// Weighted average
	score := int(udScore*0.5 + limitScore*0.3 + volScore*0.2)
	score = max(0, min(100, score))

	var label string
	switch {
	case score < 20:
		label = "极度恐慌"
	case score < 40:
		label = "恐慌"
	case score < 60:
		label = "中性"
	case score < 80:
		label = "贪婪"
	default:
		label = "极度贪婪"
	}

	return score, label
}

func calculateTemperature(fearGreed, upCount, downCount, limitUp, limitDown int) int {
	total := upCount + downCount
	if total == 0 {
		return 50
	}
	upRatio := float64(upCount) / float64(total)
	limitActivity := math.Min(float64(limitUp+limitDown)/50, 1.0) * 100
	temp := float64(fearGreed)*0.6 + upRatio*100*0.2 + limitActivity*0.2
	return max(0, min(100, int(temp)))
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
func round4(f float64) float64 { return math.Round(f*10000) / 10000 }

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

type sectorMemberFields struct {
	Code      string  `json:"f12"`
	Name      string  `json:"f14"`
	ChangePct float64 `json:"f3"`
	PE        float64 `json:"f9"`
	PB        float64 `json:"f23"`
	Amount    float64 `json:"f22"`
}

func parseSectorMembersDiff(raw json.RawMessage) ([]sectorMemberFields, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []sectorMemberFields{}, nil
	}

	var list []sectorMemberFields
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}

	var dict map[string]json.RawMessage
	if err := json.Unmarshal(raw, &dict); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(dict))
	for key := range dict {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left, leftErr := strconv.Atoi(keys[i])
		right, rightErr := strconv.Atoi(keys[j])
		if leftErr == nil && rightErr == nil {
			return left < right
		}
		return keys[i] < keys[j]
	})

	list = make([]sectorMemberFields, 0, len(keys))
	for _, key := range keys {
		var item sectorMemberFields
		if err := json.Unmarshal(dict[key], &item); err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, nil
}

func (s *EastMoneyService) HealthCheck(ctx context.Context) error {
	// Lightweight check: fetch a single stock overview to verify API is responsive
	params := url.Values{}
	params.Set("fltt", "2")
	params.Set("fields", "f12,f14")
	params.Set("secids", "1.000001") // Shanghai Composite Index

	var response struct {
		Data struct {
			Diff []struct {
				Code string `json:"f12"`
			} `json:"diff"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/ulist.np/get", params, &response); err != nil {
		return err
	}
	if len(response.Data.Diff) == 0 {
		return fmt.Errorf("eastmoney returned empty data")
	}
	return nil
}

// FetchSectorRotation fetches sector rotation data with breadth, net flow, and strength scores.
func (s *EastMoneyService) FetchSectorRotation(ctx context.Context) ([]models.SectorRotationItem, error) {
	params := url.Values{}
	params.Set("pn", "1")
	params.Set("pz", "50")
	params.Set("po", "1")
	params.Set("np", "1")
	params.Set("fltt", "2")
	params.Set("inft", "2")
	params.Set("fid", "f3")
	params.Set("fs", "m:90+t:2")
	params.Set("fields", "f2,f3,f12,f14,f62,f104,f105")

	var response struct {
		Data struct {
			Diff []struct {
				Price      float64 `json:"f2"`
				ChangePct  float64 `json:"f3"`
				Code       string  `json:"f12"`
				Name       string  `json:"f14"`
				NetFlow    float64 `json:"f62"`
				RisingCount int    `json:"f104"`
				FallingCount int   `json:"f105"`
			} `json:"diff"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/clist/get", params, &response); err != nil {
		return nil, err
	}

	// First pass: find max absolute flow for normalization
	maxAbsFlow := 1.0
	type rawSector struct {
		Code         string
		Name         string
		ChangePct    float64
		Price        float64
		RisingCount  int
		FallingCount int
		BreadthRatio float64
		NetFlow      float64
	}
	rawSectors := make([]rawSector, 0, len(response.Data.Diff))
	for _, row := range response.Data.Diff {
		total := row.RisingCount + row.FallingCount
		breadth := 0.5
		if total > 0 {
			breadth = float64(row.RisingCount) / float64(total)
		}
		if math.Abs(row.NetFlow) > maxAbsFlow {
			maxAbsFlow = math.Abs(row.NetFlow)
		}
		rawSectors = append(rawSectors, rawSector{
			Code:         row.Code,
			Name:         row.Name,
			ChangePct:    row.ChangePct,
			Price:        row.Price,
			RisingCount:  row.RisingCount,
			FallingCount: row.FallingCount,
			BreadthRatio: math.Round(breadth*10000) / 10000,
			NetFlow:      row.NetFlow,
		})
	}

	// Second pass: compute strength scores
	items := make([]models.SectorRotationItem, 0, len(rawSectors))
	for _, s := range rawSectors {
		flowNormalized := 5.0
		if maxAbsFlow > 0 {
			flowNormalized = s.NetFlow/maxAbsFlow*5 + 5
		}
		strengthScore := math.Round((s.ChangePct*0.4+s.BreadthRatio*10*0.3+flowNormalized*0.3)*100) / 100
		items = append(items, models.SectorRotationItem{
			Code:            s.Code,
			Name:            s.Name,
			ChangePct:       s.ChangePct,
			Price:           s.Price,
			RisingCount:     s.RisingCount,
			FallingCount:    s.FallingCount,
			BreadthRatio:    s.BreadthRatio,
			NetFlow:         s.NetFlow,
			StrengthScore:   strengthScore,
			WatchlistMatch:  false,
			WatchlistStocks: []string{},
		})
	}

	// Sort by strength score descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].StrengthScore > items[j].StrengthScore
	})

	return items, nil
}

type dragonTigerBoardFilter struct {
	startDate  string
	exactDate  string
	withDetail bool
}

func (s *EastMoneyService) fetchDragonTigerBoard(ctx context.Context, filter dragonTigerBoardFilter) ([]models.DragonTigerItem, error) {
	params := url.Values{}
	params.Set("sortColumns", "SECURITY_CODE")
	params.Set("sortTypes", "1")
	params.Set("pageSize", "50")
	params.Set("pageNumber", "1")
	params.Set("reportName", "RPT_DAILYBILLBOARD_DETAILSNEW")
	params.Set("columns", "ALL")
	params.Set("source", "WEB")
	params.Set("client", "WEB")
	params.Set("filter", buildDragonTigerFilter(filter.startDate, filter.exactDate))

	var response struct {
		Result struct {
			Data []struct {
				Code         string   `json:"SECURITY_CODE"`
				Name         string   `json:"SECURITY_NAME_ABBR"`
				Close        float64  `json:"CLOSE_PRICE"`
				ChangePct    float64  `json:"CHANGE_RATE"`
				NetBuyAmt    *float64 `json:"BILLBOARD_NET_AMT"`
				NetBuyAmount *float64 `json:"NET_BUY_AMT"`
				BuyAmt       *float64 `json:"BILLBOARD_BUY_AMT"`
				SellAmt      *float64 `json:"BILLBOARD_SELL_AMT"`
				Reason       string   `json:"EXPLANATION"`
				ReasonAlt    string   `json:"EXPLAIN"`
				TradeDate    string   `json:"TRADE_DATE"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := s.getJSON(ctx, "https://datacenter-web.eastmoney.com/api/data/v1/get", params, &response); err != nil {
		return nil, err
	}

	items := make([]models.DragonTigerItem, 0, len(response.Result.Data))
	for _, row := range response.Result.Data {
		code := strings.TrimSpace(row.Code)
		name := strings.TrimSpace(row.Name)
		tradeDate := normalizeEastMoneyDate(row.TradeDate)
		if code == "" || name == "" || tradeDate == "" {
			continue
		}
		if filter.exactDate != "" && tradeDate != filter.exactDate {
			continue
		}

		item := models.DragonTigerItem{
			Code:      code,
			Name:      name,
			Close:     row.Close,
			ChangePct: row.ChangePct,
			NetBuy:    firstFloat(row.NetBuyAmt, row.NetBuyAmount),
			BuyTotal:  firstFloat(row.BuyAmt),
			SellTotal: firstFloat(row.SellAmt),
			Reason:    firstString(row.Reason, row.ReasonAlt),
			TradeDate: tradeDate,
		}
		items = append(items, item)
	}

	if filter.withDetail && len(items) > 0 {
		s.attachDragonTigerDepartments(ctx, items)
	}

	return items, nil
}

func (s *EastMoneyService) attachDragonTigerDepartments(ctx context.Context, items []models.DragonTigerItem) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)

	for idx := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()

			departments, err := s.fetchDragonTigerDepartments(ctx, items[i].Code, items[i].TradeDate)
			if err != nil {
				zap.L().Error("fetch dragon tiger departments failed",
					zap.Error(err),
					zap.String("code", items[i].Code),
					zap.String("date", items[i].TradeDate))
				return
			}
			items[i].Departments = departments
		}(idx)
	}

	wg.Wait()
}

func (s *EastMoneyService) fetchDragonTigerDepartments(ctx context.Context, code, tradeDate string) ([]models.DepartmentDetail, error) {
	buyRows, err := s.fetchDragonTigerDepartmentRows(ctx, code, tradeDate, "RPT_BILLBOARD_DAILYDETAILSBUY", "buy")
	if err != nil {
		return nil, err
	}
	sellRows, err := s.fetchDragonTigerDepartmentRows(ctx, code, tradeDate, "RPT_BILLBOARD_DAILYDETAILSSELL", "sell")
	if err != nil {
		return nil, err
	}

	departments := make([]models.DepartmentDetail, 0, len(buyRows)+len(sellRows))
	departments = append(departments, buyRows...)
	departments = append(departments, sellRows...)
	return departments, nil
}

func (s *EastMoneyService) fetchDragonTigerDepartmentRows(ctx context.Context, code, tradeDate, reportName, side string) ([]models.DepartmentDetail, error) {
	params := url.Values{}
	params.Set("sortColumns", "BUY")
	params.Set("sortTypes", "-1")
	params.Set("pageSize", "20")
	params.Set("pageNumber", "1")
	params.Set("reportName", reportName)
	params.Set("columns", "ALL")
	params.Set("source", "WEB")
	params.Set("client", "WEB")
	params.Set("filter", fmt.Sprintf("(SECURITY_CODE=\"%s\")(TRADE_DATE>='%s')(TRADE_DATE<='%s')", code, tradeDate, tradeDate))

	var response struct {
		Result struct {
			Data []struct {
				Name       string   `json:"OPERATEDEPT_NAME"`
				BuyAmt     *float64 `json:"BUY"`
				BuyAmount  *float64 `json:"BUY_AMT"`
				SellAmt    *float64 `json:"SELL"`
				SellAmount *float64 `json:"SELL_AMT"`
				NetAmt     *float64 `json:"NET"`
				NetAmount  *float64 `json:"NET_AMT"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := s.getJSON(ctx, "https://datacenter-web.eastmoney.com/api/data/v1/get", params, &response); err != nil {
		return nil, err
	}

	departments := make([]models.DepartmentDetail, 0, len(response.Result.Data))
	for _, row := range response.Result.Data {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		buy := firstFloat(row.BuyAmt, row.BuyAmount)
		sell := firstFloat(row.SellAmt, row.SellAmount)
		net := firstFloat(row.NetAmt, row.NetAmount)
		if net == 0 {
			net = buy - sell
		}
		departments = append(departments, models.DepartmentDetail{
			Name: name,
			Buy:  buy,
			Sell: sell,
			Net:  net,
			Side: side,
		})
	}

	sort.Slice(departments, func(i, j int) bool {
		if departments[i].Net != departments[j].Net {
			return departments[i].Net > departments[j].Net
		}
		return departments[i].Name < departments[j].Name
	})
	return departments, nil
}

func buildDragonTigerFilter(startDate, exactDate string) string {
	if exactDate != "" {
		return fmt.Sprintf("(TRADE_DATE>='%s')(TRADE_DATE<='%s')", exactDate, exactDate)
	}
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	}
	return fmt.Sprintf("(TRADE_DATE>='%s')", startDate)
}

func buildDragonTigerDailySummary(date string, items []models.DragonTigerItem) models.DailySummary {
	summary := models.DailySummary{
		Date:       date,
		Count:      len(items),
		TopBuyers:  make([]models.StockBrief, 0, 5),
		TopSellers: make([]models.StockBrief, 0, 5),
	}

	buyers := make([]models.StockBrief, 0, len(items))
	sellers := make([]models.StockBrief, 0, len(items))
	for _, item := range items {
		if item.NetBuy >= 0 {
			summary.TotalNetBuy += item.NetBuy
			buyers = append(buyers, models.StockBrief{Code: item.Code, Name: item.Name, NetBuy: item.NetBuy})
		} else {
			summary.TotalNetSell += math.Abs(item.NetBuy)
			sellers = append(sellers, models.StockBrief{Code: item.Code, Name: item.Name, NetBuy: item.NetBuy})
		}
	}

	sort.Slice(buyers, func(i, j int) bool { return buyers[i].NetBuy > buyers[j].NetBuy })
	sort.Slice(sellers, func(i, j int) bool { return sellers[i].NetBuy < sellers[j].NetBuy })
	if len(buyers) > 5 {
		buyers = buyers[:5]
	}
	if len(sellers) > 5 {
		sellers = sellers[:5]
	}
	summary.TopBuyers = buyers
	summary.TopSellers = sellers
	return summary
}

func sortDragonTigerItems(items []models.DragonTigerItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].NetBuy != items[j].NetBuy {
			return items[i].NetBuy > items[j].NetBuy
		}
		return items[i].Code < items[j].Code
	})
}

func mapKeysSortedDesc(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })
	return keys
}

func firstFloat(values ...*float64) float64 {
	for _, value := range values {
		if value != nil {
			return *value
		}
	}
	return 0
}

func firstString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// getBody fetches the response body for the given endpoint+params.
// It checks the in-memory cache first, then acquires the rate limiter,
// executes the HTTP request with retry+backoff on transient failures,
// and caches the successful result.
func (s *EastMoneyService) getBody(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	requestURL := endpoint
	if encoded := params.Encode(); encoded != "" {
		requestURL = requestURL + "?" + encoded
	}

	// 1. Check cache
	if data, ok := s.cache.get(requestURL); ok {
		return data, nil
	}

	// 2. Rate limit
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	// 3. Execute with retry
	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		body, err, retryable := s.doRequest(ctx, requestURL)
		if err == nil {
			// Success — cache and return
			ttl := cacheTTLForURL(requestURL)
			s.cache.set(requestURL, body, ttl)
			return body, nil
		}

		lastErr = err

		if !retryable || attempt == maxAttempts {
			break
		}

		// Exponential backoff with jitter: 1s, 2s, 4s (±50% jitter)
		base := time.Duration(1<<uint(attempt-1)) * time.Second
		jitter := time.Duration(randv2.Float64()*float64(base)) - base/2 // ±50% of base
		backoff := base + jitter
		if backoff < 100*time.Millisecond {
			backoff = 100 * time.Millisecond
		}

		zap.L().Warn("eastmoney request retry",
			zap.String("url", requestURL),
			zap.Int("attempt", attempt),
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Re-acquire rate limiter before retry
		if err := s.limiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

// doRequest executes a single HTTP GET and returns the body.
// The second return value indicates whether the caller should retry.
func (s *EastMoneyService) doRequest(ctx context.Context, requestURL string) ([]byte, error, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err, false // bad URL, no retry
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "AlphaPulse/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		// Network error — retryable
		return nil, fmt.Errorf("eastmoney request error: %w", err), true
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("eastmoney read body error: %w", err), true
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("eastmoney rate limited: %s", resp.Status), true
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("eastmoney server error: %s", resp.Status), true
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("eastmoney request failed: %s", resp.Status), false
	}

	return body, nil, false
}

// getJSON fetches the response body via getBody (which handles caching,
// rate limiting, and retry) and decodes the JSON into target.
func (s *EastMoneyService) getJSON(ctx context.Context, endpoint string, params url.Values, target interface{}) error {
	body, err := s.getBody(ctx, endpoint, params)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, target); err != nil {
		return err
	}

	return nil
}

func parseKlinePoint(parts []string) (models.KlinePoint, error) {
	open, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return models.KlinePoint{}, err
	}
	closePrice, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return models.KlinePoint{}, err
	}
	high, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return models.KlinePoint{}, err
	}
	low, err := strconv.ParseFloat(parts[4], 64)
	if err != nil {
		return models.KlinePoint{}, err
	}
	volume, err := strconv.ParseFloat(parts[5], 64)
	if err != nil {
		return models.KlinePoint{}, err
	}
	amount, err := strconv.ParseFloat(parts[6], 64)
	if err != nil {
		return models.KlinePoint{}, err
	}

	return models.KlinePoint{
		Date:   parts[0],
		Open:   open,
		Close:  closePrice,
		High:   high,
		Low:    low,
		Volume: volume,
		Amount: amount,
	}, nil
}

func parseEastMoneyTime(value string) time.Time {
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func normalizeEastMoneyDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= len("2006-01-02") {
		if parsed, err := time.Parse("2006-01-02", value[:10]); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	if parsed := parseEastMoneyTime(value); !parsed.IsZero() {
		return parsed.Format("2006-01-02")
	}
	return ""
}

func parseMoneyFlowValue(value string) float64 {
	return parseFloatString(value) / 10000
}

func parseFloatString(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

func eastMoneySecuCode(code string) string {
	if IsShanghai(code) {
		return code + ".SH"
	}
	return code + ".SZ"
}

func parseJSONP(body []byte) ([]byte, error) {
	text := strings.TrimSpace(string(body))
	start := strings.IndexByte(text, '(')
	end := strings.LastIndexByte(text, ')')
	if start < 0 || end <= start {
		return nil, fmt.Errorf("invalid jsonp payload")
	}
	return []byte(strings.TrimSpace(text[start+1 : end])), nil
}

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

func stripHTML(value string) string {
	return htmlTagPattern.ReplaceAllString(value, "")
}

func buildAnnouncementURL(artCode string) string {
	artCode = strings.TrimSpace(artCode)
	if artCode == "" {
		return ""
	}
	return "https://data.eastmoney.com/notices/detail/" + artCode + ".html"
}
