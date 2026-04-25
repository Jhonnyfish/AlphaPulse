package services

import (
	"context"
	"encoding/json"
	"fmt"
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
