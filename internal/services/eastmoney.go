package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"alphapulse/internal/models"
)

type EastMoneyService struct {
	client *http.Client
}

func NewEastMoneyService(timeout time.Duration) *EastMoneyService {
	return &EastMoneyService{
		client: &http.Client{Timeout: timeout},
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
	if err := s.getJSON(ctx, "https://push2his.eastmoney.com/api/qt/stock/kline/get", params, &response); err != nil {
		return nil, err
	}

	points := make([]models.KlinePoint, 0, len(response.Data.Klines))
	for _, line := range response.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 7 {
			continue
		}
		point, err := parseKlinePoint(parts)
		if err != nil {
			continue
		}
		if err := point.Validate(); err != nil {
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
			Diff []struct {
				Price         float64 `json:"f2"`
				ChangePercent float64 `json:"f3"`
				Change        float64 `json:"f4"`
				Code          string  `json:"f12"`
				Name          string  `json:"f14"`
			} `json:"diff"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, "https://push2.eastmoney.com/api/qt/clist/get", params, &response); err != nil {
		return nil, err
	}

	sectors := make([]models.Sector, 0, len(response.Data.Diff))
	for _, item := range response.Data.Diff {
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
	params.Set("fields", "f2,f3,f4,f5,f6,f7,f12,f14") // price, change%, change, volume, amount, amplitude, code, name

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
				Type int    `json:"SecurityTypeName"`
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

func (s *EastMoneyService) getJSON(ctx context.Context, endpoint string, params url.Values, target interface{}) error {
	requestURL := endpoint
	if encoded := params.Encode(); encoded != "" {
		requestURL = requestURL + "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "AlphaPulse/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("eastmoney request failed: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
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
