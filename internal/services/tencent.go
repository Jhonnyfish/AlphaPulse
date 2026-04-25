package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"alphapulse/internal/models"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type TencentService struct {
	client *http.Client
}

func NewTencentService(timeout time.Duration) *TencentService {
	return &TencentService{
		client: &http.Client{Timeout: timeout},
	}
}

func (s *TencentService) FetchQuote(ctx context.Context, code string) (models.Quote, error) {
	symbol := TencentSymbol(code)
	requestURL := fmt.Sprintf("https://qt.gtimg.cn/q=%s", symbol)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return models.Quote{}, err
	}
	req.Header.Set("User-Agent", "AlphaPulse/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return models.Quote{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return models.Quote{}, fmt.Errorf("tencent request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Quote{}, err
	}

	// Decode GBK → UTF-8
	decoded, _, _ := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), body)

	quote, err := parseTencentQuote(code, string(decoded))
	if err != nil {
		return models.Quote{}, err
	}
	if err := quote.Validate(); err != nil {
		return models.Quote{}, fmt.Errorf("tencent data validation failed: %w", err)
	}
	return quote, nil
}

func parseTencentQuote(code, payload string) (models.Quote, error) {
	start := strings.Index(payload, "\"")
	end := strings.LastIndex(payload, "\"")
	if start < 0 || end <= start {
		return models.Quote{}, fmt.Errorf("unexpected quote payload")
	}

	fields := strings.Split(payload[start+1:end], "~")
	if len(fields) < 35 {
		fields = strings.Split(payload[start+1:end], "|")
	}
	if len(fields) < 10 {
		return models.Quote{}, fmt.Errorf("quote payload missing fields")
	}

	price, _ := strconv.ParseFloat(fieldValue(fields, 3), 64)
	prevClose, _ := strconv.ParseFloat(fieldValue(fields, 4), 64)
	open, _ := strconv.ParseFloat(fieldValue(fields, 5), 64)
	change, _ := strconv.ParseFloat(fieldValue(fields, 31), 64)
	changePercent, _ := strconv.ParseFloat(fieldValue(fields, 32), 64)
	high, _ := strconv.ParseFloat(fieldValue(fields, 33), 64)
	low, _ := strconv.ParseFloat(fieldValue(fields, 34), 64)

	return models.Quote{
		Code:          code,
		Name:          fieldValue(fields, 1),
		Price:         price,
		Open:          open,
		PrevClose:     prevClose,
		High:          high,
		Low:           low,
		Change:        change,
		ChangePercent: changePercent,
		UpdatedAt:     fieldValue(fields, 30),
	}, nil
}

// FetchIndexQuotes batch-fetches quotes for multiple indices via Tencent API.
// Each code should be in Tencent format: "sh000001", "sz399001", etc.
func (s *TencentService) FetchIndexQuotes(ctx context.Context, indices [][2]string) ([]models.IndexQuote, error) {
	if len(indices) == 0 {
		return nil, nil
	}
	codes := make([]string, len(indices))
	for i, idx := range indices {
		codes[i] = idx[0]
	}
	codesStr := strings.Join(codes, ",")
	requestURL := fmt.Sprintf("https://qt.gtimg.cn/q=%s", codesStr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AlphaPulse/1.0")
	req.Header.Set("Referer", "https://finance.qq.com")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("tencent index request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	decoded, _, _ := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), body)
	codeToName := make(map[string]string, len(indices))
	for _, idx := range indices {
		codeToName[idx[0]] = idx[1]
	}

	var results []models.IndexQuote
	for _, line := range strings.Split(string(decoded), ";") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		start := strings.Index(line, "\"")
		end := strings.LastIndex(line, "\"")
		if start < 0 || end <= start {
			continue
		}
		// Extract code from v_XXX="..." format, strip trailing = if present
		codeEnd := strings.Index(line, "=")
		if codeEnd < 0 {
			codeEnd = start
		}
		fullCode := line[strings.Index(line, "v_")+2 : codeEnd]
		fields := strings.Split(line[start+1:end], "~")
		if len(fields) < 38 || fieldValue(fields, 1) == "" {
			continue
		}
		price, _ := strconv.ParseFloat(fieldValue(fields, 3), 64)
		prevClose, _ := strconv.ParseFloat(fieldValue(fields, 4), 64)
		change, _ := strconv.ParseFloat(fieldValue(fields, 31), 64)
		changePct, _ := strconv.ParseFloat(fieldValue(fields, 32), 64)
		volume, _ := strconv.ParseInt(fieldValue(fields, 36), 10, 64)
		amount, _ := strconv.ParseFloat(fieldValue(fields, 37), 64)

		results = append(results, models.IndexQuote{
			Code:          fullCode,
			Name:          codeToName[fullCode],
			Price:         price,
			PrevClose:     prevClose,
			Change:        change,
			ChangePercent: changePct,
			Volume:        volume,
			Amount:        amount,
		})
	}
	return results, nil
}

func (s *TencentService) HealthCheck(ctx context.Context) error {
	// Lightweight check: fetch Shanghai Composite Index quote
	symbol := "sh000001"
	requestURL := fmt.Sprintf("https://qt.gtimg.cn/q=%s", symbol)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "AlphaPulse/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("tencent health check failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return fmt.Errorf("tencent returned empty response")
	}
	return nil
}

func fieldValue(fields []string, index int) string {
	if index < 0 || index >= len(fields) {
		return ""
	}
	return fields[index]
}
